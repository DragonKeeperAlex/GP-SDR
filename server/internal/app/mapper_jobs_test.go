package app

import (
	"math"
	"strings"
	"testing"
	"time"
)

func mapperTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	runtimeState, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	limit := 20e6
	runtimeState.mu.Lock()
	runtimeState.devices = []SDRDevice{
		{ID: "simulator-0", Name: "Test receiver A", Kind: "Simulator", Driver: "built-in", Connected: true, Available: true, SampleRateLimit: &limit},
		{ID: "simulator-1", Name: "Test receiver B", Kind: "Simulator", Driver: "built-in", Connected: true, Available: true, SampleRateLimit: &limit},
	}
	runtimeState.mu.Unlock()
	t.Cleanup(func() {
		runtimeState.StopAllMapperJobs()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			runtimeState.mu.RLock()
			active := len(runtimeState.mapperJobs)
			runtimeState.mu.RUnlock()
			if active == 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		runtimeState.Stop()
	})
	return runtimeState
}

func saveMapperTestJob(t *testing.T, runtimeState *Runtime, name, deviceID, mode string, startHz, endHz float64) MapperJob {
	t.Helper()
	job, err := runtimeState.SaveMapperJob(MapperJob{Name: name, Config: MapperConfig{
		Mode: mode, DeviceID: deviceID, StartHz: startHz, EndHz: endHz, StepHz: 100_000,
		DwellMilliseconds: 200, DecipherListenSeconds: 5, IdentifyMinimumHits: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func waitForMapperJob(t *testing.T, runtimeState *Runtime, id string, predicate func(MapperJob) bool) MapperJob {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if job, ok := runtimeState.mapper.Job(id); ok && predicate(job) {
			return job
		}
		time.Sleep(20 * time.Millisecond)
	}
	job, _ := runtimeState.mapper.Job(id)
	t.Fatalf("Mapper job %s did not reach expected state: %+v", id, job)
	return MapperJob{}
}

func TestMapperRunsIndependentConcurrentReceiverJobs(t *testing.T) {
	runtimeState := mapperTestRuntime(t)
	jobA := saveMapperTestJob(t, runtimeState, "VHF discovery", "simulator-0", "discovery", 150e6, 150.2e6)
	jobB := saveMapperTestJob(t, runtimeState, "UHF discovery", "simulator-1", "discovery", 450e6, 450.2e6)
	if _, err := runtimeState.StartMapperJob(jobA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeState.StartMapperJob(jobB.ID); err != nil {
		t.Fatal(err)
	}
	progressA := waitForMapperJob(t, runtimeState, jobA.ID, func(job MapperJob) bool { return job.Progress.ChecksCompleted > 0 })
	progressB := waitForMapperJob(t, runtimeState, jobB.ID, func(job MapperJob) bool { return job.Progress.ChecksCompleted > 0 })
	if progressA.Progress.CurrentFrequencyHz < 150e6 || progressA.Progress.CurrentFrequencyHz > 150.2e6 {
		t.Fatalf("job A escaped its assigned range: %+v", progressA.Progress)
	}
	if progressB.Progress.CurrentFrequencyHz < 450e6 || progressB.Progress.CurrentFrequencyHz > 450.2e6 {
		t.Fatalf("job B escaped its assigned range: %+v", progressB.Progress)
	}
	if progressA.Progress.MonitoredChannels != 3 || progressA.Progress.TotalBatches != 1 {
		t.Fatalf("nearby Discovery targets were not processed as one simultaneous batch: %+v", progressA.Progress)
	}
	runtimeState.StopMapperJob(jobA.ID)
	waitForMapperJob(t, runtimeState, jobA.ID, func(job MapperJob) bool { return !job.Progress.Running })
	if job, _ := runtimeState.mapper.Job(jobB.ID); !job.Progress.Running {
		t.Fatal("stopping one receiver job stopped the other")
	}
}

func TestMapperRejectsTwoJobsOnSameReceiver(t *testing.T) {
	runtimeState := mapperTestRuntime(t)
	jobA := saveMapperTestJob(t, runtimeState, "First", "simulator-0", "discovery", 150e6, 150.1e6)
	jobB := saveMapperTestJob(t, runtimeState, "Second", "simulator-0", "discovery", 450e6, 450.1e6)
	if _, err := runtimeState.StartMapperJob(jobA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeState.StartMapperJob(jobB.ID); err == nil || !strings.Contains(err.Error(), "already assigned") {
		t.Fatalf("expected clear receiver ownership error, got %v", err)
	}
}

func TestMapperDoesNotStealReceiverFromTuner(t *testing.T) {
	runtimeState := mapperTestRuntime(t)
	if err := runtimeState.Tune(TunerRequest{DeviceID: "simulator-0", FrequencyHz: 100.1e6, Mode: "wfm", BandwidthHz: 180e3, IQGain: 1}); err != nil {
		t.Fatal(err)
	}
	job := saveMapperTestJob(t, runtimeState, "Mapper", "simulator-0", "discovery", 100e6, 100.2e6)
	if _, err := runtimeState.StartMapperJob(job.ID); err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("expected Tuner receiver ownership error, got %v", err)
	}
}

func TestMapperRunsDiscoveryAndDecipherTogether(t *testing.T) {
	runtimeState := mapperTestRuntime(t)
	runtimeState.mapper.Observe(162_550_000, true, -25, -80, "NFM", "Analog FM", "NOAA Weather", "")
	discovery := saveMapperTestJob(t, runtimeState, "Discovery", "simulator-0", "discovery", 460e6, 460.2e6)
	decipher := saveMapperTestJob(t, runtimeState, "Decipher", "simulator-1", "decipher", 162e6, 163e6)
	if _, err := runtimeState.StartMapperJob(discovery.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeState.StartMapperJob(decipher.ID); err != nil {
		t.Fatal(err)
	}
	waitForMapperJob(t, runtimeState, discovery.ID, func(job MapperJob) bool { return job.Progress.ChecksCompleted > 0 })
	result := waitForMapperJob(t, runtimeState, decipher.ID, func(job MapperJob) bool { return job.Progress.ChecksCompleted > 0 })
	if result.Progress.Mode != "decipher" || result.Progress.CurrentFrequencyHz != 162_550_000 {
		t.Fatalf("unexpected decipher progress: %+v", result.Progress)
	}
}

func TestMapperRecordsReceiverAndJobProvenance(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), jobs: make(map[string]MapperJob)}
	manager.ObserveJob("job-a", "receiver-a", MapperConfig{}, 462_675_000, true, -30, -80, "NFM", "Analog FM", "GMRS", "")
	manager.ObserveJob("job-b", "receiver-b", MapperConfig{}, 462_675_000, true, -28, -81, "NFM", "Analog FM", "GMRS", "")
	record := manager.Status().Records[0]
	if len(record.JobIDs) != 2 || len(record.DeviceIDs) != 2 {
		t.Fatalf("expected merged provenance from both jobs: %+v", record)
	}
}

func TestIdentifyCountsHitsWithoutPromotingDiscoveryOneOffs(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), jobs: make(map[string]MapperJob)}
	manager.ObserveJob("discovery", "receiver-a", MapperConfig{Mode: "discovery"}, 462_675_000, true, -30, -80, "NFM", "Analog FM", "GMRS", "")
	manager.ObserveJob("identify", "receiver-b", MapperConfig{Mode: "decipher"}, 462_675_000, true, -28, -81, "NFM", "Analog FM", "GMRS", "")
	record := manager.Status().Records[0]
	if record.Hits != 2 || record.DiscoveryHits != 1 || record.IdentifyHits != 1 || record.DiscoveryChecks != 1 || record.IdentifyChecks != 1 {
		t.Fatalf("expected combined and per-workflow hit history: %+v", record)
	}
	job := MapperJob{Config: MapperConfig{Mode: "decipher", IdentifyMinimumHits: 2, IdentifyHitSource: "discovery"}}
	if _, err := mapperJobTargets(job, []MapperFrequencyRecord{record}); err == nil {
		t.Fatal("one Discovery hit should remain ineligible even after an Identify hit")
	}
	job.Config.IdentifyHitSource = "combined"
	if targets, err := mapperJobTargets(job, []MapperFrequencyRecord{record}); err != nil || len(targets) != 1 {
		t.Fatalf("combined history should include both hit sources: targets=%+v error=%v", targets, err)
	}
}

func TestIdentifyFiltersBySuccessfulCheckPercentageAndLimit(t *testing.T) {
	now := time.Now()
	records := []MapperFrequencyRecord{
		{FrequencyHz: 150e6, LastSeen: now, Hits: 8, Checks: 10, DiscoveryHits: 8, DiscoveryChecks: 10},
		{FrequencyHz: 151e6, LastSeen: now, Hits: 3, Checks: 10, DiscoveryHits: 3, DiscoveryChecks: 10},
		{FrequencyHz: 152e6, LastSeen: now, Hits: 9, Checks: 10, DiscoveryHits: 9, DiscoveryChecks: 10},
	}
	job := MapperJob{Config: MapperConfig{Mode: "decipher", IdentifyMinimumHits: 2, IdentifyHitSource: "discovery", IdentifyMinimumOccupancy: .5, IdentifyMaximumChannels: 1, IdentifyOrder: "occupancy"}}
	targets, err := mapperJobTargets(job, records)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].FrequencyHz != 152e6 {
		t.Fatalf("expected only the highest successful-check channel, got %+v", targets)
	}
}

