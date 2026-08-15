//go:build !windows

package app

import (
	"os"
	"syscall"
)

func interruptSignal() os.Signal { return syscall.SIGTERM }
