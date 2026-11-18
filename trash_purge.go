package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// Both automatic expiry and explicit purge release references in the deleting transaction.
// Bytes remain behind the upload grace and the normal delayed, fenced GC pipeline.
func (s *Server) purgePaths(ctx context.Context, t pgx.Tx, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	rows, e := t.Query(ctx, "SELECT id FROM versions WHERE path=ANY($1)", paths)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var h string
		if e = rows.Scan(&h); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, h)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if e = s.purgeVersions(ctx, t, ids); e != nil {
		return e
	}
	if _, e = t.Exec(ctx, "DELETE FROM paths WHERE path=ANY($1)", paths); e != nil {
		return e
	}
	// Pending writes to a permanently removed namespace cannot bring old files back.
	if _, e = t.Exec(ctx, `UPDATE uploads SET expires_at=now() WHERE NOT committed AND EXISTS(
  SELECT 1 FROM unnest($1::text[]) p WHERE uploads.path=p OR starts_with(uploads.path,p||'/'))`, paths); e != nil {
		return e
	}
	return nil
}

func (s *Server) purgeTree(ctx context.Context, t pgx.Tx, root string) error {
	// Active descendants indicate a changed namespace; reject instead of orphaning them.
	var active bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE (path=$1 OR starts_with(path,$1||'/')) AND trashed_at IS NULL)", root).Scan(&active); e != nil {
		return e
	}
	if active {
		return fail(409, "trash_changed", "entry is no longer in trash")
	}
	rows, e := t.Query(ctx, "SELECT path FROM paths WHERE path=$1 OR starts_with(path,$1||'/')", root)
	if e != nil {
		return e
	}
	paths := []string{}
	for rows.Next() {
		var p string
		if e = rows.Scan(&p); e != nil {
			rows.Close()
			return e
		}
		paths = append(paths, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	return s.purgePaths(ctx, t, paths)
}
