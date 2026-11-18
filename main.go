package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	path := flag.String("config", "config.json", "strict JSON config")
	flag.Parse()
	c, e := readConfig(*path)
	if e != nil {
		slog.Error("config", "error", e)
		os.Exit(1)
	}
	n, e := openNode(c)
	if e != nil {
		slog.Error("storage", "error", e)
		os.Exit(1)
	}
	defer n.close()
	closeRuntime := n.startRuntimeOperations()
	defer closeRuntime()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	go func() { defer close(done); n.background(ctx) }()
	resourcesDone := make(chan struct{})
	go func() { defer close(resourcesDone); n.resourceBackground(ctx) }()
	srv := &http.Server{Addr: c.Listen, Handler: n.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 12 * time.Second, WriteTimeout: 12 * time.Second, IdleTimeout: 30 * time.Second}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if srv.Shutdown(sh) != nil {
			srv.Close()
		}
	}()
	if c.TLSCert != "" {
		e = srv.ListenAndServeTLS(c.TLSCert, c.TLSKey)
	} else {
		e = srv.ListenAndServe()
	}
	if e != nil && !errors.Is(e, http.ErrServerClosed) {
		slog.Error("http", "error", e)
		stop()
	}
	<-done
	<-resourcesDone
	<-shutdownDone
}
