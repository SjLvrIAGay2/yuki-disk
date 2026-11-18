package main

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"path"
	"strings"
	"time"
)

func (s *Server) shareSecret(kind, id string) string {
	return hex.EncodeToString(mac(s.cfg.SessionKey, "file-share-v2:"+kind+":"+id))
}
func legacyPublicGone(http.ResponseWriter, *http.Request, []byte, string) error {
	return fail(410, "legacy_share_expired", "legacy public links have expired; ask the owner for a new share")
}
func (s *Server) createShare(w http.ResponseWriter, r *http.Request, b []byte, owner string) error {
	var p struct {
		Version   string `json:"version_id"`
		Kind      string `json:"kind"`
		Expires   int    `json:"expires_in"`
		Permanent bool   `json:"permanent"`
		Password  string `json:"password"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if (p.Kind != "user" && p.Kind != "api") || (p.Permanent && (p.Kind != "api" || p.Expires != 0)) || (!p.Permanent && (p.Expires < 60 || p.Expires > 36500*86400 || (p.Kind == "user" && p.Expires > 30*86400))) || len(p.Password) > 128 || (p.Kind == "api" && p.Password != "") {
		return fail(400, "invalid_share", "invalid share scope or lifetime")
	}
	idem := r.Header.Get("Idempotency-Key")
	if e := checkIdempotencyKey(idem); e != nil {
		return e
	}
	canonical, _ := json.Marshal(p)
	requestDigest := hex.EncodeToString(mac(s.cfg.SessionKey, "file-share-request-v2:"+string(canonical)))
	id := randomID()
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var prior string
		e := t.QueryRow(ctx, "SELECT id,request_digest FROM file_shares WHERE owner_id=$1 AND idem=$2", owner, idem).Scan(&id, &prior)
		if e == nil {
			if prior != requestDigest {
				return fail(409, "idempotency_conflict", "key used with another share")
			}
			return nil
		}
		if e != pgx.ErrNoRows {
			return e
		}
		var valid bool
		if e = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM versions v JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path WHERE v.id=$1 AND v.owner_id=$2 AND p.trashed_at IS NULL)", p.Version, owner).Scan(&valid); e != nil {
			return e
		}
		if !valid {
			return fail(404, "not_found", "file unavailable")
		}
		hash := ""
		if p.Password != "" {
			hash = passwordHash(p.Password)
		}
		if _, e = t.Exec(ctx, "INSERT INTO file_shares(id,owner_id,version_id,kind,password_hash,expires_at,idem,request_digest) VALUES($1,$2,$3,$4,$5,CASE WHEN $6::bigint=0 THEN NULL ELSE now()+make_interval(secs=>$6::double precision) END,$7,$8)", id, owner, p.Version, p.Kind, hash, p.Expires, idem, requestDigest); e != nil {
			return e
		}
		return audit(ctx, t, owner, "share.create", id)
	})
	if e != nil {
		return e
	}
	out := map[string]any{"id": id, "kind": p.Kind}
	if p.Kind == "user" {
		out["access_key"] = s.shareSecret("user", id)
	}
	jsonOut(w, 200, out)
	return nil
}
func (s *Server) listShares(w http.ResponseWriter, r *http.Request, _ []byte, owner string) error {
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT id,version_id,kind,expires_at,revoked_at FROM file_shares WHERE owner_id=$1 ORDER BY created_at DESC LIMIT 1000", owner)
	if e != nil {
		return e
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, version, kind string
		var expires *time.Time
		var revoked *time.Time
		if e = rows.Scan(&id, &version, &kind, &expires, &revoked); e != nil {
			return e
		}
		item := map[string]any{"id": id, "version_id": version, "kind": kind, "expires_at": expires, "revoked_at": revoked}
		if kind == "user" {
			item["access_key"] = s.shareSecret("user", id)
		}
		list = append(list, item)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": list})
	return nil
}
func (s *Server) revokeShare(w http.ResponseWriter, r *http.Request, _ []byte, owner string) error {
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		tag, e := t.Exec(r.Context(), "UPDATE file_shares SET revoked_at=coalesce(revoked_at,now()) WHERE id=$1 AND owner_id=$2", r.PathValue("id"), owner)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return fail(404, "not_found", "share not found")
		}
		return audit(r.Context(), t, owner, "share.revoke", r.PathValue("id"))
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"status": "revoked"})
	return nil
}

type shareResource struct {
	Owner, Version, Name, Password string
	Expires                        *time.Time
}

func (s *Server) shareResource(ctx context.Context, t pgx.Tx, id, kind string) (shareResource, error) {
	var out shareResource
	var revoked *time.Time
	var disabled bool
	var role string
	e := t.QueryRow(ctx, `SELECT s.owner_id,s.version_id,s.password_hash,s.expires_at,s.revoked_at,a.disabled,a.role FROM file_shares s JOIN accounts a ON a.id=s.owner_id WHERE s.id=$1 AND s.kind=$2`, id, kind).Scan(&out.Owner, &out.Version, &out.Password, &out.Expires, &revoked, &disabled, &role)
	if e == pgx.ErrNoRows {
		return out, fail(404, "share_unavailable", "share unavailable")
	}
	if e != nil {
		return out, e
	}
	if revoked != nil {
		return out, fail(410, "share_revoked", "share revoked")
	}
	if out.Expires != nil && !out.Expires.After(time.Now()) {
		return out, fail(410, "share_expired", "share expired")
	}
	if disabled || role != "user" {
		return out, fail(404, "share_unavailable", "share unavailable")
	}
	e = t.QueryRow(ctx, `SELECT v.path FROM versions v JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path WHERE v.id=$1 AND v.owner_id=$2 AND p.trashed_at IS NULL`, out.Version, out.Owner).Scan(&out.Name)
	if e == pgx.ErrNoRows {
		return out, fail(404, "resource_unavailable", "file unavailable")
	}
	out.Name = path.Base(out.Name)
	return out, e
}
func (s *Server) shareAccess(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	if e := s.loginLimit(r); e != nil {
		return e
	}
	id := r.PathValue("id")
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(key), []byte(s.shareSecret("user", id))) != 1 {
		return fail(404, "share_unavailable", "share unavailable")
	}
	var p struct {
		Password string `json:"password"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Password) > 128 {
		return fail(400, "invalid_password", "invalid access credential")
	}
	var resource shareResource
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		var e error
		resource, e = s.shareResource(r.Context(), t, id, "user")
		if e != nil {
			return e
		}
		if resource.Password != "" && !passwordMatches(p.Password, resource.Password) {
			return fail(401, "share_password_required", "access credential required")
		}
		return nil
	})
	if e != nil {
		return e
	}
	expires := time.Now().Add(5 * time.Minute)
	if resource.Expires != nil && resource.Expires.Before(expires) {
		expires = *resource.Expires
	}
	jsonOut(w, 200, map[string]any{"name": resource.Name, "token": encodeToken(s.cfg.SessionKey, Session{2, "share-download", expires.Unix(), id}), "expires_at": resource.Expires})
	return nil
}
func (s *Server) applicationCreate(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	var p struct {
		Grant     string `json:"grant_id"`
		Name      string `json:"name"`
		Expires   int    `json:"expires_in"`
		Permanent bool   `json:"permanent"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Name) < 1 || len(p.Name) > 80 || (p.Permanent && p.Expires != 0) || (!p.Permanent && (p.Expires < 60 || p.Expires > 36500*86400)) {
		return fail(400, "invalid_application", "invalid application")
	}
	idem := r.Header.Get("Idempotency-Key")
	if e := checkIdempotencyKey(idem); e != nil {
		return e
	}
	canonical, _ := json.Marshal(p)
	id := randomID()
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		var prior string
		e := t.QueryRow(r.Context(), "SELECT id,request_digest FROM api_applications WHERE idem=$1", idem).Scan(&id, &prior)
		if e == nil {
			if prior != digest(canonical) {
				return fail(409, "idempotency_conflict", "application changed")
			}
			return nil
		}
		if e != pgx.ErrNoRows {
			return e
		}
		grant, e := s.shareResource(r.Context(), t, p.Grant, "api")
		if e != nil {
			return e
		}
		var expiry *time.Time
		if p.Permanent {
			if grant.Expires != nil {
				return fail(409, "grant_not_permanent", "permanent application requires a permanent owner grant")
			}
		} else {
			value := time.Now().Add(time.Duration(p.Expires) * time.Second)
			expiry = &value
			if grant.Expires != nil && expiry.After(*grant.Expires) {
				expiry = grant.Expires
			}
		}
		if _, e = t.Exec(r.Context(), "INSERT INTO api_applications(id,grant_id,name,credential_hash,expires_at,created_by,idem,request_digest) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", id, p.Grant, p.Name, digest([]byte(s.shareSecret("api", id))), expiry, actor, idem, digest(canonical)); e != nil {
			return e
		}
		return audit(r.Context(), t, actor, "application.create", id)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"id": id, "credential": s.shareSecret("api", id), "operation": "download"})
	return nil
}
func (s *Server) applicationList(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT id,grant_id,name,expires_at,revoked_at FROM api_applications ORDER BY created_at DESC LIMIT 1000")
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, grant, name string
		var expires *time.Time
		var revoked *time.Time
		if e = rows.Scan(&id, &grant, &name, &expires, &revoked); e != nil {
			return e
		}
		items = append(items, map[string]any{"id": id, "grant_id": grant, "name": name, "expires_at": expires, "revoked_at": revoked})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items})
	return nil
}
func (s *Server) applicationRevoke(w http.ResponseWriter, r *http.Request, _ []byte, actor string) error {
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		tag, e := t.Exec(r.Context(), "UPDATE api_applications SET revoked_at=coalesce(revoked_at,now()) WHERE id=$1", r.PathValue("id"))
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return fail(404, "not_found", "application not found")
		}
		return audit(r.Context(), t, actor, "application.revoke", r.PathValue("id"))
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"status": "revoked"})
	return nil
}
func (s *Server) sharedDownload(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
		response := &committedResponse{header: http.Header{}, status: 200}
		e := s.tx(r.Context(), func(t pgx.Tx) error {
			ctx := context.WithValue(r.Context(), requestTransactionKey{}, t)
			id := r.PathValue("id")
			grant := id
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if kind == "user" {
				var p Session
				if decodeToken(s.cfg.SessionKey, raw, &p) != nil || p.Version != 2 || p.Purpose != "share-download" || p.ID != id || p.Expires <= time.Now().Unix() {
					return fail(401, "share_access_required", "share access required")
				}
			} else {
				var expires *time.Time
				var revoked *time.Time
				e := t.QueryRow(ctx, "SELECT grant_id,expires_at,revoked_at FROM api_applications WHERE id=$1 AND credential_hash=$2", id, digest([]byte(raw))).Scan(&grant, &expires, &revoked)
				if e == pgx.ErrNoRows {
					return fail(401, "application_rejected", "application credential rejected")
				}
				if e != nil {
					return e
				}
				if revoked != nil || (expires != nil && !expires.After(time.Now())) {
					return fail(410, "application_expired", "application expired or revoked")
				}
			}
			var count int
			e := t.QueryRow(ctx, `INSERT INTO auth_attempts(key,count,expires_at) VALUES($1,1,now()+interval '1 minute') ON CONFLICT(key) DO UPDATE SET count=CASE WHEN auth_attempts.expires_at<=now() THEN 1 ELSE auth_attempts.count+1 END,expires_at=CASE WHEN auth_attempts.expires_at<=now() THEN now()+interval '1 minute' ELSE auth_attempts.expires_at END RETURNING count`, "download:"+kind+":"+id).Scan(&count)
			if e != nil {
				return e
			}
			if count > s.runtime(ctx).DownloadAuthorizations {
				return fail(429, "rate_limited", "transfer request limit reached")
			}
			resource, e := s.shareResource(ctx, t, grant, kind)
			if e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", resource.Owner); e != nil {
				return e
			}
			rr := r.Clone(ctx)
			rr.SetPathValue("id", resource.Version)
			if rr.PathValue("hash") != "" {
				e = s.downloadBlock(response, rr, nil, resource.Owner)
			} else {
				q := rr.URL.Query()
				q.Set("manifest", "true")
				rr.URL.RawQuery = q.Encode()
				e = s.download(response, rr, nil, resource.Owner)
				if e == nil {
					var data map[string]any
					if e = json.Unmarshal(response.Bytes(), &data); e != nil {
						return e
					}
					delete(data, "path")
					delete(data, "version_id")
					data["name"] = resource.Name
					response.Reset()
					jsonOut(response, 200, data)
				}
			}
			if e != nil {
				return e
			}
			return audit(ctx, t, "", kind+".download", id)
		})
		if e != nil {
			return e
		}
		for k, v := range response.header {
			w.Header()[k] = v
		}
		w.WriteHeader(response.status)
		_, e = w.Write(response.Bytes())
		return e
	}
}
