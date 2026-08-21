package app

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func findP25Recording(root string, groupID uint32, startedAt time.Time) string {
	marker := "_TO_" + strconv.FormatUint(uint64(groupID), 10)
	newestPath := ""
	newestTime := time.Time{}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if extension != ".wav" && extension != ".mp3" || !strings.Contains(strings.ToUpper(entry.Name()), marker) {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil || info.ModTime().Before(startedAt.Add(-2*time.Second)) {
			return nil
		}
		if newestPath == "" || info.ModTime().After(newestTime) {
			newestPath, newestTime = path, info.ModTime()
		}
		return nil
	})
	return newestPath
}

func archiveP25Recording(dataDirectory, source string, groupID uint32, startedAt time.Time) (string, error) {
	extension := strings.ToLower(filepath.Ext(source))
	if extension != ".wav" && extension != ".mp3" {
		return "", fmt.Errorf("unsupported P25 recording format %q", extension)
	}
	directory := filepath.Join(dataDirectory, "Recordings", startedAt.UTC().Format("2006-01-02"))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	destination := filepath.Join(directory, fmt.Sprintf("%s-p25-tg%d%s", startedAt.UTC().Format("20060102T150405.000Z"), groupID, extension))
	input, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer input.Close()
	temporary := destination + ".part"
	output, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(temporary)
		if copyErr != nil {
			return "", copyErr
		}
		return "", closeErr
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	return destination, nil
}
