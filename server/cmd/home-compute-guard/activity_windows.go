//go:build windows

package main

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")
var lastInput = user32.NewProc("GetLastInputInfo")
var tickCount = syscall.NewLazyDLL("kernel32.dll").NewProc("GetTickCount64")
var foreground = user32.NewProc("GetForegroundWindow")
var windowRect = user32.NewProc("GetWindowRect")
var monitorFromWindow = user32.NewProc("MonitorFromWindow")
var monitorInfo = user32.NewProc("GetMonitorInfoW")
var className = user32.NewProc("GetClassNameW")
var xinput = syscall.NewLazyDLL("xinput1_4.dll").NewProc("XInputGetState")

type rect struct{ Left, Top, Right, Bottom int32 }

func nativeActivity() activity {
	input := struct{ Size, Tick uint32 }{Size: 8}
	ok, _, _ := lastInput.Call(uintptr(unsafe.Pointer(&input)))
	if ok == 0 {
		return activity{Err: errors.New("input state unavailable")}
	}
	now, _, _ := tickCount.Call()
	a := activity{Idle: time.Duration(uint32(now)-input.Tick) * time.Millisecond}
	hwnd, _, _ := foreground.Call()
	if hwnd == 0 {
		a.Err = errors.New("interactive desktop unavailable")
		return a
	}
	var cls [256]uint16
	className.Call(hwnd, uintptr(unsafe.Pointer(&cls[0])), uintptr(len(cls)))
	class := syscall.UTF16ToString(cls[:])
	if class != "Progman" && class != "WorkerW" && class != "Shell_TrayWnd" {
		var wr rect
		info := struct {
			Size          uint32
			Monitor, Work rect
			Flags         uint32
		}{Size: 40}
		m, _, _ := monitorFromWindow.Call(hwnd, 2)
		good, _, _ := windowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
		valid, _, _ := monitorInfo.Call(m, uintptr(unsafe.Pointer(&info)))
		if good == 0 || valid == 0 {
			a.Err = errors.New("foreground state unavailable")
			return a
		}
		a.Fullscreen = wr.Left <= info.Monitor.Left && wr.Top <= info.Monitor.Top && wr.Right >= info.Monitor.Right && wr.Bottom >= info.Monitor.Bottom
	}
	if xinput.Find() == nil {
		for i := uintptr(0); i < 4; i++ {
			state := struct {
				Packet         uint32
				Buttons        uint16
				LT, RT         uint8
				LX, LY, RX, RY int16
			}{}
			result, _, _ := xinput.Call(i, uintptr(unsafe.Pointer(&state)))
			if result == 0 && (state.Buttons != 0 || state.LT > 30 || state.RT > 30 || abs16(state.LX) > 8000 || abs16(state.LY) > 8000 || abs16(state.RX) > 8000 || abs16(state.RY) > 8000) {
				controllerLast = time.Now()
			}
		}
	}
	a.Controller = !controllerLast.IsZero() && time.Since(controllerLast) < 10*time.Minute
	return a
}

var controllerLast time.Time

func abs16(v int16) int32 {
	if v < 0 {
		return -int32(v)
	}
	return int32(v)
}
