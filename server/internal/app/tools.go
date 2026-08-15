package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func firstEnvironment(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

// findTool searches the normal PATH and the private helper directories used by
// packaged GP-SDR builds. This lets release bundles carry small bridge
// executables without changing the user's shell configuration.
func findTool(names ...string) (string, error) {
	var lastErr error
	directories := make([]string, 0, 16)
	configured := strings.TrimSpace(os.Getenv("GPSDR_HELPERS"))
	if configured == "" {
		configured = strings.TrimSpace(os.Getenv("SIGNALHARBOR_HELPERS"))
	}
	if configured != "" {
		directories = append(directories, filepath.SplitList(configured)...)
	}
	if executable, err := os.Executable(); err == nil {
		directory := filepath.Dir(executable)
		directories = append(directories, directory, filepath.Join(directory, "helpers"), filepath.Join(directory, "..", "Resources", "bin"))
	}
	// GUI applications on macOS normally receive only /usr/bin:/bin:/usr/sbin:/sbin,
	// so Homebrew-installed SDR tools are invisible to exec.LookPath. Prefer the
	// native prefix and still allow the alternate prefix for Rosetta installations.
	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			directories = append(directories, "/opt/homebrew/bin", "/opt/homebrew/sbin", "/usr/local/bin", "/usr/local/sbin")
		} else {
			directories = append(directories, "/usr/local/bin", "/usr/local/sbin", "/opt/homebrew/bin", "/opt/homebrew/sbin")
		}
	} else if runtime.GOOS != "windows" {
		directories = append(directories, "/usr/local/bin", "/usr/bin", "/snap/bin")
	}
	for _, directory := range directories {
		for _, name := range names {
			candidates := []string{filepath.Join(directory, name)}
			if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
				candidates = append(candidates, filepath.Join(directory, name+".exe"))
			}
			for _, candidate := range candidates {
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					return candidate, nil
				}
			}
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		} else {
			lastErr = err
		}
	}
	return "", lastErr
}

func findHomebrew() (string, error) {
	return findTool("brew")
}
