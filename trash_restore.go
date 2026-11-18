package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"path"
	"strings"
)

// The caller holds the same transaction lock used by upload, purge and namespace operations.
func (s *Server) restoreTree(ctx context.Context, t pgx.Tx, source string) (string, error) {
	var deleted bool
	if e := t.QueryRow(ctx, "SELECT trashed_at IS NOT NULL FROM paths WHERE path=$1", source).Scan(&deleted); e != nil {
		return "", e
	}
	if !deleted {
		return source, nil
	}
	var expired bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE (path=$1 OR starts_with(path,$1||'/')) AND expires_at<=now())", source).Scan(&expired); e != nil {
		return "", e
	}
	if expired {
		return "", fail(410, "trash_expired", "restore window expired")
	}
	// Do not recreate deleted parents: find the nearest complete, active ancestor chain.
	parent, err := restoreParent(ctx, t, source)
	if err != nil {
		return "", err
	}
	target := path.Join(parent, path.Base(source))
	rows, e := t.Query(ctx, "SELECT path FROM paths WHERE path=$1 OR starts_with(path,$1||'/') ORDER BY path", source)
	if e != nil {
		return "", e
	}
	items := []string{}
	for rows.Next() {
		var p string
		if e = rows.Scan(&p); e != nil {
			rows.Close()
			return "", e
		}
		items = append(items, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return "", e
	}
	if target != source {
		base := target
		for n := 0; ; n++ {
			target = numberedPath(base, n)
			if !validPath(target) {
				return "", fail(409, "invalid_path", "restored path is too long")
			}
			var collision bool
			if e = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1 OR starts_with(path,$1||'/'))", target).Scan(&collision); e != nil {
				return "", e
			}
			if !collision {
				break
			}
		}
		for _, p := range items {
			dest := target + strings.TrimPrefix(p, source)
			if !validPath(dest) {
				return "", fail(409, "invalid_path", "restored path is too long")
			}
			if _, e = t.Exec(ctx, "UPDATE paths SET path=$2 WHERE path=$1", p, dest); e != nil {
				return "", e
			}
		}
		if _, e = t.Exec(ctx, "UPDATE uploads SET expires_at=now() WHERE NOT committed AND (path=$1 OR starts_with(path,$1||'/'))", source); e != nil {
			return "", e
		}
	}
	_, e = t.Exec(ctx, "UPDATE paths SET trashed_at=NULL,expires_at=NULL,trash_id=NULL WHERE path=$1 OR starts_with(path,$1||'/')", target)
	return target, e
}
