package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
)

// All mutations and expiry share the existing PG advisory transaction boundary.
func (s *Server) accountDeletion(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	var p struct {
		Version   int64  `json:"version"`
		Operation string `json:"operation"`
		Username  string `json:"username"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if p.Version < 1 || (p.Operation != "delete" && p.Operation != "restore") {
		return fail(400, "invalid_account", "invalid deletion operation")
	}
	id := r.PathValue("id")
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var name, role string
		var disabled bool
		var version int64
		var deleted, deadline, purged *time.Time
		err := t.QueryRow(ctx, "SELECT username,role,disabled,auth_epoch,deleted_at,purge_after,purged_at FROM accounts WHERE id=$1 FOR UPDATE", id).Scan(&name, &role, &disabled, &version, &deleted, &deadline, &purged)
		if err == pgx.ErrNoRows {
			return fail(404, "not_found", "account not found")
		}
		if err != nil {
			return err
		}
		if version != p.Version {
			return fail(409, "version_conflict", "account changed")
		}
		if purged != nil {
			return fail(410, "account_purged", "account data already removed")
		}
		if p.Operation == "delete" {
			if p.Username != name {
				return fail(400, "confirmation_required", "confirm account username")
			}
			if deleted != nil {
				return fail(409, "account_deleted", "account already pending deletion")
			}
			if role == "admin" && !disabled {
				var count int
				if err = t.QueryRow(ctx, "SELECT count(*) FROM accounts WHERE role='admin' AND NOT disabled AND deleted_at IS NULL AND id<>$1", id).Scan(&count); err != nil {
					return err
				}
				if count == 0 {
					return fail(409, "last_admin", "at least one available administrator required")
				}
			}
			if _, err = t.Exec(ctx, "UPDATE accounts SET disabled=true,deleted_at=now(),purge_after=now()+interval '30 days',auth_epoch=auth_epoch+1 WHERE id=$1", id); err != nil {
				return err
			}
			if _, err = t.Exec(ctx, "UPDATE file_shares SET revoked_at=coalesce(revoked_at,now()) WHERE owner_id=$1", id); err != nil {
				return err
			}
		} else {
			if deleted == nil || deadline == nil {
				return fail(409, "account_not_deleted", "account not pending deletion")
			}
			if !deadline.After(time.Now()) {
				return fail(410, "restore_expired", "account restoration deadline expired")
			}
			// Pause recycle countdown while the account is quarantined; preserve prior remaining time.
			for _, table := range []string{"paths", "versions"} {
				if _, err = t.Exec(ctx, "UPDATE "+table+" SET expires_at=expires_at+(now()-$2::timestamptz) WHERE owner_id=$1 AND expires_at IS NOT NULL", id, *deleted); err != nil {
					return err
				}
			}
			if _, err = t.Exec(ctx, "UPDATE accounts SET disabled=false,deleted_at=NULL,purge_after=NULL,auth_epoch=auth_epoch+1 WHERE id=$1", id); err != nil {
				return err
			}
		}
		if _, err = t.Exec(ctx, "UPDATE sessions SET revoked_at=coalesce(revoked_at,now()) WHERE user_id=$1", id); err != nil {
			return err
		}
		return audit(ctx, t, actor, "account."+p.Operation, id)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"status": p.Operation, "version": p.Version + 1})
	return nil
}

func (s *Server) purgeDeletedAccounts(ctx context.Context, t pgx.Tx) error {
	rows, e := t.Query(ctx, "SELECT id FROM accounts WHERE purge_after<=now() AND purged_at IS NULL ORDER BY id LIMIT 10")
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if _, e = t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", id); e != nil {
			return e
		}
		vr, err := t.Query(ctx, "SELECT id FROM versions WHERE owner_id=$1", id)
		if err != nil {
			return err
		}
		versions := []string{}
		for vr.Next() {
			var v string
			if err = vr.Scan(&v); err != nil {
				vr.Close()
				return err
			}
			versions = append(versions, v)
		}
		err = vr.Err()
		vr.Close()
		if err != nil {
			return err
		}
		if e = s.purgeVersions(ctx, t, versions); e != nil {
			return e
		}
		// Removing a mount grant never modifies the external source directory.
		for _, table := range []string{"mounts", "visibility", "paths", "uploads", "operations"} {
			if _, e = t.Exec(ctx, "DELETE FROM "+table+" WHERE owner_id=$1", id); e != nil {
				return e
			}
		}
		if _, e = t.Exec(ctx, "DELETE FROM api_applications WHERE grant_id IN(SELECT id FROM file_shares WHERE owner_id=$1)", id); e != nil {
			return e
		}
		if _, e = t.Exec(ctx, "DELETE FROM file_shares WHERE owner_id=$1", id); e != nil {
			return e
		}
		if _, e = t.Exec(ctx, "DELETE FROM sessions WHERE user_id=$1", id); e != nil {
			return e
		}
		if _, e = t.Exec(ctx, "UPDATE accounts SET username='deleted-'||id,password_hash='',role='user',purged_at=now(),auth_epoch=auth_epoch+1 WHERE id=$1", id); e != nil {
			return e
		}
		if e = audit(ctx, t, "", "account.purge", id); e != nil {
			return e
		}
	}
	_, e = t.Exec(ctx, "SELECT set_config('yuki.owner','',true)")
	return e
}
