//go:build windows

package app

import "os"

func interruptSignal() os.Signal { return os.Interrupt }
