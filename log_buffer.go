package main

// This unit owns its producer. Only the documented JSON protocol crosses repositories.
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

type bufferedLog struct {
	Sequence int64     `json:"sequence"`
	Level    string    `json:"level"`
	Code     string    `json:"code"`
	Occurred time.Time `json:"occurred_at"`
	Count    int       `json:"count"`
}
type bufferedBatch struct {
	PolicyVersion int64         `json:"policy_version"`
	Stream        string        `json:"stream"`
	Entries       []bufferedLog `json:"entries"`
}
type bufferedPolicy struct {
	Version       int64  `json:"version"`
	UploadSeconds int    `json:"upload_seconds"`
	MinLevel      string `json:"min_level"`
}
type bufferedReply struct {
	Ack    int64          `json:"ack"`
	Policy bufferedPolicy `json:"policy"`
}
type logBuffer struct {
	quietStop   bool
	mu          sync.Mutex
	flushMu     sync.Mutex
	stream      string
	seq, frozen int64
	entries     []bufferedLog
	dropped     int
	policy      bufferedPolicy
	send        func(context.Context, bufferedBatch) (bufferedReply, error)
	override    func() bufferedPolicy
}

func newLogBuffer(send func(context.Context, bufferedBatch) (bufferedReply, error)) *logBuffer {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return &logBuffer{stream: hex.EncodeToString(b), entries: []bufferedLog{}, policy: bufferedPolicy{0, 10, "info"}, send: send}
}
func (l *logBuffer) currentPolicy() bufferedPolicy {
	if l.override != nil {
		return l.override()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.policy
}
func logRank(level string) int {
	switch level {
	case "error":
		return 3
	case "warn":
		return 2
	default:
		return 1
	}
}
func (l *logBuffer) add(level, code string) {
	if logRank(level) < logRank(l.currentPolicy().MinLevel) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.addLocked(level, code, 1)
}
func (l *logBuffer) addLocked(level, code string, count int) {
	if n := len(l.entries); n > 0 {
		last := &l.entries[n-1]
		if last.Sequence > l.frozen && last.Level == level && last.Code == code && last.Count+count <= 1000000 {
			last.Count += count
			return
		}
	}
	if len(l.entries) >= 1000 {
		l.dropped = min(1000000, l.dropped+count)
		return
	}
	l.seq++
	l.entries = append(l.entries, bufferedLog{l.seq, level, code, time.Now().UTC(), count})
}
func (l *logBuffer) flush(ctx context.Context) error {
	l.flushMu.Lock()
	defer l.flushMu.Unlock()
	l.mu.Lock()
	for len(l.entries) > 0 && time.Since(l.entries[0].Occurred) > 6*24*time.Hour {
		l.entries = l.entries[1:]
		l.dropped = min(1000000, l.dropped+1)
	}
	if l.dropped > 0 && len(l.entries) < 1000 {
		count := l.dropped
		l.dropped = 0
		l.addLocked("warn", "logs_dropped", count)
	}
	entries := append([]bufferedLog{}, l.entries[:min(100, len(l.entries))]...)
	if len(entries) > 0 {
		l.frozen = entries[len(entries)-1].Sequence
	}
	maxAck := l.seq
	policyVersion := l.policy.Version
	l.mu.Unlock()
	reply, e := l.send(ctx, bufferedBatch{policyVersion, l.stream, entries})
	if e != nil {
		return e
	}
	if reply.Ack < 0 || reply.Ack > maxAck || (len(entries) > 0 && reply.Ack != entries[len(entries)-1].Sequence) {
		return errors.New("invalid log acknowledgement")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for len(l.entries) > 0 && l.entries[0].Sequence <= reply.Ack {
		l.entries = l.entries[1:]
	}
	if reply.Policy.UploadSeconds >= 2 && reply.Policy.UploadSeconds <= 60 && (reply.Policy.MinLevel == "info" || reply.Policy.MinLevel == "warn" || reply.Policy.MinLevel == "error") {
		l.policy = reply.Policy
	}
	return nil
}
func (l *logBuffer) start() func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		delay := time.Duration(l.currentPolicy().UploadSeconds) * time.Second
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			attempt, stop := context.WithTimeout(ctx, 2*time.Second)
			e := l.flush(attempt)
			stop()
			if e != nil {
				delay = min(time.Minute, max(2*time.Second, delay*2))
			} else {
				delay = time.Duration(l.currentPolicy().UploadSeconds) * time.Second
			}
		}
	}()
	return func() {
		cancel()
		<-done
		if !l.quietStop {
			l.add("info", "stopping")
		}
		ctx, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		for {
			if l.flush(ctx) != nil {
				return
			}
			l.mu.Lock()
			empty := len(l.entries) == 0
			l.mu.Unlock()
			if empty || ctx.Err() != nil {
				return
			}
		}
	}
}

type bufferedLogHandler struct{ buffer *logBuffer }

func (h bufferedLogHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h bufferedLogHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h bufferedLogHandler) WithGroup(string) slog.Handler            { return h }
func (h bufferedLogHandler) Handle(_ context.Context, r slog.Record) error {
	codes := map[string]string{"listening": "started", "started": "started", "request": "request_failed", "server": "request_failed", "http": "request_failed", "health_server": "request_failed", "reconcile": "reconcile_failed", "resource_report": "resource_report_failed", "corrupt_block": "storage_corrupt", "uncommitted_container_descriptor": "storage_corrupt", "maintenance_retry": "connection_failed", "task_failed": "task_failed"}
	code, ok := codes[r.Message]
	if !ok {
		if r.Level < slog.LevelWarn {
			return nil
		}
		code = "request_failed"
	}
	level := "info"
	if r.Level >= slog.LevelError {
		level = "error"
	} else if r.Level >= slog.LevelWarn {
		level = "warn"
	}
	if r.Level >= slog.LevelWarn {
		reason := "unclassified"
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "error" {
				if err, ok := a.Value.Any().(error); ok {
					reason = diagnosticReason(err)
				}
			}
			return true
		})
		if code == "reconcile_failed" && reason != "unclassified" {
			code = "reconcile_" + reason
		}
		fmt.Fprintf(os.Stderr, "{\"level\":\"%s\",\"event\":\"%s\",\"reason\":\"%s\"}\n", level, code, reason)
	}
	h.buffer.add(level, code)
	return nil
}
