package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (s *Server) trash(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	return s.changeTrash(w, r, b, false)
}

func (s *Server) restore(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	return s.changeTrash(w, r, b, true)
}

func (s *Server) changeTrash(w http.ResponseWriter, r *http.Request, b []byte, restore bool) error {
	var req struct {
		Path string `json:"path"`
	}
	if e := parse(b, &req); e != nil {
		return e
	}
	if !validPath(req.Path) {
		return fail(400, "invalid_path", "invalid path")
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		return s.changeTree(r.Context(), t, req.Path, restore)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"path": req.Path, "restored": restore})
	return nil
}

func (s *Server) changeTree(ctx context.Context, t pgx.Tx, p string, restore bool) error {
	if err := readonlyPath(ctx, t, p, true); err != nil {
		return err
	}
	var exists bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1)", p).Scan(&exists); e != nil {
		return e
	}
	if !exists {
		return fail(404, "not_found", "path not found")
	}
	if restore {
		_, e := s.restoreTree(ctx, t, p)
		return e
	}
	_, e := t.Exec(ctx, "UPDATE paths SET trash_id=coalesce(trash_id,md5($3||path)),trashed_at=coalesce(trashed_at,now()),expires_at=coalesce(expires_at,now()+make_interval(secs=>$2)) WHERE path=$1 OR starts_with(path,$1||'/')", p, s.runtime(ctx).TrashSeconds, randomID())
	return e
}
