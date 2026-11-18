package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	p := flag.String("config", "config.json", "strict JSON config")
	flag.Parse()
	b, e := os.ReadFile(*p)
	if e != nil {
		slog.Error("config", "error", e)
		os.Exit(1)
	}
	c := Config{Listen: "127.0.0.1:8090"}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&c); e != nil || len(c.Key) < 32 || !validURLPath(c.Control, c.Development, true) || c.MonitorKey != "" && (len(c.MonitorKey) < 32 || c.MonitorKey == c.Key) {
		slog.Error("invalid_config")
		os.Exit(1)
	}
	if d.Decode(&struct{}{}) != io.EOF {
		slog.Error("trailing_config_data")
		os.Exit(1)
	}
	if !c.Development && (strings.Contains(strings.ToLower(c.Key), "development") || strings.Contains(strings.ToLower(c.MonitorKey), "development")) {
		slog.Error("production_rejects_development_credentials")
		os.Exit(1)
	}
	for _, n := range c.Nodes {
		if !validURL(n, c.Development) {
			slog.Error("invalid_node")
			os.Exit(1)
		}
	}
	w := Worker{cfg: c, client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect forbidden") }}, owner: randomID()}
	closeLogs := w.startRuntimeLogs()
	defer closeLogs()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /statusz", w.statusDetails)
	mux.HandleFunc("GET /healthz", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		io.WriteString(rw, `{"status":"alive"}`)
	})
	mux.HandleFunc("GET /readyz", w.readiness)
	mux.HandleFunc("GET /metrics", func(rw http.ResponseWriter, r *http.Request) {
		if !monitorAllowed(rw, r, w.cfg.MonitorKey) {
			return
		}
		rw.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(rw, "yuki_maintenance_last_contact_seconds %d\n", w.lastContact.Load())
	})
	listener, err := net.Listen("tcp", c.Listen)
	if err != nil {
		slog.Error("health_listener", "error", err)
		os.Exit(1)
	}
	health := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second}
	go func() {
		if err := health.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health_server", "error", err)
			stop()
		}
	}()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		health.Shutdown(shutdown)
	}()
	for {
		if e = w.step(ctx); e != nil {
			slog.Warn("maintenance_retry", "error", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
