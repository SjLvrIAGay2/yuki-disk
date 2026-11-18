package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"time"
)

func (s *Server) startRuntimeLogs() func() {
	buffer := newLogBuffer(func(ctx context.Context, b bufferedBatch) (bufferedReply, error) {
		raw, _ := json.Marshal(b)
		var batch logBatch
		json.Unmarshal(raw, &batch)
		ack, p, e := s.acceptLogs(ctx, "control", "control", batch)
		return bufferedReply{ack, bufferedPolicy{p.Version, p.UploadSeconds, p.MinLevel}}, e
	})
	slog.SetDefault(slog.New(bufferedLogHandler{buffer}))
	finish := buffer.start()
	// Cleanup runs even during quiet periods and never depends on a public read.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			_ = s.logTransaction(ctx, func(t pgx.Tx) error {
				p, e := readLogPolicy(ctx, t)
				if e != nil {
					return e
				}
				return trimRuntimeLogs(ctx, t, p)
			})
		}
	}()
	return func() { cancel(); <-done; finish() }
}
