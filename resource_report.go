package main

import (
	"context"
	"log/slog"
	"time"
)

type StorageStatus struct {
	Total           int64  `json:"total_bytes"`
	Free            int64  `json:"free_bytes"`
	Available       int64  `json:"available_bytes"`
	ReadOnly        bool   `json:"read_only"`
	Mode            string `json:"mode"`
	Configured      int64  `json:"configured_bytes"`
	Stored          int64  `json:"stored_bytes"`
	AllocationUnit  int64  `json:"allocation_unit_bytes"`
	VolumeAvailable int64  `json:"volume_available_bytes"`
}

func (n *Node) volumeStatus() (StorageStatus, string, error) {
	var result StorageStatus
	dir, e := n.root.Open(".")
	if e != nil {
		return result, "", e
	}
	defer dir.Close()
	result, key, e := filesystemStatus(dir, n.cfg)
	if e != nil {
		return result, "", e
	}
	if n.blocks != nil {
		stored, allocated, remaining := n.blocks.Usage()
		result.Stored = stored
		if container, ok := n.blocks.(*containerBlocks); ok {
			result.Total = int64(container.pageCount) * containerPage
			result.Free = result.Total - allocated
			result.Available = min(result.Free, remaining)
			result.AllocationUnit = containerPage
			result.VolumeAvailable = result.Available
			key = container.allocationKey(key)
		} else if n.cfg.CapacityBytes > 0 {
			result.Total = n.cfg.CapacityBytes
			result.Free = max(0, n.cfg.CapacityBytes-allocated)
			result.Available = min(result.Available, result.Free)
		}
	}
	return result, key, nil
}

type ResourceReport struct {
	Version      int           `json:"version"`
	Boot         string        `json:"boot_id"`
	Sequence     int64         `json:"sequence"`
	SampledAt    time.Time     `json:"sampled_at"`
	Volume       string        `json:"volume"`
	Storage      StorageStatus `json:"storage"`
	Received     int64         `json:"received_bytes"`
	Sent         int64         `json:"sent_bytes"`
	ReceiveRate  float64       `json:"receive_bps"`
	SendRate     float64       `json:"send_bps"`
	ReadSpeed    float64       `json:"read_speed_bps"`
	WriteSpeed   float64       `json:"write_speed_bps"`
	ActiveReads  int64         `json:"active_reads"`
	ActiveWrites int64         `json:"active_writes"`
}

func (n *Node) resourceBackground(ctx context.Context) {
	priorAt := time.Now()
	priorIn, priorOut := n.network.received.Load(), n.network.sent.Load()
	var seq int64
	for {
		now := time.Now()
		in, out := n.network.received.Load(), n.network.sent.Load()
		elapsed := now.Sub(priorAt).Seconds()
		rx, tx := 0.0, 0.0
		if elapsed > .1 {
			rx = float64(in-priorIn) / elapsed
			tx = float64(out-priorOut) / elapsed
		}
		volume, key, e := n.volumeStatus()
		seq++
		if e == nil {
			read, write := n.network.speeds()
			_, e = n.signed(ctx, "/internal/v1/resources", ResourceReport{1, n.resourceBoot, seq, now.UTC(), key, volume, in, out, rx, tx, read, write, n.network.readers.Load(), n.network.writers.Load()})
		}
		if e != nil && ctx.Err() == nil {
			slog.Warn("resource_report", "error", e)
		}
		priorAt, priorIn, priorOut = now, in, out
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(n.runtimeValues().ResourceInterval) * time.Second):
		}
	}
}
