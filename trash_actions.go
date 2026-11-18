package main

import (
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
)

type TrashItem struct {
	Path    string `json:"path"`
	TrashID string `json:"trash_id"`
}
type TrashAction struct {
	Operation string      `json:"operation"`
	Items     []TrashItem `json:"items"`
}

func (s *Server) trashAction(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	var p TrashAction
	if e := parse(b, &p); e != nil {
		return e
	}
	if p.Operation != "restore" && p.Operation != "purge" {
		return fail(400, "invalid_operation", "unsupported trash operation")
	}
	if len(p.Items) == 0 || len(p.Items) > 100 {
		return fail(400, "invalid_selection", "select 1..100 entries")
	}
	for _, item := range p.Items {
		if !validPath(item.Path) || len(item.TrashID) < 32 || len(item.TrashID) > 64 {
			return fail(400, "invalid_selection", "invalid trash identity")
		}
	}
	key := r.Header.Get("Idempotency-Key")
	if err := checkIdempotencyKey(key); err != nil {
		return err
	}
	canonical, _ := json.Marshal(p)
	var result json.RawMessage
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		return operationResult(ctx, t, key, digest(canonical), &result, func() error {
			var e error
			sources := []string{}
			// Validate every identity before changing any row. A stale selection cannot purge a new deletion.
			for _, item := range p.Items {
				var id *string
				e = t.QueryRow(ctx, "SELECT trash_id FROM paths WHERE path=$1", item.Path).Scan(&id)
				if e == pgx.ErrNoRows {
					if p.Operation == "purge" {
						continue
					}
					return fail(404, "not_found", "entry no longer exists")
				}
				if e != nil {
					return e
				}
				if id == nil || *id != item.TrashID {
					return fail(409, "trash_changed", "entry changed; refresh selection")
				}
				sources = append(sources, item.Path)
			}
			changed := []map[string]string{}
			for _, source := range compactPaths(sources) {
				target := ""
				if p.Operation == "purge" {
					e = s.purgeTree(ctx, t, source)
				} else {
					target, e = s.restoreTree(ctx, t, source)
				}
				if e != nil {
					return e
				}
				changed = append(changed, map[string]string{"path": source, "target": target})
			}
			result, _ = json.Marshal(map[string]any{"operation": p.Operation, "items": changed})
			return nil
		})
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, result)
	return nil
}
