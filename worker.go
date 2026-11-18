package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
)

type Worker struct {
	cfg         Config
	client      *http.Client
	owner       string
	lastContact atomic.Int64
	busy        atomic.Bool
	completed   atomic.Int64
	failed      atomic.Int64
}

func (w *Worker) step(ctx context.Context) error {
	var response struct {
		Fence int64 `json:"fence"`
		Task  *Task `json:"task"`
	}
	if e := w.call(ctx, w.cfg.Control+"/internal/v1/claim", map[string]string{"owner": w.owner}, &response); e != nil {
		return e
	}
	if response.Task == nil {
		return nil
	}
	w.busy.Store(true)
	defer w.busy.Store(false)
	t := response.Task
	allowed := false
	for _, u := range w.cfg.Nodes {
		if u == t.URL {
			allowed = true
		}
	}
	msg := ""
	if !allowed {
		msg = "target not allowlisted"
	} else {
		if e := w.call(ctx, t.URL+"/internal/v1/task", t, nil); e != nil {
			msg = e.Error()
		}
	}
	if msg != "" {
		slog.Warn("task_failed")
	}
	err := w.call(ctx, w.cfg.Control+"/internal/v1/complete", map[string]any{"id": t.ID, "owner": w.owner, "fence": t.Fence, "error": msg}, nil)
	if err == nil {
		if msg == "" {
			w.completed.Add(1)
		} else {
			w.failed.Add(1)
		}
	}
	return err
}
