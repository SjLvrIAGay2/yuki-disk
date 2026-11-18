package main

import (
	"net/http"
	"time"
)

func (s *Server) files(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	limit, _, e := pagination(r)
	if e != nil {
		return e
	}
	cursor := r.URL.Query().Get("cursor")
	trash := r.URL.Query().Get("trash") == "true"
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT v.id,v.path,v.size,v.created_at,p.trashed_at,p.expires_at FROM versions v JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path WHERE (p.trashed_at IS NOT NULL)=$1 AND v.id>$2 AND ($4='' OR v.path=$4) ORDER BY v.id LIMIT $3", trash, cursor, limit, r.URL.Query().Get("path"))
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, path string
		var size int64
		var created time.Time
		var deleted, expires *time.Time
		if e = rows.Scan(&id, &path, &size, &created, &deleted, &expires); e != nil {
			return e
		}
		items = append(items, map[string]any{"id": id, "path": path, "size": size, "created_at": created, "trashed_at": deleted, "expires_at": expires})
		cursor = id
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_cursor": cursor})
	return nil
}
