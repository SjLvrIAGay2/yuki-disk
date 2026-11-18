package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,63}$`)

func passwordHash(password string) string {
	salt := randomID()
	key, _ := pbkdf2.Key(sha256.New, password, []byte(salt), 600000, 32)
	return "pbkdf2-sha256$600000$" + salt + "$" + hex.EncodeToString(key)
}
func passwordMatches(password, stored string) bool {
	a := strings.Split(stored, "$")
	if len(a) != 4 || a[0] != "pbkdf2-sha256" || a[1] != "600000" {
		return false
	}
	key, _ := pbkdf2.Key(sha256.New, password, []byte(a[2]), 600000, 32)
	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(key)), []byte(a[3])) == 1
}
func validCredentials(username, password string) bool {
	return usernamePattern.MatchString(username) && len(password) >= 12 && len(password) <= 1024
}
func (s *Server) bootstrapIdentity(ctx context.Context, t pgx.Tx) error {
	var bypass bool
	if e := t.QueryRow(ctx, "SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname=current_user").Scan(&bypass); e != nil {
		return e
	}
	if bypass {
		return errors.New("database role must not bypass row security")
	}
	var initialized bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE role='admin')").Scan(&initialized); e != nil {
		return e
	}
	if initialized {
		return nil
	}
	a := s.cfg.BootstrapAdmin
	if a == nil || !validCredentials(a.Username, a.Password) {
		return errors.New("first start requires explicit bootstrap_admin credentials")
	}
	var legacy bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths) OR EXISTS(SELECT 1 FROM uploads) OR EXISTS(SELECT 1 FROM operations) OR EXISTS(SELECT 1 FROM visibility)").Scan(&legacy); e != nil {
		return e
	}
	if legacy && s.cfg.LegacyUsername == "" {
		return errors.New("existing data requires explicit legacy_username before migration")
	}
	if s.cfg.LegacyUsername != "" {
		if !validCredentials(s.cfg.LegacyUsername, s.cfg.Password) || s.cfg.LegacyUsername == a.Username {
			return errors.New("invalid legacy account")
		}
		if _, e := t.Exec(ctx, "UPDATE accounts SET username=$1,password_hash=$2,disabled=false WHERE id='legacy'", s.cfg.LegacyUsername, passwordHash(s.cfg.Password)); e != nil {
			return e
		}
	}
	_, e := t.Exec(ctx, "INSERT INTO accounts(id,username,password_hash,role) VALUES($1,$2,$3,'admin')", randomID(), a.Username, passwordHash(a.Password))
	return e
}

func (s *Server) accountList(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT id,username,role,disabled,auth_epoch,deleted_at,purge_after FROM accounts WHERE username<>'!unassigned-legacy' AND purged_at IS NULL ORDER BY username LIMIT 1000")
	if e != nil {
		return e
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, name, role string
		var disabled bool
		var epoch int64
		var deleted, deadline *time.Time
		if e = rows.Scan(&id, &name, &role, &disabled, &epoch, &deleted, &deadline); e != nil {
			return e
		}
		list = append(list, map[string]any{"id": id, "username": name, "role": role, "disabled": disabled, "version": epoch, "deleted_at": deleted, "purge_after": deadline})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": list})
	return nil
}
func (s *Server) accountCreate(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	var p struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if !validCredentials(p.Username, p.Password) {
		return fail(400, "invalid_credentials", "invalid username or password length")
	}
	id := randomID()
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		var exists bool
		if e := t.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM accounts WHERE username=$1)", p.Username).Scan(&exists); e != nil {
			return e
		}
		if exists {
			return fail(409, "account_exists", "account already exists")
		}
		if _, e := t.Exec(r.Context(), "INSERT INTO accounts(id,username,password_hash,role) VALUES($1,$2,$3,'user')", id, p.Username, passwordHash(p.Password)); e != nil {
			return e
		}
		return audit(r.Context(), t, actor, "account.create", id)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 201, map[string]any{"id": id, "username": p.Username, "role": "user"})
	return nil
}
func (s *Server) accountUpdate(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	var p struct {
		Version  int64  `json:"version"`
		Disabled *bool  `json:"disabled"`
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if p.Version < 1 || (p.Role != "" && p.Role != "user" && p.Role != "admin") || (p.Password != "" && (len(p.Password) < 12 || len(p.Password) > 1024)) {
		return fail(400, "invalid_account", "invalid account update")
	}
	id := r.PathValue("id")
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var role string
		var disabled bool
		var version int64
		e := t.QueryRow(ctx, "SELECT role,disabled,auth_epoch FROM accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&role, &disabled, &version)
		if e == pgx.ErrNoRows {
			return fail(404, "not_found", "account not found")
		}
		if e != nil {
			return e
		}
		if version != p.Version {
			return fail(409, "version_conflict", "account changed")
		}
		nextRole, nextDisabled := role, disabled
		if p.Role != "" {
			nextRole = p.Role
		}
		if p.Disabled != nil {
			nextDisabled = *p.Disabled
		}
		if role == "admin" && !disabled && (nextRole != "admin" || nextDisabled) {
			var n int
			if e = t.QueryRow(ctx, "SELECT count(*) FROM accounts WHERE role='admin' AND NOT disabled AND id<>$1", id).Scan(&n); e != nil {
				return e
			}
			if n == 0 {
				return fail(409, "last_admin", "at least one available administrator is required")
			}
		}
		hash := ""
		if p.Password != "" {
			hash = passwordHash(p.Password)
		}
		if _, e = t.Exec(ctx, "UPDATE accounts SET role=$2,disabled=$3,auth_epoch=auth_epoch+1,password_hash=CASE WHEN $4='' THEN password_hash ELSE $4 END WHERE id=$1", id, nextRole, nextDisabled, hash); e != nil {
			return e
		}
		if _, e = t.Exec(ctx, "UPDATE sessions SET revoked_at=coalesce(revoked_at,now()) WHERE user_id=$1", id); e != nil {
			return e
		}
		return audit(ctx, t, actor, "account.update", id)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"status": "updated", "version": p.Version + 1})
	return nil
}
func audit(ctx context.Context, t pgx.Tx, actor, action, target string) error {
	_, e := t.Exec(ctx, "INSERT INTO security_audit(actor_id,action,target_id) VALUES(NULLIF($1,''),$2,$3)", actor, action, target)
	return e
}
