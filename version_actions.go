package main

import (
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
)

type VersionSelection struct {
	ID      string `json:"id"`
	TrashID string `json:"trash_id,omitempty"`
}
type VersionAction struct {
	Operation string             `json:"operation"`
	Items     []VersionSelection `json:"items"`
}

func (s *Server) versionAction(w http.ResponseWriter, r *http.Request, body []byte, _ string) error {
	var p VersionAction
	if err := parse(body, &p); err != nil {
		return err
	}
	if p.Operation != "trash" && p.Operation != "restore" && p.Operation != "purge" {
		return fail(400, "invalid_operation", "invalid version operation")
	}
	if len(p.Items) < 1 || len(p.Items) > 100 {
		return fail(400, "invalid_selection", "select 1..100 versions")
	}
	for _, v := range p.Items {
		if len(v.ID) < 16 || len(v.ID) > 128 || (p.Operation != "trash" && (len(v.TrashID) < 32 || len(v.TrashID) > 64)) {
			return fail(400, "invalid_selection", "invalid version identity")
		}
	}
	key := r.Header.Get("Idempotency-Key")
	if err := checkIdempotencyKey(key); err != nil {
		return err
	}
	canonical, _ := json.Marshal(p)
	signature := digest(append([]byte("version-actions:"), canonical...))
	var result json.RawMessage
	err := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		return operationResult(ctx, t, key, signature, &result, func() error {
			var err error
			changed := []map[string]string{}
			seen := map[string]bool{}
			for _, v := range p.Items {
				if seen[v.ID] {
					continue
				}
				seen[v.ID] = true
				var active, original, trashID *string
				var expires *time.Time
				err = t.QueryRow(ctx, "SELECT path,trash_path,trash_id,expires_at FROM versions WHERE id=$1", v.ID).Scan(&active, &original, &trashID, &expires)
				if err == pgx.ErrNoRows {
					return fail(404, "not_found", "version not found")
				}
				if err != nil {
					return err
				}
				target := ""
				switch p.Operation {
				case "trash":
					if active == nil {
						return fail(409, "version_changed", "version already removed")
					}
					var pathDeleted bool
					if err = t.QueryRow(ctx, "SELECT trashed_at IS NOT NULL FROM paths WHERE path=$1", *active).Scan(&pathDeleted); err != nil {
						return err
					}
					if pathDeleted {
						return fail(409, "version_changed", "restore the path before selecting versions")
					}
					_, err = t.Exec(ctx, "UPDATE versions SET trash_path=path,path=NULL,trashed_at=now(),expires_at=now()+make_interval(secs=>$2),trash_id=$3 WHERE id=$1", v.ID, s.runtime(ctx).TrashSeconds, randomID())
					if err == nil {
						_, err = t.Exec(ctx, "DELETE FROM paths p WHERE path=$1 AND NOT is_dir AND NOT EXISTS(SELECT 1 FROM versions v WHERE v.path=p.path)", *active)
					}
				case "restore", "purge":
					if active != nil || trashID == nil || *trashID != v.TrashID {
						return fail(409, "version_changed", "version deletion identity changed")
					}
					if p.Operation == "purge" {
						err = s.purgeVersions(ctx, t, []string{v.ID})
					} else {
						if expires == nil || !time.Now().Before(*expires) {
							return fail(410, "trash_expired", "restore window expired")
						}
						target, err = s.versionRestorePath(ctx, t, *original)
						if err == nil {
							_, err = t.Exec(ctx, "INSERT INTO paths(path) VALUES($1) ON CONFLICT DO NOTHING", target)
						}
						if err == nil {
							_, err = t.Exec(ctx, "UPDATE versions SET path=$2,trash_path=NULL,trashed_at=NULL,expires_at=NULL,trash_id=NULL WHERE id=$1", v.ID, target)
						}
					}
				}
				if err != nil {
					return err
				}
				changed = append(changed, map[string]string{"id": v.ID, "target": target})
			}
			result, _ = json.Marshal(map[string]any{"operation": p.Operation, "items": changed})
			return nil
		})
	})
	if err != nil {
		return err
	}
	jsonOut(w, 200, result)
	return nil
}
