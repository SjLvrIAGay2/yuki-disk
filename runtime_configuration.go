package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
)

type runtimeValues struct {
	TrashSeconds           int `json:"trash_seconds"`
	UploadSeconds          int `json:"upload_seconds"`
	UserShareSeconds       int `json:"user_share_seconds"`
	APIShareSeconds        int `json:"api_share_seconds"`
	LoginAttempts          int `json:"login_attempts"`
	DownloadAuthorizations int `json:"download_authorizations"`
}
type runtimeSnapshot struct {
	Version int64
	Values  runtimeValues
}
type runtimeContextKey struct{}

func (v runtimeValues) valid() bool {
	return v.TrashSeconds >= 60 && v.TrashSeconds <= 90*86400 && v.UploadSeconds >= 60 && v.UploadSeconds <= 86400 && v.UserShareSeconds >= 60 && v.UserShareSeconds <= 30*86400 && v.APIShareSeconds >= 0 && v.APIShareSeconds <= 36500*86400 && (v.APIShareSeconds == 0 || v.APIShareSeconds >= 60) && v.LoginAttempts >= 1 && v.LoginAttempts <= 120 && v.DownloadAuthorizations >= 1 && v.DownloadAuthorizations <= 6000
}
func (s *Server) runtime(ctx context.Context) runtimeValues {
	if v, ok := ctx.Value(runtimeContextKey{}).(runtimeSnapshot); ok {
		return v.Values
	}
	if v := s.runtimeState.Load(); v != nil {
		return v.Values
	}
	return runtimeValues{TrashSeconds: s.cfg.TrashSeconds, UploadSeconds: s.cfg.UploadSeconds, UserShareSeconds: 86400, APIShareSeconds: 86400, LoginAttempts: 20, DownloadAuthorizations: 600}
}
func (s *Server) initializeRuntime(ctx context.Context, t pgx.Tx) error {
	v := runtimeValues{TrashSeconds: s.cfg.TrashSeconds, UploadSeconds: s.cfg.UploadSeconds, UserShareSeconds: 86400, APIShareSeconds: 86400, LoginAttempts: 20, DownloadAuthorizations: 600}
	b, _ := json.Marshal(v)
	if _, e := t.Exec(ctx, "INSERT INTO runtime_configuration(singleton,version,settings) VALUES(true,1,$1) ON CONFLICT DO NOTHING", b); e != nil {
		return e
	}
	var version int64
	var raw []byte
	if e := t.QueryRow(ctx, "SELECT version,settings FROM runtime_configuration WHERE singleton").Scan(&version, &raw); e != nil {
		return e
	}
	if e := json.Unmarshal(raw, &v); e != nil {
		return e
	}
	if !v.valid() {
		return fail(503, "invalid_runtime_configuration", "persisted configuration is invalid")
	}
	s.runtimeState.Store(&runtimeSnapshot{version, v})
	return nil
}
func (s *Server) observeConfiguration(ctx context.Context) error {
	var version int64
	var raw []byte
	if e := s.db.QueryRow(ctx, "SELECT version,settings FROM runtime_configuration WHERE singleton").Scan(&version, &raw); e != nil {
		return e
	}
	var v runtimeValues
	state := "applied"
	if json.Unmarshal(raw, &v) != nil || !v.valid() {
		state = "failed"
	} else {
		s.runtimeState.Store(&runtimeSnapshot{version, v})
	}
	applied := s.runtimeState.Load().Version
	_, e := s.db.Exec(ctx, "INSERT INTO configuration_instances(id,target_version,applied_version,state) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET target_version=$2,applied_version=$3,state=$4,observed_at=now()", s.instanceID, version, applied, state)
	return e
}
func (s *Server) watchConfiguration(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		probe, cancel := context.WithTimeout(ctx, time.Second)
		_ = s.observeConfiguration(probe)
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Server) configuration(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	if r.Method == "POST" {
		var p struct {
			Version  int64         `json:"version"`
			Settings runtimeValues `json:"settings"`
		}
		if e := parse(b, &p); e != nil {
			return e
		}
		e := s.tx(r.Context(), func(t pgx.Tx) error {
			var version int64
			var current []byte
			if e := t.QueryRow(r.Context(), "SELECT version,settings FROM runtime_configuration WHERE singleton FOR UPDATE").Scan(&version, &current); e != nil {
				return e
			}
			if version != p.Version {
				return fail(409, "version_conflict", "configuration changed")
			}
			// Omitted fields inherit the authoritative target, never a lagging process snapshot.
			if e := json.Unmarshal(current, &p.Settings); e != nil {
				return e
			}
			if e := parse(b, &p); e != nil {
				return e
			}
			if !p.Settings.valid() {
				return fail(400, "invalid_configuration", "configuration outside allowed limits")
			}
			raw, _ := json.Marshal(p.Settings)
			tag, e := t.Exec(r.Context(), "UPDATE runtime_configuration SET version=version+1,settings=$2,updated_at=now() WHERE singleton AND version=$1", p.Version, raw)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return fail(409, "version_conflict", "configuration changed")
			}
			return audit(r.Context(), t, actor, "configuration.update", "")
		})
		if e != nil {
			return e
		}
	}
	var target int64
	var settings json.RawMessage
	if e := s.query(r.Context()).QueryRow(r.Context(), "SELECT version,settings FROM runtime_configuration WHERE singleton").Scan(&target, &settings); e != nil {
		return e
	}
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT id,target_version,applied_version,state,observed_at FROM configuration_instances WHERE observed_at>now()-interval '1 day' ORDER BY id")
	if e != nil {
		return e
	}
	defer rows.Close()
	instances := []map[string]any{}
	all := true
	for rows.Next() {
		var id, state string
		var wanted, applied int64
		var observed time.Time
		if e = rows.Scan(&id, &wanted, &applied, &state, &observed); e != nil {
			return e
		}
		if observed.Before(time.Now().Add(-10 * time.Second)) {
			state = "stale"
		}
		if state != "applied" || applied != target {
			all = false
		}
		instances = append(instances, map[string]any{"id": id, "target_version": wanted, "applied_version": applied, "state": state, "observed_at": observed})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"version": target, "settings": settings, "instances": instances, "all_observed_applied": all && len(instances) > 0})
	return nil
}
