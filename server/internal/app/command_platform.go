package app

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func platformCommand(executable string, arguments ...string) *exec.Cmd {
	extension := strings.ToLower(filepath.Ext(executable))
	if runtime.GOOS == "windows" && (extension == ".bat" || extension == ".cmd") {
		commandArguments := append([]string{"/d", "/c", executable}, arguments...)
		return exec.Command("cmd.exe", commandArguments...)
	}
	return exec.Command(executable, arguments...)
}
