//go:build windows || darwin

package main

import (
	"context"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"os"
	"runtime"
	"time"
)

var telemetryStarted = time.Now()

// Each unit owns its wire projection. Unavailable counters are omitted.
func serviceTelemetry(service string, development bool) map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	p := map[string]any{"uptime_seconds": time.Since(telemetryStarted).Seconds(), "goroutines": runtime.NumGoroutine()}
	var heap runtime.MemStats
	runtime.ReadMemStats(&heap)
	p["heap_bytes"] = heap.HeapAlloc
	if self, err := process.NewProcessWithContext(ctx, int32(os.Getpid())); err == nil {
		if memory, err := self.MemoryInfoWithContext(ctx); err == nil {
			p["rss_bytes"] = memory.RSS
		}
		if times, err := self.TimesWithContext(ctx); err == nil {
			p["cpu_seconds"] = times.User + times.System
		}
		if counters, err := self.IOCountersWithContext(ctx); err == nil {
			p["read_bytes"] = counters.ReadBytes
			p["write_bytes"] = counters.WriteBytes
		}
	}
	host := map[string]any{"cpu_count": runtime.NumCPU()}
	if memory, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		host["memory_total_bytes"] = memory.Total
		host["memory_available_bytes"] = memory.Available
	}
	if times, err := cpu.TimesWithContext(ctx, false); err == nil && len(times) == 1 {
		v := times[0]
		host["cpu_total"] = v.User + v.System + v.Idle + v.Nice + v.Iowait + v.Irq + v.Softirq + v.Steal
		host["cpu_idle"] = v.Idle + v.Iowait
	}
	mode := "production"
	if development {
		mode = "development"
	}
	return map[string]any{"version": 1, "service": service, "mode": mode, "runtime": runtime.Version(), "boot_id": telemetryStarted.Format(time.RFC3339Nano), "sampled_at": time.Now().UTC().Format(time.RFC3339Nano), "process": p, "host": host}
}
