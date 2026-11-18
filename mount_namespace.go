package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func readonlyPath(ctx context.Context, t pgx.Tx, p string, tree bool) error {
	var overlap bool
	if err := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM mounts WHERE path=$1 OR starts_with($1,path||'/') OR ($2 AND starts_with(path,$1||'/')))", p, tree).Scan(&overlap); err != nil {
		return err
	}
	if overlap {
		return fail(409, "readonly_mount", "server folders are read-only")
	}
	return nil
}
func (s *Server) syncMounts(ctx context.Context, t pgx.Tx) error {
	// Configuration is a compatibility seed. PG administrative mappings are authoritative thereafter.
	for _, m := range s.cfg.Mounts {
		var exists bool
		if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM security_audit WHERE action='mount.seed' AND target_id=$1)", m.ID).Scan(&exists); e != nil {
			return e
		}
		if exists {
			continue
		}
		if _, e := t.Exec(ctx, "INSERT INTO mount_sources(id,name,node_id,export_id) VALUES($1,$2,$3,$4)", m.ID, m.Path, m.Node, m.Export); e != nil {
			return e
		}
		if e := s.addMountGrant(ctx, t, m.ID, "legacy", m.Path, m.ID); e != nil {
			return e
		}
		if e := audit(ctx, t, "", "mount.seed", m.ID); e != nil {
			return e
		}
	}
	return nil
}
func (s *Server) mountFor(ctx context.Context, p string) (MountConfig, bool) {
	var m MountConfig
	e := s.query(ctx).QueryRow(ctx, "SELECT id,path,node_id,export_id FROM mounts WHERE path=$1 OR starts_with($1,path||'/') ORDER BY length(path) DESC LIMIT 1", p).Scan(&m.ID, &m.Path, &m.Node, &m.Export)
	return m, e == nil
}

// Config selects allowed endpoints; PostgreSQL decides whether a mapping is still active.
// A process with stale configuration must not serve a removed mapping.
func currentMount(ctx context.Context, q queryRow, m MountConfig) error {
	var ok bool
	if e := q.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM mounts WHERE id=$1 AND path=$2 AND node_id=$3 AND export_id=$4)", m.ID, m.Path, m.Node, m.Export).Scan(&ok); e != nil {
		return e
	}
	if !ok {
		return fail(404, "not_found", "mount removed")
	}
	return nil
}
