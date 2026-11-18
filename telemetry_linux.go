//go:build linux

package main

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var telemetryStarted = time.Now()

// Each deployment unit owns this protocol subset; no shared source imports.
func telemetryFields(filename string) map[string]float64 {
	result := map[string]float64{}
	data, err := os.ReadFile(filename)
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(parts[1], 64)
		if err == nil && value >= 0 {
			if len(parts) > 2 && parts[2] == "kB" {
				value *= 1024
			}
			result[strings.TrimSuffix(parts[0], ":")] = value
		}
	}
	return result
}

func serviceTelemetry(service string, development bool) map[string]any {
	process := map[string]any{"uptime_seconds": time.Since(telemetryStarted).Seconds(), "goroutines": runtime.NumGoroutine()}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	process["heap_bytes"] = mem.HeapAlloc
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) == nil {
		process["cpu_seconds"] = float64(usage.Utime.Sec+usage.Stime.Sec) + float64(usage.Utime.Usec+usage.Stime.Usec)/1e6
	}
	if value, ok := telemetryFields("/proc/self/status")["VmRSS"]; ok {
		process["rss_bytes"] = value
	}
	io := telemetryFields("/proc/self/io")
	for _, name := range []string{"read_bytes", "write_bytes"} {
		if value, ok := io[name]; ok {
			process[name] = value
		}
	}
	host := map[string]any{"cpu_count": runtime.NumCPU()}
	memory := telemetryFields("/proc/meminfo")
	if total, ok := memory["MemTotal"]; ok {
		host["memory_total_bytes"] = total
	}
	if available, ok := memory["MemAvailable"]; ok {
		host["memory_available_bytes"] = available
	}
	if data, err := os.ReadFile("/proc/stat"); err == nil {
		fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
		if len(fields) >= 9 && fields[0] == "cpu" {
			var total, idle float64
			valid := true
			// Guest time is already included in user/nice; never count it twice.
			for i := 1; i <= 8; i++ {
				value, err := strconv.ParseFloat(fields[i], 64)
				if err != nil || value < 0 {
					valid = false
					break
				}
				total += value
				if i == 4 || i == 5 {
					idle += value
				}
			}
			if valid {
				host["cpu_total"] = total
				host["cpu_idle"] = idle
			}
		}
	}
	mode := "production"
	if development {
		mode = "development"
	}
	return map[string]any{"version": 1, "service": service, "mode": mode, "runtime": runtime.Version(), "boot_id": telemetryStarted.Format(time.RFC3339Nano), "sampled_at": time.Now().UTC().Format(time.RFC3339Nano), "process": process, "host": host}
}
