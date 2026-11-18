package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// A failed plan must not commit any upload, generation or reservation changes.
// Only the bounded reclamation schedule survives its savepoint rollback.
func (s *Server) planUpload(ctx context.Context, plan func(pgx.Tx) error) (*apiError, error) {
	var capacity *apiError
	err := s.tx(ctx, func(t pgx.Tx) error {
		savepoint, err := t.Begin(ctx)
		if err != nil {
			return err
		}
		if err = plan(savepoint); err == nil {
			return savepoint.Commit(ctx)
		}
		if rollback := savepoint.Rollback(ctx); rollback != nil {
			return rollback
		}
		var api *apiError
		if !errors.As(err, &api) || api.Code != "insufficient_capacity" {
			return err
		}
		capacity = api
		pool, err := s.routingPool(ctx, t)
		if err != nil {
			return err
		}
		fresh := 0
		for _, n := range pool {
			if n.fresh && !n.resource.Storage.ReadOnly {
				fresh++
			}
		}
		// Missing/offline/read-only nodes are not evidence of space exhaustion.
		if fresh < s.cfg.Replicas {
			return nil
		}
		// Internal GC needs the global reference view, not this caller's namespace.
		// It exposes no records and can only schedule already unreferenced blocks.
		var owner string
		if err = t.QueryRow(ctx, "SELECT coalesce(current_setting('yuki.owner',true),'')").Scan(&owner); err != nil {
			return err
		}
		if _, err = t.Exec(ctx, "SELECT set_config('yuki.owner','',true)"); err != nil {
			return err
		}
		if err = s.reclaimCapacity(ctx, t); err != nil {
			return err
		}
		_, err = t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", owner)
		return err
	})
	return capacity, err
}

// Expediting grace never counts as free capacity. Only a later signed resource
// report after fenced physical deletion may admit another upload.
func (s *Server) reclaimCapacity(ctx context.Context, t pgx.Tx) error {
	rows, err := t.Query(ctx, `WITH active_upload_blocks AS MATERIALIZED (
 SELECT DISTINCT chunk->>'hash' AS hash FROM uploads u CROSS JOIN LATERAL jsonb_array_elements(u.manifest->'chunks') chunk
 WHERE NOT u.committed AND u.expires_at>now()
 ) SELECT b.hash,b.generation FROM blocks b
 WHERE b.state IN ('reclaimable','available') AND b.refs=0
 AND NOT EXISTS(SELECT 1 FROM version_blocks v WHERE v.hash=b.hash)
 AND NOT EXISTS(SELECT 1 FROM active_upload_blocks u WHERE u.hash=b.hash)
 AND NOT EXISTS(SELECT 1 FROM placement_reservations p WHERE p.hash=b.hash AND p.expires_at>now()
   AND NOT EXISTS(SELECT 1 FROM replicas r WHERE r.hash=p.hash AND r.node_id=p.node_id AND r.generation=p.generation))
 AND NOT EXISTS(SELECT 1 FROM tasks q WHERE q.hash=b.hash AND q.kind='copy' AND q.state IN ('queued','running'))
 AND NOT EXISTS(SELECT 1 FROM tasks q WHERE q.hash=b.hash AND q.kind='delete' AND q.payload->>'reason'='capacity_gc' AND q.state IN ('queued','running'))
 ORDER BY b.gc_after,b.hash LIMIT 256 FOR UPDATE OF b`)
	if err != nil {
		return err
	}
	type candidate struct {
		hash       string
		generation int64
	}
	items := []candidate{}
	for rows.Next() {
		var v candidate
		if err = rows.Scan(&v.hash, &v.generation); err != nil {
			rows.Close()
			return err
		}
		items = append(items, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, v := range items {
		if _, err = t.Exec(ctx, "UPDATE blocks SET state='reclaimable',gc_after=now() WHERE hash=$1", v.hash); err != nil {
			return err
		}
		if err = s.scheduleBlockGC(ctx, t, v.hash, v.generation, "capacity_gc"); err != nil {
			return err
		}
		if _, err = t.Exec(ctx, "UPDATE tasks SET payload=jsonb_set(payload,'{reason}','\"capacity_gc\"') WHERE hash=$1 AND generation=$2 AND kind='delete' AND state='queued' AND payload->>'reason'='gc'", v.hash, v.generation); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) scheduleBlockGC(ctx context.Context, t pgx.Tx, hash string, generation int64, reason string) error {
	rows, err := t.Query(ctx, "SELECT node_id FROM replicas WHERE hash=$1", hash)
	if err != nil {
		return err
	}
	nodes := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		nodes = append(nodes, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range nodes {
		if err = s.enqueue(ctx, t, "delete", hash, id, generation, map[string]any{"reason": reason}); err != nil {
			return err
		}
	}
	if len(nodes) == 0 {
		_, err = t.Exec(ctx, "UPDATE blocks SET state='deleted' WHERE hash=$1", hash)
	}
	return err
}
