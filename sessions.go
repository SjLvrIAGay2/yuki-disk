package main

import (
	"context"
	"encoding/hex"
	"github.com/jackc/pgx/v5"
	"net"
	"net/http"
	"strings"
	"time"
)

const refreshCookie = "yuki_refresh"
const statusRefreshCookie = "yuki_status_refresh"

type principal struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
type principalKey struct{}

func (s *Server) authenticate(ctx context.Context, t pgx.Tx, r *http.Request) (principal, error) {
	var token Session
	var p principal
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || decodeToken(s.cfg.SessionKey, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), &token) != nil || token.Version != 1 || (token.Purpose != "session" && token.Purpose != "status-session") || token.Expires <= time.Now().Unix() {
		return p, fail(401, "unauthorized", "session required")
	}
	if token.Purpose == "status-session" && (r.Method != "GET" || (r.URL.Path != "/v1/status" && r.URL.Path != "/v1/events" && r.URL.Path != "/v1/session/me")) {
		return p, fail(403, "scope_denied", "read-only status session required")
	}
	e := t.QueryRow(ctx, `SELECT a.id,a.username,a.role FROM sessions s JOIN accounts a ON a.id=s.user_id WHERE s.id=$1 AND (s.client='status')=$2 AND s.revoked_at IS NULL AND s.expires_at>now() AND s.idle_until>now() AND NOT a.disabled AND a.auth_epoch=s.auth_epoch FOR SHARE OF a,s`, token.ID, token.Purpose == "status-session").Scan(&p.ID, &p.Username, &p.Role)
	if e == pgx.ErrNoRows {
		return p, fail(401, "unauthorized", "session expired or revoked")
	}
	return p, e
}
func (s *Server) cookie(w http.ResponseWriter, value string, expiry time.Time) {
	s.clientCookie(w, value, expiry, "web")
}
func cookieName(client string) string {
	if client == "status" {
		return statusRefreshCookie
	}
	return refreshCookie
}
func (s *Server) clientCookie(w http.ResponseWriter, value string, expiry time.Time, client string) {
	// The refresh cookie is scoped to the session routes under the control
	// plane's public path prefix, which is empty when it serves at its origin
	// root and "/api" (for example) when published behind a reverse proxy.
	c := &http.Cookie{Name: cookieName(client), Value: value, Path: s.cfg.publicPrefix() + "/v1/session", HttpOnly: true, Secure: !s.cfg.Development, SameSite: http.SameSiteStrictMode, Expires: expiry}
	if value == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}
func (s *Server) csrf(r *http.Request) error {
	if r.Header.Get("Origin") == "" || r.Header.Get("X-Yuki-CSRF") != "1" {
		return fail(403, "csrf_required", "browser origin and CSRF header required")
	}
	return nil
}
func (s *Server) sessionResponse(w http.ResponseWriter, id string, p principal, client string) {
	purpose := "session"
	if client == "status" {
		purpose = "status-session"
	}
	jsonOut(w, 200, map[string]any{"token": encodeToken(s.cfg.SessionKey, Session{1, purpose, time.Now().Add(5 * time.Minute).Unix(), id}), "expires_in": 300, "user": p})
}
func (s *Server) loginLimit(r *http.Request) error {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	key := digest([]byte(host))
	var n int
	e := s.db.QueryRow(r.Context(), `INSERT INTO auth_attempts(key,count,expires_at) VALUES($1,1,now()+interval '1 minute') ON CONFLICT(key) DO UPDATE SET count=CASE WHEN auth_attempts.expires_at<=now() THEN 1 ELSE auth_attempts.count+1 END,expires_at=CASE WHEN auth_attempts.expires_at<=now() THEN now()+interval '1 minute' ELSE auth_attempts.expires_at END RETURNING count`, key).Scan(&n)
	if e != nil {
		return e
	}
	if n > s.runtime(r.Context()).LoginAttempts {
		return fail(429, "rate_limited", "try again later")
	}
	return nil
}
func (s *Server) login(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Client   string `json:"client"`
	}
	if e := parse(b, &req); e != nil {
		return e
	}
	if req.Client == "" {
		req.Client = "cli"
	}
	if req.Client != "cli" && req.Client != "web" && req.Client != "status" {
		return fail(400, "invalid_client", "invalid client")
	}
	if req.Client != "cli" {
		if e := s.csrf(r); e != nil {
			return e
		}
	}
	if e := s.loginLimit(r); e != nil {
		return e
	}
	if len(req.Password) > 1024 {
		return fail(401, "unauthorized", "invalid credentials")
	}
	if req.Username == "" {
		req.Username = s.cfg.LegacyUsername
	}
	var p principal
	var hash string
	var epoch int64
	var disabled bool
	e := s.db.QueryRow(r.Context(), "SELECT id,username,role,password_hash,auth_epoch,disabled FROM accounts WHERE username=$1", req.Username).Scan(&p.ID, &p.Username, &p.Role, &hash, &epoch, &disabled)
	if e != nil && e != pgx.ErrNoRows {
		return e
	}
	if e == pgx.ErrNoRows {
		hash = "pbkdf2-sha256$600000$invalid-account$" + strings.Repeat("0", 64)
	}
	if !passwordMatches(req.Password, hash) || disabled || p.ID == "" {
		return fail(401, "unauthorized", "invalid credentials")
	}
	if req.Client == "cli" && p.Role != "user" {
		return fail(403, "user_required", "ordinary user required")
	}
	if req.Client == "status" && p.Role != "admin" {
		return fail(403, "administrator_required", "administrator required")
	}
	id, refresh := randomID(), randomID()
	expiry := time.Now().Add(30 * 24 * time.Hour)
	e = s.tx(r.Context(), func(t pgx.Tx) error {
		var valid bool
		if e := t.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND auth_epoch=$2 AND NOT disabled)", p.ID, epoch).Scan(&valid); e != nil {
			return e
		}
		if !valid {
			return fail(401, "unauthorized", "credentials changed")
		}
		refreshHash := ""
		if req.Client != "cli" {
			refreshHash = digest([]byte(refresh))
		}
		if _, e := t.Exec(r.Context(), "INSERT INTO sessions(id,user_id,auth_epoch,client,refresh_hash,expires_at,idle_until) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,least($6,now()+interval '7 days'))", id, p.ID, epoch, req.Client, refreshHash, expiry); e != nil {
			return e
		}
		return audit(r.Context(), t, p.ID, "session.login", id)
	})
	if e != nil {
		return e
	}
	if req.Client != "cli" {
		s.clientCookie(w, refresh, expiry, req.Client)
	}
	s.sessionResponse(w, id, p, req.Client)
	return nil
}

