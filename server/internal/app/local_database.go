package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type LocalDatabaseStatus struct {
	Folder       string     `json:"folder,omitempty"`
	CanManage    bool       `json:"canManage"`
	Scanning     bool       `json:"scanning"`
	Files        int        `json:"files"`
	Profiles     int        `json:"profiles"`
	Channels     int        `json:"channels"`
	LastScan     *time.Time `json:"lastScan,omitempty"`
	LastError    string     `json:"lastError,omitempty"`
	SkippedFiles int        `json:"skippedFiles"`
}

type localDatabaseConfig struct {
	Folder string `json:"folder"`
}

type LocalDatabaseManager struct {
	mu       sync.RWMutex
	path     string
	profiles *ProfileStore
	status   LocalDatabaseStatus
}

func NewLocalDatabaseManager(dataDirectory string, profiles *ProfileStore) *LocalDatabaseManager {
	manager := &LocalDatabaseManager{path: filepath.Join(dataDirectory, "Data", "local-database.json"), profiles: profiles}
	if data, err := os.ReadFile(manager.path); err == nil {
		var config localDatabaseConfig
		if json.Unmarshal(data, &config) == nil {
			manager.status.Folder = config.Folder
		}
	}
	if manager.status.Folder != "" {
		go manager.Scan()
	}
	return manager
}

func (manager *LocalDatabaseManager) Status() LocalDatabaseStatus {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return manager.status
}

func (manager *LocalDatabaseManager) SetFolder(folder string) (LocalDatabaseStatus, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return LocalDatabaseStatus{}, errors.New("choose a local database folder")
	}
	absolute, err := filepath.Abs(folder)
	if err != nil {
		return LocalDatabaseStatus{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.IsDir() {
		return LocalDatabaseStatus{}, errors.New("the selected local database folder is unavailable")
	}
	data, _ := json.MarshalIndent(localDatabaseConfig{Folder: absolute}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(manager.path), 0o700); err != nil {
		return LocalDatabaseStatus{}, err
	}
	if err := os.WriteFile(manager.path, data, 0o600); err != nil {
		return LocalDatabaseStatus{}, err
	}
	manager.mu.Lock()
	manager.status.Folder = absolute
	manager.status.LastError = ""
	manager.mu.Unlock()
	go manager.Scan()
	return manager.Status(), nil
}

func (manager *LocalDatabaseManager) Scan() LocalDatabaseStatus {
	manager.mu.Lock()
	if manager.status.Scanning {
		status := manager.status
		manager.mu.Unlock()
		return status
	}
	folder := manager.status.Folder
	manager.status.Scanning = true
	manager.status.LastError = ""
	manager.mu.Unlock()

	files, walkErr := localDatabaseFiles(folder)
	profiles, channels, skipped := 0, 0, 0
	errorsFound := make([]string, 0, 4)
	if walkErr == nil {
		for _, path := range files {
			importedProfiles, importedChannels, err := manager.importFile(folder, path)
			if err != nil {
				skipped++
				if len(errorsFound) < 4 {
					errorsFound = append(errorsFound, filepath.Base(path)+": "+err.Error())
				}
				continue
			}
			profiles += importedProfiles
			channels += importedChannels
		}
	}
	now := time.Now()
	manager.mu.Lock()
	manager.status.Scanning = false
	manager.status.Files = len(files)
	manager.status.Profiles = profiles
	manager.status.Channels = channels
	manager.status.SkippedFiles = skipped
	manager.status.LastScan = &now
	if walkErr != nil {
		manager.status.LastError = walkErr.Error()
	} else if len(errorsFound) > 0 {
		manager.status.LastError = strings.Join(errorsFound, " · ")
	}
	status := manager.status
	manager.mu.Unlock()
	return status
}

func localDatabaseFiles(folder string) ([]string, error) {
	if strings.TrimSpace(folder) == "" {
		return nil, errors.New("choose a local database folder")
	}
	files := make([]string, 0, 128)
	err := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if extension == ".csv" || extension == ".tsv" || extension == ".json" {
			files = append(files, path)
			if len(files) > 20_000 {
				return errors.New("local database contains more than 20,000 supported files")
			}
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func (manager *LocalDatabaseManager) importFile(root, path string) (int, int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > 5_000_000 {
		return 0, 0, errors.New("file is not a regular supported file or is larger than 5 MB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return 0, 0, errors.New("file is outside the selected folder")
	}
	if strings.EqualFold(filepath.Ext(path), ".json") {
		var profile ScanProfile
		if err := json.Unmarshal(data, &profile); err != nil {
			return 0, 0, errors.New("JSON is not a GP-SDR profile")
		}
		profile.ID, profile.BuiltIn = localDatabaseProfileID(relative, 0), false
		if strings.TrimSpace(profile.Summary) == "" {
			profile.Summary = "Local database · " + relative
		}
		saved, saveErr := manager.profiles.Save(profile)
		return 1, len(saved.Channels), saveErr
	}
	channels, err := parseChannelCSVWithLimit(data, 100_000)
	if err != nil {
		return 0, 0, err
	}
	saved, err := manager.profiles.saveChannelBanks(path, relative, channels)
	return len(saved), len(channels), err
}

func localDatabaseProfileID(relative string, bank int) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", filepath.ToSlash(strings.ToLower(relative)), bank)))
	return "localdb-" + hex.EncodeToString(digest[:10])
}

func (status LocalDatabaseStatus) Summary() string {
	if status.Scanning {
		return "Scanning local database folder…"
	}
	if status.LastError != "" {
		return status.LastError
	}
	return fmt.Sprintf("%d files · %d profiles · %d channels", status.Files, status.Profiles, status.Channels)
}
