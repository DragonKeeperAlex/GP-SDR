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

// removeOrphanedAtomicWriteTemps discards incomplete same-directory snapshots
// left by a process or removable-media interruption. The canonical file is
// never matched, and callers should only invoke it during single-process
// startup before new writers are active.
func removeOrphanedAtomicWriteTemps(directory string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(directory, ".gpsdr-write-*.tmp"))
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, path := range matches {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return removed, err
		}
		// Do not follow or remove a symlink or unexpected directory.
		if !info.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
