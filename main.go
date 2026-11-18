package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	p := flag.String("config", "config.json", "strict JSON config")
	migrationOnly := flag.Bool("migrate-only", false, "migrate metadata and report ownership counts without starting HTTP")
	flag.Parse()
	c, e := loadConfig(*p)
	if e != nil {
		slog.Error("config", "error", e)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	s, e := openStore(ctx, c)
	if e != nil {
		slog.Error("database startup", "error", e)
		os.Exit(1)
	}
	defer s.db.Close()
	if *migrationOnly {
		var paths, versions, uploads, shares int64
		if e = s.db.QueryRow(ctx, "SELECT (SELECT count(*) FROM paths),(SELECT count(*) FROM versions),(SELECT count(*) FROM uploads),(SELECT count(*) FROM visibility)").Scan(&paths, &versions, &uploads, &shares); e != nil {
			slog.Error("migration_report_failed")
			os.Exit(1)
		}
		fmt.Printf("migration complete: paths=%d versions=%d uploads=%d legacy_visibility=%d\n", paths, versions, uploads, shares)
		return
	}
	go s.watchConfiguration(ctx)
	closeLogs := s.startRuntimeLogs()
	defer closeLogs()
	h := &http.Server{Addr: c.Listen, Handler: s.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 25 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16384}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if e := h.Shutdown(shutdown); e != nil {
			h.Close()
		}
	}()
	slog.Info("listening", "address", c.Listen)
	if c.TLSCert != "" {
		e = h.ListenAndServeTLS(c.TLSCert, c.TLSKey)
	} else {
		e = h.ListenAndServe()
	}
	if e != nil && !errors.Is(e, http.ErrServerClosed) {
		slog.Error("server", "error", e)
		os.Exit(1)
	}
	<-shutdownDone
}