// Deterministic successor lets a concurrent retry receive exactly the same cookie
// without persisting plaintext refresh credentials. The grace window never slides.
func (s *Server) successor(old string) string {
	return hex.EncodeToString(mac(s.cfg.SessionKey, "session-refresh-v1:"+old))
}
func browserClient(b []byte) (string, error) {
	var p struct {
		Client string `json:"client"`
	}
	if len(b) > 0 && string(b) != "null" {
		if e := parse(b, &p); e != nil {
			return "", e
		}
	}
	if p.Client == "" {
		p.Client = "web"
	}
	if p.Client != "web" && p.Client != "status" {
		return "", fail(400, "invalid_client", "invalid browser client")
	}
	return p.Client, nil
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	if e := s.csrf(r); e != nil {
		return e
	}
	client, e := browserClient(b)
	if e != nil {
		return e
	}
	cookie, e := r.Cookie(cookieName(client))
	if e != nil || len(cookie.Value) > 128 {
		return fail(401, "unauthorized", "session required")
	}
	var p principal
	var id, current, previous string
	var grace *time.Time
	var expires time.Time
	next := ""
	e = s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		hash := digest([]byte(cookie.Value))
		e := t.QueryRow(ctx, `SELECT s.id,a.id,a.username,a.role,s.refresh_hash,coalesce(s.previous_hash,''),s.previous_until,s.expires_at FROM sessions s JOIN accounts a ON a.id=s.user_id WHERE (s.refresh_hash=$1 OR s.previous_hash=$1) AND s.client=$2 AND ($2<>'status' OR a.role='admin') AND s.revoked_at IS NULL AND s.expires_at>now() AND s.idle_until>now() AND NOT a.disabled AND a.auth_epoch=s.auth_epoch FOR UPDATE OF s`, hash, client).Scan(&id, &p.ID, &p.Username, &p.Role, &current, &previous, &grace, &expires)
		if e == pgx.ErrNoRows {
			return fail(401, "unauthorized", "session expired or revoked")
		}
		if e != nil {
			return e
		}
		if hash == previous {
			if grace == nil || !grace.After(time.Now()) {
				return fail(401, "unauthorized", "refresh credential expired")
			}
			next = s.successor(cookie.Value)
			if digest([]byte(next)) != current {
				return fail(401, "unauthorized", "refresh credential rejected")
			}
			return nil
		}
		// Hold the current generation stable during the retry window, so racing responses
		// cannot strand another tab with a credential older than the accepted predecessor.
		if grace != nil && grace.After(time.Now()) {
			next = cookie.Value
			return nil
		}
		next = s.successor(cookie.Value)
		_, e = t.Exec(ctx, "UPDATE sessions SET previous_hash=refresh_hash,previous_until=now()+interval '30 seconds',refresh_hash=$2,rotation=rotation+1,idle_until=least(expires_at,now()+interval '7 days') WHERE id=$1", id, digest([]byte(next)))
		return e
	})
	if e != nil {
		return e
	}
	s.clientCookie(w, next, expires, client)
	s.sessionResponse(w, id, p, client)
	return nil
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	if e := s.csrf(r); e != nil {
		return e
	}
	client, e := browserClient(b)
	if e != nil {
		return e
	}
	cookie, e := r.Cookie(cookieName(client))
	if e == nil {
		e = s.tx(r.Context(), func(t pgx.Tx) error {
			_, e := t.Exec(r.Context(), "UPDATE sessions SET revoked_at=coalesce(revoked_at,now()) WHERE (refresh_hash=$1 OR previous_hash=$1) AND client=$2", digest([]byte(cookie.Value)), client)
			return e
		})
		if e != nil {
			return e
		}
	}
	s.clientCookie(w, "", time.Unix(1, 0), client)
	jsonOut(w, 200, map[string]string{"status": "signed_out"})
	return nil
}
func (s *Server) me(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	jsonOut(w, 200, r.Context().Value(principalKey{}))
	return nil
}
