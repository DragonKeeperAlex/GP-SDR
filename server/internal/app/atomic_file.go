package app

import (
	"errors"
	"os"
	"path/filepath"
)

// Replace only after a complete, synced same-directory write. Unique temporary
// files avoid collisions between callers; filesystem crash recovery still
// depends on the underlying filesystem, especially removable exFAT storage.
func writeBytesAtomic(path string, data []byte) error {
	if path == "" {
		return errors.New("empty persistence path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".gpsdr-write-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}
