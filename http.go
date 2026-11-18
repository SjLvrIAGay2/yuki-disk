package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func output(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (n *Node) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /statusz", n.statusDetails)
	m.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		if !monitorAllowed(w, r, n.cfg.MonitorKey) {
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		ready := 0
		if n.isReady() {
			ready = 1
		}
		fmt.Fprintf(w, "yuki_node_ready %d\n", ready)
	})
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { output(w, 200, map[string]string{"status": "alive"}) })
	m.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ready := n.isReady()
		code := 200
		if !ready {
			code = 503
		}
		output(w, code, map[string]bool{"ready": ready})
	})
	m.HandleFunc("PUT /v1/blocks/{hash}", func(w http.ResponseWriter, r *http.Request) {
		n.mu.Lock()
		defer n.mu.Unlock()
		if !n.ready {
			output(w, 503, map[string]string{"code": "control_unavailable"})
			return
		}
		p, e := n.capability(r, "put")
		if e != nil {
			output(w, 403, map[string]string{"code": "capability_rejected", "message": e.Error()})
			return
		}
		if e = blockTransferDeadline(w); e != nil {
			output(w, 503, map[string]string{"code": "transfer_unavailable"})
			return
		}
		e = n.put(r.Context(), p, r.Body)
		if e != nil {
			output(w, 409, map[string]string{"code": "write_rejected", "message": e.Error()})
			return
		}
		if e = n.confirmUpload(r.Context(), p); e != nil {
			output(w, 503, map[string]string{"code": "receipt_pending"})
			return
		}
		output(w, 200, map[string]bool{"stored": true})
	})
	m.HandleFunc("GET /v1/blocks/{hash}", n.serveBlock)
	m.HandleFunc("GET /v1/content", n.serveFile)
	m.HandleFunc("GET /v1/content/range", n.serveFile)
	m.HandleFunc("GET /public/v1/content", n.serveFile)
	m.HandleFunc("POST /internal/v1/export", n.exportMetadata)
	m.HandleFunc("POST /internal/v1/task", n.task)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if origin := r.Header.Get("Origin"); origin != "" {
			ok := false
			for _, o := range n.cfg.Origins {
				if o == origin {
					ok = true
				}
			}
			if !ok {
				output(w, 403, map[string]string{"code": "origin_denied"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Content-Range,Accept-Ranges,ETag")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Range,If-Range")
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,OPTIONS")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		m.ServeHTTP(w, r)
	})
}