func TestMapperGroupsNearbyTargetsWithinConfiguredLimit(t *testing.T) {
	limit := 20e6
	device := SDRDevice{ID: "hackrf-test", Kind: "HackRF", Connected: true, Available: true, SampleRateLimit: &limit}
	targets := []surveyTarget{
		{FrequencyHz: 450_000_000, BandwidthHz: 12_500, Mode: "nfm"},
		{FrequencyHz: 450_025_000, BandwidthHz: 12_500, Mode: "nfm"},
		{FrequencyHz: 450_050_000, BandwidthHz: 12_500, Mode: "nfm"},
		{FrequencyHz: 450_075_000, BandwidthHz: 12_500, Mode: "nfm"},
		{FrequencyHz: 450_100_000, BandwidthHz: 12_500, Mode: "nfm"},
	}
	job := MapperJob{Config: MapperConfig{Mode: "discovery", ConcurrentChannels: 4, SampleRateHz: 8_000_000}}
	batches := mapperJobTargetBatches(job, device, targets)
	if len(batches) != 2 || len(batches[0].Targets) != 4 || len(batches[1].Targets) != 1 {
		t.Fatalf("expected a four-channel batch and one remainder, got %+v", batches)
	}
	if batches[0].SampleRate != 8_000_000 {
		t.Fatalf("expected configured sample rate, got %d", batches[0].SampleRate)
	}
}

