package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicSnapshotConcurrentReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshots", "data.json")
	var writers sync.WaitGroup
	for i := 0; i < 20; i++ {
		writers.Add(1)
		go func(value int) {
			defer writers.Done()
			if err := writeJSONAtomic(path, map[string]int{"value": value}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	writers.Wait()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]int
	if err = json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatal("temporary files leaked")
	}
}

func TestAtomicSnapshotMarshalFailurePreservesPrevious(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := writeJSONAtomic(path, map[string]int{"value": 42}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := writeJSONAtomic(path, make(chan int)); err == nil {
		t.Fatal("invalid snapshot accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed snapshot replaced previous data")
	}
}
