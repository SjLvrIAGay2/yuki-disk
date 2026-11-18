package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (s *Server) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /internal/v1/logs", s.wrap("internal", s.ingestLogs))
	m.HandleFunc("POST /v1/client-logs", s.wrap("log-user", s.clientLogs))
	m.HandleFunc("GET /admin/v1/logs", s.wrap("admin", s.adminLogs))
	m.HandleFunc("GET /admin/v1/log-policy", s.wrap("admin", s.adminLogPolicy))
	m.HandleFunc("POST /admin/v1/log-policy", s.wrap("admin", s.adminLogPolicy))
	m.HandleFunc("GET /admin/v1/node-configurations", s.wrap("admin", s.nodeConfigurations))
	m.HandleFunc("POST /admin/v1/node-configurations/{id}", s.wrap("admin", s.nodeConfigurations))
	m.HandleFunc("POST /internal/v1/node-configuration", s.wrap("internal", s.nodeConfigurationPoll))
	m.HandleFunc("GET /statusz", s.statusDetails)
	m.HandleFunc("GET /v1/status", s.wrap("admin", s.serviceStatus))
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { jsonOut(w, 200, map[string]string{"status": "alive"}) })
	m.HandleFunc("GET /readyz", s.readiness)
	m.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		if !monitorAllowed(w, r, s.cfg.MonitorKey) {
			return
		}
		var b, v, t int
		s.db.QueryRow(r.Context(), "SELECT (SELECT count(*) FROM blocks WHERE state<>'deleted'),(SELECT count(*) FROM versions),(SELECT count(*) FROM tasks WHERE state='failed')").Scan(&b, &v, &t)
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "yuki_blocks "+strconv.Itoa(b)+"\nyuki_versions "+strconv.Itoa(v)+"\nyuki_failed_tasks "+strconv.Itoa(t)+"\n")
	})
	m.HandleFunc("POST /v1/session", s.wrap("", s.login))
	m.HandleFunc("POST /v1/session/refresh", s.wrap("", s.refresh))
	m.HandleFunc("DELETE /v1/session", s.wrap("", s.logout))
	m.HandleFunc("GET /v1/session/me", s.wrap("session", s.me))
	m.HandleFunc("GET /admin/v1/users", s.wrap("admin", s.accountList))
	m.HandleFunc("POST /admin/v1/users", s.wrap("admin", s.accountCreate))
	m.HandleFunc("POST /admin/v1/users/{id}", s.wrap("admin", s.accountUpdate))
	m.HandleFunc("POST /admin/v1/users/{id}/deletion", s.wrap("admin", s.accountDeletion))
	m.HandleFunc("GET /admin/v1/mounts", s.wrap("admin", s.adminMounts))
	m.HandleFunc("POST /admin/v1/mounts", s.wrap("admin", s.adminMounts))
	m.HandleFunc("POST /admin/v1/mounts/{id}", s.wrap("admin", s.adminMountChange))
	m.HandleFunc("GET /admin/v1/exports", s.wrap("admin", s.adminExportBrowse))
	m.HandleFunc("GET /v1/preferences", s.wrap("user", func(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
		v := s.runtime(r.Context())
		jsonOut(w, 200, map[string]int{"user_share_seconds": v.UserShareSeconds, "api_share_seconds": v.APIShareSeconds})
		return nil
	}))
	m.HandleFunc("GET /admin/v1/configuration", s.wrap("admin", s.configuration))
	m.HandleFunc("POST /admin/v1/configuration", s.wrap("admin", s.configuration))
	m.HandleFunc("POST /v2/shares", s.wrap("user", s.createShare))
	m.HandleFunc("GET /v2/shares", s.wrap("user", s.listShares))
	m.HandleFunc("POST /v2/shares/{id}/revoke", s.wrap("user", s.revokeShare))
	m.HandleFunc("POST /public/v2/shares/{id}/access", s.wrap("", s.shareAccess))
	m.HandleFunc("GET /public/v2/shares/{id}/download", s.wrap("", s.sharedDownload("user")))
	m.HandleFunc("GET /public/v2/shares/{id}/blocks/{hash}", s.wrap("", s.sharedDownload("user")))
	m.HandleFunc("GET /admin/v1/applications", s.wrap("admin", s.applicationList))
	m.HandleFunc("POST /admin/v1/applications", s.wrap("admin", s.applicationCreate))
	m.HandleFunc("POST /admin/v1/applications/{id}/revoke", s.wrap("admin", s.applicationRevoke))
	m.HandleFunc("GET /api/v1/files/{id}", s.wrap("", s.sharedDownload("api")))
	m.HandleFunc("GET /api/v1/files/{id}/blocks/{hash}", s.wrap("", s.sharedDownload("api")))
	m.HandleFunc("GET /v1/contract", s.wrap("user", func(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
		jsonOut(w, 200, map[string]any{"version": 1, "chunk_size": 4194304, "replicas": s.cfg.Replicas, "hash": "sha256"})
		return nil
	}))
	m.HandleFunc("POST /v1/uploads", s.wrap("user", s.prepare))
	m.HandleFunc("POST /v1/p2p/{node}", s.wrap("p2p-user", s.p2pTicket))
	m.HandleFunc("POST /v1/uploads/{id}/commit", s.wrap("user", s.commit))
	m.HandleFunc("GET /v1/visibility", s.wrap("user", s.visibility))
	m.HandleFunc("POST /v1/visibility", s.wrap("user", s.visibility))
	m.HandleFunc("GET /public/v1/browse", s.wrap("", legacyPublicGone))
	m.HandleFunc("GET /public/v1/content", s.wrap("", legacyPublicGone))
	m.HandleFunc("POST /internal/v1/content", s.wrap("internal", s.contentPlan))
	m.HandleFunc("GET /v1/mounts", s.wrap("user", s.mountList))
	m.HandleFunc("POST /v1/download/{id}/ranges", s.wrap("user", s.mountRanges))
	m.HandleFunc("GET /v1/files", s.wrap("user", s.files))
	m.HandleFunc("GET /v1/versions", s.wrap("user", s.versions))
	m.HandleFunc("POST /v1/versions/actions", s.wrap("user", s.versionAction))
	m.HandleFunc("GET /v1/browse", s.wrap("user", s.browse))
	m.HandleFunc("POST /v1/folders", s.wrap("user", s.mkdir))
	m.HandleFunc("POST /v1/operations", s.wrap("user", s.operate))
	m.HandleFunc("POST /v1/trash", s.wrap("user", s.trash))
	m.HandleFunc("POST /v1/trash/actions", s.wrap("user", s.trashAction))
	m.HandleFunc("POST /v1/restore", s.wrap("user", s.restore))
	m.HandleFunc("GET /v1/download/{id}", s.wrap("user", s.download))
	m.HandleFunc("GET /v1/download/{id}/blocks/{hash}", s.wrap("user", s.downloadBlock))
	m.HandleFunc("GET /v1/blocks/{hash}", s.wrap("admin", s.block))
	m.HandleFunc("GET /v1/events", s.wrap("admin", s.events))
	m.HandleFunc("GET /v1/tasks", s.wrap("admin", s.tasks))
	m.HandleFunc("POST /internal/v1/upload-proof", s.wrap("internal", s.uploadProof))
	m.HandleFunc("POST /internal/v1/upload-receipt", s.wrap("internal", s.uploadReceipt))
	m.HandleFunc("POST /internal/v1/report", s.wrap("internal", s.report))
	m.HandleFunc("POST /internal/v1/resources", s.wrap("internal", s.resources))
	m.HandleFunc("POST /internal/v1/inventory-page", s.wrap("internal", s.inventoryPage))
	m.HandleFunc("POST /internal/v1/claim", s.wrap("internal", s.claim))
	m.HandleFunc("POST /internal/v1/complete", s.wrap("internal", s.complete))
	m.HandleFunc("POST /internal/v1/permit", s.wrap("internal", s.permit))
	m.HandleFunc("OPTIONS /", s.wrap("", func(w http.ResponseWriter, r *http.Request, b []byte, _ string) error { return nil }))
	return s.mounted(m)
}

