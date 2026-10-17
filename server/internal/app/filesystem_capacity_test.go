package app

import (
	"path/filepath"
	"testing"
)

func TestFilesystemCapacityReportsExistingVolumeAndRejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	total, free, err := filesystemCapacity(dir)
	if err != nil || total == 0 || free == 0 || free > total {
		t.Fatalf("invalid volume capacity: total=%d free=%d err=%v", total, free, err)
	}
	if _, _, err := filesystemCapacity(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing storage reported as available")
	}
}
