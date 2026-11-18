package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"path"
	"strings"
)

func parentPath(p string) string {
	d := path.Dir(p)
	if d == "." {
		return ""
	}
	return d
}
func ensureParents(ctx context.Context, t pgx.Tx, p string) error {
	if err := readonlyPath(ctx, t, p, false); err != nil {
		return err
	}
	parts := strings.Split(p, "/")
	for i := 1; i < len(parts); i++ {
		dir := strings.Join(parts[:i], "/")
		if _, e := t.Exec(ctx, "INSERT INTO paths(path,is_dir) VALUES($1,true) ON CONFLICT DO NOTHING", dir); e != nil {
			return e
		}
		var usable bool
		if e := t.QueryRow(ctx, "SELECT is_dir AND trashed_at IS NULL FROM paths WHERE path=$1", dir).Scan(&usable); e != nil {
			return e
		}
		if !usable {
			return fail(409, "parent_unavailable", "parent is a file or is in the recycle bin")
		}
	}
	return nil
}
func checkUploadPath(ctx context.Context, t pgx.Tx, p string) error {
	if e := ensureParents(ctx, t, p); e != nil {
		return e
	}
	var conflict bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1 AND (is_dir OR trashed_at IS NOT NULL))", p).Scan(&conflict); e != nil {
		return e
	}
	if conflict {
		return fail(409, "path_unavailable", "path is a folder or is in the recycle bin")
	}
	return nil
}
func (s *Server) mkdir(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	var p struct {
		Path string `json:"path"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if !validPath(p.Path) {
		return fail(400, "invalid_path", "invalid folder path")
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		if e := ensureParents(r.Context(), t, p.Path); e != nil {
			return e
		}
		var exists bool
		if e := t.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1)", p.Path).Scan(&exists); e != nil {
			return e
		}
		if exists {
			return fail(409, "path_exists", "destination already exists")
		}
		_, e := t.Exec(r.Context(), "INSERT INTO paths(path,is_dir) VALUES($1,true)", p.Path)
		return e
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"path": p.Path})
	return nil
}

func (s *Server) browse(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	dir := r.URL.Query().Get("path")
	if dir != "" && !validPath(dir) {
		return fail(400, "invalid_path", "invalid directory")
	}
	limit, _, e := pagination(r)
	if e != nil {
		return e
	}
	trash := r.URL.Query().Get("trash") == "true"
	if !trash {
		if m, ok := s.mountFor(r.Context(), dir); ok {
			return s.browseMount(w, r, m)
		}
	}
	query := r.URL.Query().Get("q")
	if len(query) > 256 {
		return fail(400, "invalid_search", "search is too long")
	}
	if dir != "" && !trash {
		var ok bool
		if e = s.query(r.Context()).QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1 AND is_dir AND trashed_at IS NULL)", dir).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return fail(404, "not_found", "folder not found")
		}
	}
	rows, e := s.query(r.Context()).Query(r.Context(), `SELECT p.path,p.is_dir,p.expires_at,p.created_at,p.trashed_at,p.trash_id,
 coalesce(v.id,''),coalesce(v.size,0),v.created_at,
 (SELECT count(*) FROM versions WHERE path=p.path), EXISTS(SELECT 1 FROM mounts m WHERE p.path=m.path OR starts_with(p.path,m.path||'/'))
 FROM paths p LEFT JOIN LATERAL (SELECT id,size,created_at FROM versions WHERE path=p.path ORDER BY created_at DESC,id DESC LIMIT 1) v ON true
 WHERE (p.trashed_at IS NOT NULL)=$1 AND p.path>$2
 AND ($3='' OR strpos(lower(p.path),lower($3))>0)
 AND (CASE WHEN $1 THEN true
      WHEN $3<>'' THEN true ELSE regexp_replace(p.path,'/?[^/]+$','')=$4 END)
 ORDER BY p.path LIMIT $5`, trash, r.URL.Query().Get("cursor"), query, dir, limit+1)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var p string
		var isDir bool
		var readonly bool
		var expiry, created, modified, deleted, trashID any
		var id string
		var size, count int64
		if e = rows.Scan(&p, &isDir, &expiry, &created, &deleted, &trashID, &id, &size, &modified, &count, &readonly); e != nil {
			return e
		}
		items = append(items, map[string]any{"path": p, "name": path.Base(p), "is_dir": isDir, "expires_at": expiry, "created_at": created, "modified_at": modified, "trashed_at": deleted, "trash_id": trashID, "version_id": id, "size": size, "versions": count, "readonly": readonly})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1]["path"].(string)
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_cursor": next})
	return nil
}
