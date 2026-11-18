package main

import (
	"context"
	"log/slog"
)

func (w *Worker) startRuntimeLogs() func() {
	logs := newLogBuffer(func(ctx context.Context, b bufferedBatch) (v bufferedReply, e error) {
		e = w.call(ctx, w.cfg.Control+"/internal/v1/logs", b, &v)
		return
	})
	slog.SetDefault(slog.New(bufferedLogHandler{logs}))
	logs.add("info", "started")
	return logs.start()
}
