package main

import (
	"net/http"
)

// History includes individually removed versions even after the last live
// version or its original parent directory disappears.
func (s *Server) versions(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	path := r.URL.Query().Get("path")
	trash := r.URL.Query().Get("trash") == "true"
	if (!trash && path == "") || (path != "" && !validPath(path)) {
		return fail(400, "invalid_path", "path required")
	}
	limit, _, err := pagination(r)
	if err != nil {
		return err
	}
	rows, err := s.query(r.Context()).Query(r.Context(), `SELECT v.id,coalesce(v.path,v.trash_path),v.size,v.created_at,
 coalesce(v.trashed_at,p.trashed_at),coalesce(v.expires_at,p.expires_at),v.trash_id,v.path IS NULL
 FROM versions v LEFT JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path
 WHERE ($1='' OR v.path=$1 OR v.trash_path=$1) AND (NOT $2 OR v.path IS NULL) AND v.id>$3
 ORDER BY v.id LIMIT $4`, path, trash, r.URL.Query().Get("cursor"), limit+1)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, p string
		var size int64
		var created, deleted, expires, trashID any
		var removed bool
		if err = rows.Scan(&id, &p, &size, &created, &deleted, &expires, &trashID, &removed); err != nil {
			return err
		}
		items = append(items, map[string]any{"id": id, "path": p, "size": size, "created_at": created, "trashed_at": deleted, "expires_at": expires, "trash_id": trashID, "deleted": removed})
	}
	if err = rows.Err(); err != nil {
		return err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1]["id"].(string)
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_cursor": next})
	return nil
}
