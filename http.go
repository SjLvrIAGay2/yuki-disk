package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http"
	"unicode/utf8"
)

type handler func(http.ResponseWriter, *http.Request, []byte, string) error

func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func parse(b []byte, v any) error {
	if !utf8.Valid(b) {
		return fail(400, "invalid_json", "JSON must be valid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return fail(400, "invalid_json", e.Error())
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return fail(400, "invalid_json", "trailing JSON")
	}
	return nil
}

func (s *Server) wrap(auth string, f handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		origin := r.Header.Get("Origin")
		if origin != "" {
			ok := false
			for _, o := range s.cfg.Origins {
				if origin == o {
					ok = true
				}
			}
			if !ok {
				jsonOut(w, 403, map[string]string{"code": "origin_denied"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,Range,If-Range,X-Yuki-CSRF")
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,POST,DELETE,OPTIONS")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		var e error
		var id string
		body, er := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
		if er != nil {
			e = fail(413, "body_too_large", "metadata body exceeds limit")
		} else if auth == "internal" {
			id, er = s.internal(r, body)
			if er != nil {
				e = fail(401, "unauthorized", er.Error())
			}
		}
		if e == nil && (auth == "user" || auth == "admin" || auth == "session" || auth == "log-user" || auth == "p2p-user") {
			response := &committedResponse{header: http.Header{}, status: 200}
			var authorized principal
			transaction := s.tx
			role := auth
			if auth == "p2p-user" {
				role = "user"
			}
			if auth == "log-user" {
				transaction = s.logTransaction
				role = "user"
			}
			e = transaction(r.Context(), func(t pgx.Tx) error {
				ctx := context.WithValue(r.Context(), requestTransactionKey{}, t)
				if auth == "log-user" {
					ctx = context.WithValue(ctx, logTransactionKey{}, t)
				}
				p, err := s.authenticate(ctx, t, r)
				if err != nil {
					return err
				}
				if auth != "session" && p.Role != role {
					return fail(403, "role_denied", "role not allowed")
				}
				if auth == "p2p-user" {
					authorized = p
					return nil
				}
				if auth == "user" {
					if _, err = t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", p.ID); err != nil {
						return err
					}
				}
				ctx = context.WithValue(ctx, principalKey{}, p)
				if snapshot := s.runtimeState.Load(); snapshot != nil {
					ctx = context.WithValue(ctx, runtimeContextKey{}, *snapshot)
				}
				return f(response, r.WithContext(ctx), body, p.ID)
			})
			if e == nil && auth == "p2p-user" {
				// The broker exchanges metadata only. Never hold PG locks across its
				// network request. Revalidate the same session before releasing a ticket.
				e = f(response, r, body, authorized.ID)
				if e == nil {
					e = s.tx(r.Context(), func(t pgx.Tx) error {
						p, err := s.authenticate(r.Context(), t, r)
						if err != nil {
							return err
						}
						if p.ID != authorized.ID || p.Role != "user" {
							return fail(403, "role_denied", "role not allowed")
						}
						return nil
					})
				}
			}
			if e == nil {
				for k, v := range response.header {
					w.Header()[k] = v
				}
				w.WriteHeader(response.status)
				_, e = w.Write(response.Bytes())
			}
		} else if e == nil {
			e = f(w, r, body, id)
		}
		if e != nil {
			var a *apiError
			if errors.As(e, &a) {
				if a.Status == 409 || a.Status == 503 {
					w.Header().Set("Retry-After", "2")
				}
				jsonOut(w, a.Status, map[string]string{"code": a.Code, "message": a.Message})
			} else {
				slog.Error("request", "path", r.URL.Path, "error", e)
				jsonOut(w, 503, map[string]string{"code": "unavailable", "message": "operation unavailable; retry safely"})
			}
		}
	}
}

// Metadata is released only after the authoritative transaction commits.
type committedResponse struct {
	bytes.Buffer
	header http.Header
	status int
}

func (w *committedResponse) Header() http.Header    { return w.header }
func (w *committedResponse) WriteHeader(status int) { w.status = status }
