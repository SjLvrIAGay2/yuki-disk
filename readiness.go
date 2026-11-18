package main

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) databaseReady(ctx context.Context) bool {
	return s.db.Ping(ctx) == nil
}

func (s *Server) readiness(w http.ResponseWriter, r *http.Request) {
	if !s.databaseReady(r.Context()) {
		jsonOut(w, 503, map[string]string{"status": "database_unavailable"})
		return
	}
	jsonOut(w, 200, map[string]string{"status": "ready"})
}

func (s *Server) statusDetails(w http.ResponseWriter, r *http.Request) {
	if !monitorAllowed(w, r, s.cfg.MonitorKey) {
		return
	}
	// Bound the new database observation so telemetry remains available on failure.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	ready := s.databaseReady(ctx)
	result := serviceTelemetry("control", s.cfg.Development)
	result["ready"] = ready
	jsonOut(w, 200, result)
}
