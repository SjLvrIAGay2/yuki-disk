package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"path"
)

func (s *Server) versionRestorePath(ctx context.Context, t pgx.Tx, source string) (string, error) {
	parent, err := restoreParent(ctx, t, source)
	if err != nil {
		return "", err
	}
	base := path.Join(parent, path.Base(source))
	target := base
	for n := 0; ; n++ {
		target = numberedPath(base, n)
		if !validPath(target) {
			return "", fail(409, "invalid_path", "restored path too long")
		}
		var exists, merge bool
		if err := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1),EXISTS(SELECT 1 FROM paths WHERE path=$1 AND NOT is_dir AND trashed_at IS NULL)", target).Scan(&exists, &merge); err != nil {
			return "", err
		}
		if !exists || (target == source && merge) {
			if err := readonlyPath(ctx, t, target, false); err != nil {
				return "", err
			}
			return target, nil
		}
	}
}
func (s *Server) purgeVersions(ctx context.Context, t pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := t.Query(ctx, "SELECT DISTINCT hash FROM version_blocks WHERE version_id=ANY($1)", ids)
	if err != nil {
		return err
	}
	hashes := []string{}
	for rows.Next() {
		var h string
		if err = rows.Scan(&h); err != nil {
			rows.Close()
			return err
		}
		hashes = append(hashes, h)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if _, err = t.Exec(ctx, "DELETE FROM versions WHERE id=ANY($1)", ids); err != nil {
		return err
	}
	if _, err = t.Exec(ctx, "UPDATE blocks b SET refs=(SELECT count(*) FROM version_blocks v WHERE v.hash=b.hash) WHERE hash=ANY($1)", hashes); err != nil {
		return err
	}
	_, err = t.Exec(ctx, "UPDATE blocks SET state='reclaimable',gc_after=now()+make_interval(secs=>$2) WHERE hash=ANY($1) AND refs=0 AND state IN ('uploading','available') AND coalesce(upload_until,'-infinity')<now()", hashes, s.cfg.GCSeconds)
	return err
}
func (s *Server) expireVersions(ctx context.Context, t pgx.Tx) error {
	rows, err := t.Query(ctx, "SELECT id FROM versions WHERE path IS NULL AND expires_at<=now() AND NOT EXISTS(SELECT 1 FROM accounts a WHERE a.id=versions.owner_id AND a.deleted_at IS NOT NULL)")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	return s.purgeVersions(ctx, t, ids)
}
