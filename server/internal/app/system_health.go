package app

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemHealth is intentionally read-only and best-effort. Missing platform
// counters remain unavailable instead of being represented as fake zeros.
type SystemHealth struct {
	OS              string              `json:"os"`
	Architecture    string              `json:"architecture"`
	Available       bool                `json:"available"`
	CPUCount        int                 `json:"cpuCount"`
	Load1           float64             `json:"load1,omitempty"`
	MemoryTotal     uint64              `json:"memoryTotalBytes,omitempty"`
	MemoryAvailable uint64              `json:"memoryAvailableBytes,omitempty"`
	StorageTotal    uint64              `json:"storageTotalBytes,omitempty"`
	StorageFree     uint64              `json:"storageFreeBytes,omitempty"`
	Temperatures    []SystemTemperature `json:"temperatures,omitempty"`
	GPU             string              `json:"gpu,omitempty"`
	UptimeSeconds   int64               `json:"uptimeSeconds,omitempty"`
	UpdatedAt       time.Time           `json:"updatedAt"`
}

type SystemTemperature struct {
	Name    string  `json:"name"`
	Celsius float64 `json:"celsius"`
}

func readSystemHealth(dataDirectory string) SystemHealth {
	h := SystemHealth{OS: runtime.GOOS, Architecture: runtime.GOARCH, Available: true, CPUCount: runtime.NumCPU(), UpdatedAt: time.Now()}
	if load, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(load))
		if len(fields) > 0 {
			h.Load1, _ = strconv.ParseFloat(fields[0], 64)
		}
	}
	if uptime, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(uptime))
		if len(fields) > 0 {
			value, _ := strconv.ParseFloat(fields[0], 64)
			h.UptimeSeconds = int64(value)
		}
	}
	if mem, err := os.Open("/proc/meminfo"); err == nil {
		defer mem.Close()
		scanner := bufio.NewScanner(mem)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 2 {
				continue
			}
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			switch fields[0] {
			case "MemTotal:":
				h.MemoryTotal = value * 1024
			case "MemAvailable:":
				h.MemoryAvailable = value * 1024
			}
		}
	}
	if total, free, err := filesystemCapacity(dataDirectory); err == nil {
		h.StorageTotal, h.StorageFree = total, free
	}
	for _, path := range []string{"/sys/class/thermal/thermal_zone0/temp", "/sys/class/thermal/thermal_zone1/temp"} {
		if raw, err := os.ReadFile(path); err == nil {
			value, _ := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
			if value > 200 {
				value /= 1000
			}
			if value > 0 {
				h.Temperatures = append(h.Temperatures, SystemTemperature{Name: filepath.Base(filepath.Dir(path)), Celsius: value})
			}
		}
	}
	if _, err := os.Stat("/dev/dri"); err == nil {
		h.GPU = "GPU device available"
	} else if runtime.GOOS == "darwin" {
		h.GPU = "Metal available"
	} else {
		h.GPU = "Not detected"
	}
	return h
}
