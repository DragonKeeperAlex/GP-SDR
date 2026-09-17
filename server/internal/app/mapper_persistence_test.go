package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMapperJobDiskFailurePreservesState(t *testing.T) {
	root := t.TempDir()
	m := &MapperManager{jobsPath: filepath.Join(root, "jobs.json"), jobs: map[string]MapperJob{}}
	job, err := m.SaveJob(MapperJob{Name: "Original", Config: MapperConfig{Mode: "discovery", DeviceID: "test", StartHz: 98e6, EndHz: 99e6, StepHz: 100e3, DwellMilliseconds: 200}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(m.jobsPath)
	if err != nil {
		t.Fatal(err)
	}
	// An ordinary file as the parent creates a deterministic write failure,
	// even when tests run with elevated filesystem privileges.
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	m.jobsPath = filepath.Join(blocker, "jobs.json")
	edit := job
	edit.Name = "Changed"
	if _, err := m.SaveJob(edit); err == nil {
		t.Fatal("failed save reported success")
	}
	if got, _ := m.Job(job.ID); got.Name != "Original" {
		t.Fatal("failed save changed job")
	}
	if err := m.DeleteJob(job.ID); err == nil {
		t.Fatal("failed delete reported success")
	}
	if _, ok := m.Job(job.ID); !ok {
		t.Fatal("failed delete removed job")
	}
	if _, err := m.SaveJob(MapperJob{Config: job.Config}); err == nil {
		t.Fatal("failed create reported success")
	}
	if len(m.Jobs()) != 1 {
		t.Fatal("failed create left phantom job")
	}
	after, err := os.ReadFile(filepath.Join(root, "jobs.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("previous disk snapshot changed")
	}
}

func TestMapperBackgroundPersistenceErrorVisible(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	m := &MapperManager{recordsPath: filepath.Join(blocker, "results.json"), jobsPath: filepath.Join(blocker, "jobs.json"), jobs: map[string]MapperJob{}, records: map[string]MapperFrequencyRecord{}}
	m.persistRecords()
	if m.Status().LastError == "" {
		t.Fatal("results write failure hidden")
	}
	m.lastError = ""
	m.persistJobs()
	if m.Status().LastError == "" {
		t.Fatal("job state write failure hidden")
	}
}
