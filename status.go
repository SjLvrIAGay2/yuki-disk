package main

import (
	"net/http"
)

func (n *Node) statusDetails(w http.ResponseWriter, r *http.Request) {
	if !monitorAllowed(w, r, n.cfg.MonitorKey) {
		return
	}
	result := serviceTelemetry("storage", n.cfg.Development)
	result["ready"] = n.isReady()
	if storage, _, e := n.volumeStatus(); e == nil {
		result["storage"] = storage
	}
	read, write := n.network.speeds()
	result["network"] = map[string]any{"received_bytes": n.network.received.Load(), "sent_bytes": n.network.sent.Load(), "read_speed_bps": read, "write_speed_bps": write, "active_reads": n.network.readers.Load(), "active_writes": n.network.writers.Load()}
	output(w, 200, result)
}