func TestMapperSplitsTargetsOutsideInstantaneousBandwidth(t *testing.T) {
	limit := 3.2e6
	device := SDRDevice{ID: "rtl-test", Kind: "RTL-SDR", Connected: true, Available: true, SampleRateLimit: &limit}
	targets := []surveyTarget{
		{FrequencyHz: 150_000_000, BandwidthHz: 12_500, Mode: "nfm"},
		{FrequencyHz: 151_500_000, BandwidthHz: 12_500, Mode: "nfm"},
	}
	job := MapperJob{Config: MapperConfig{Mode: "decipher", ConcurrentChannels: 8, SampleRateHz: 1_000_000}}
	batches := mapperJobTargetBatches(job, device, targets)
	if len(batches) != 2 {
		t.Fatalf("targets outside a 1 MHz capture must be split, got %+v", batches)
	}
}

func TestMapperBatchCenterKeepsChannelsInUsableWindowAndOffDC(t *testing.T) {
	device := SDRDevice{ID: "hackrf-test", Kind: "HackRF", Connected: true, Available: true}
	batch := surveyTargetBatch{SampleRate: 8_000_000, Targets: []surveyTarget{
		{FrequencyHz: 450_000_000, BandwidthHz: 12_500},
		{FrequencyHz: 450_025_000, BandwidthHz: 12_500},
		{FrequencyHz: 450_050_000, BandwidthHz: 12_500},
	}}
	spec, ok := mapperBatchCaptureSpec(device, batch)
	if !ok {
		t.Fatal("nearby channels should fit in one HackRF capture")
	}
	for _, target := range batch.Targets {
		offset := target.FrequencyHz - float64(spec.CenterFrequencyHz)
		if abs := math.Abs(offset); abs > float64(spec.SampleRateHz)*.42 {
			t.Fatalf("target %.0f is outside usable sample window: offset %.0f", target.FrequencyHz, offset)
		} else if abs < 50_000 {
			t.Fatalf("target %.0f was placed too close to the DC spike: offset %.0f", target.FrequencyHz, offset)
		}
	}
}
