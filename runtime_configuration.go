package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

type nodeRuntimeValues struct {
	ReadBandwidth    int64  `json:"read_bandwidth_bps"`
	WriteBandwidth   int64  `json:"write_bandwidth_bps"`
	ReserveBytes     int64  `json:"reserve_bytes"`
	ResourceInterval int    `json:"resource_interval_seconds"`
	LogUploadSeconds int    `json:"log_upload_seconds"`
	LogMinLevel      string `json:"log_min_level"`
}
type nodeRuntimeSnapshot struct {
	Version  int64             `json:"version"`
	Settings nodeRuntimeValues `json:"settings"`
}

func (n *Node) runtimeValues() nodeRuntimeValues {
	if v := n.runtimeState.Load(); v != nil {
		return v.Settings
	}
	return nodeRuntimeValues{ResourceInterval: 3, LogUploadSeconds: 10, LogMinLevel: "info"}
}
func (v nodeRuntimeValues) valid() bool {
	return v.ResourceInterval >= 1 && v.ResourceInterval <= 5 && v.LogUploadSeconds >= 2 && v.LogUploadSeconds <= 60 && (v.LogMinLevel == "info" || v.LogMinLevel == "warn" || v.LogMinLevel == "error") && v.ReadBandwidth >= 0 && v.ReadBandwidth <= 1000000000000 && v.WriteBandwidth >= 0 && v.WriteBandwidth <= 1000000000000 && v.ReserveBytes >= 0 && v.ReserveBytes <= 9007199254740991
}
func (n *Node) startRuntimeOperations() func() {
	logs := newLogBuffer(func(ctx context.Context, b bufferedBatch) (v bufferedReply, e error) {
		raw, e := n.signed(ctx, "/internal/v1/logs", b)
		if e == nil {
			e = json.Unmarshal(raw, &v)
		}
		return
	})
	logs.override = func() bufferedPolicy {
		v := n.runtimeValues()
		return bufferedPolicy{0, v.LogUploadSeconds, v.LogMinLevel}
	}
	slog.SetDefault(slog.New(bufferedLogHandler{logs}))
	logs.add("info", "started")
	finish := logs.start()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		state := "applied"
		for {
			var applied int64
			if v := n.runtimeState.Load(); v != nil {
				applied = v.Version
			}
			attempt, stop := context.WithTimeout(ctx, 2*time.Second)
			raw, e := n.signed(attempt, "/internal/v1/node-configuration", map[string]any{"instance_id": logs.stream, "applied_version": applied, "state": state})
			stop()
			if e == nil {
				var target nodeRuntimeSnapshot
				if json.Unmarshal(raw, &target) != nil || !target.Settings.valid() || target.Version < applied {
					state = "failed"
					logs.add("error", "configuration_failed")
				} else {
					if target.Version != applied {
						n.runtimeState.Store(&target)
						logs.add("info", "configuration_applied")
					}
					state = "applied"
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
	return func() { cancel(); <-done; finish() }
}