// mountedPathKey carries the request path exactly as it arrived on the wire, before
// the control plane's own mount prefix was removed.
type mountedPathKey struct{}

// wirePath returns the path internal callers signed. Internal requests sign
// URL.EscapedPath() as sent, and callers build that URL from the control plane's
// public base, so when the base carries a prefix the signed string includes it
// while the routed path does not. Verification therefore has to consider both.
func wirePath(r *http.Request) string {
	if original, ok := r.Context().Value(mountedPathKey{}).(string); ok {
		return original
	}
	return r.URL.EscapedPath()
}

// mounted makes the router tolerate the control plane's own public path prefix.
//
// Every address the control plane hands out is built as <public base><route>, so
// when the public base carries a prefix (https://host/api) the routes really do
// arrive as /api/internal/v1/report and so on. The prefix is removed here, before
// routing, rather than by the reverse proxy, because internal requests sign the
// path as sent: stripping it at the edge would leave callers and the server
// disagreeing about what was signed. The original path is preserved on the
// request so signature verification can accept the caller's view of it.
func (s *Server) mounted(next http.Handler) http.Handler {
	prefix := s.cfg.publicPrefix()
	if prefix == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		if path != prefix && !strings.HasPrefix(path, prefix+"/") {
			next.ServeHTTP(w, r)
			return
		}
		r2 := r.Clone(context.WithValue(r.Context(), mountedPathKey{}, path))
		if path == prefix {
			// <base> with no route: let the router answer it as the root.
			r2.URL.Path, r2.URL.RawPath = "/", ""
			next.ServeHTTP(w, r2)
			return
		}
		trimmed := strings.TrimPrefix(path, prefix)
		if unescaped, e := url.PathUnescape(trimmed); e == nil {
			r2.URL.Path, r2.URL.RawPath = unescaped, trimmed
		} else {
			r2.URL.Path, r2.URL.RawPath = trimmed, ""
		}
		next.ServeHTTP(w, r2)
	})
}
