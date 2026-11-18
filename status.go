package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

func (w *Worker) connected() bool {
	return time.Now().Unix()-w.lastContact.Load() <= 30
}

func (w *Worker) readiness(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	if !w.connected() {
		rw.WriteHeader(503)
		io.WriteString(rw, `{"status":"control_unavailable"}`)
		return
	}
	io.WriteString(rw, `{"status":"ready"}`)
}

func (w *Worker) statusDetails(rw http.ResponseWriter, r *http.Request) {
	if !monitorAllowed(rw, r, w.cfg.MonitorKey) {
		return
	}
	result := serviceTelemetry("maintenance", w.cfg.Development)
	ready := w.connected()
	result["ready"] = ready
	state := "idle"
	if !ready {
		state = "disconnected"
	} else if w.busy.Load() {
		state = "executing"
	}
	result["maintenance"] = map[string]any{"state": state, "last_contact_seconds": w.lastContact.Load(), "completed_tasks": w.completed.Load(), "failed_tasks": w.failed.Load()}
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(result)
}
