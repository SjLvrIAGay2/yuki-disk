package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Server) sweep(ctx context.Context, t pgx.Tx) error {
	if e := s.purgeDeletedAccounts(ctx, t); e != nil {
		return e
	}
	if e := s.expireVersions(ctx, t); e != nil {
		return e
	}
	rows, e := t.Query(ctx, "SELECT owner_id,path FROM paths WHERE expires_at<=now() AND NOT EXISTS(SELECT 1 FROM accounts a WHERE a.id=paths.owner_id AND a.deleted_at IS NOT NULL)")
	if e != nil {
		return e
	}
	expired := map[string][]string{}
	for rows.Next() {
		var owner, p string
		if e = rows.Scan(&owner, &p); e != nil {
			rows.Close()
			return e
		}
		expired[owner] = append(expired[owner], p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for owner, paths := range expired {
		if _, e = t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", owner); e != nil {
			return e
		}
		if e = s.purgePaths(ctx, t, paths); e != nil {
			return e
		}
	}
	if _, e = t.Exec(ctx, "SELECT set_config('yuki.owner','',true)"); e != nil {
		return e
	}
	queries := []string{
		"DELETE FROM auth_attempts WHERE expires_at<now()",
		"DELETE FROM sessions WHERE expires_at<now()-interval '1 day' OR revoked_at<now()-interval '1 day'",
		"UPDATE tasks t SET state='done',finished_at=now(),error='obsolete copy' WHERE kind='copy' AND state='queued' AND NOT EXISTS(SELECT 1 FROM blocks b WHERE b.hash=t.hash AND b.generation=t.generation AND b.state='available')",
		"UPDATE tasks SET state='failed',finished_at=now(),error='copy queue deadline exceeded' WHERE kind='copy' AND state='queued' AND created_at<now()-interval '15 minutes'",
		"DELETE FROM placement_reservations p WHERE expires_at<=now() AND NOT EXISTS(SELECT 1 FROM tasks t WHERE t.hash=p.hash AND t.node_id=p.node_id AND t.generation=p.generation AND t.kind='copy' AND t.state IN ('queued','running'))",
		"DELETE FROM inventory_sessions WHERE updated_at<now()-interval '1 day'",
		"DELETE FROM operations WHERE created_at<now()-interval '7 days'",
		"UPDATE blocks b SET refs=(SELECT count(*) FROM version_blocks v WHERE v.hash=b.hash) WHERE refs<>(SELECT count(*) FROM version_blocks v WHERE v.hash=b.hash)",
		"DELETE FROM replay_nonces WHERE expires_at<now()",
		"DELETE FROM uploads WHERE expires_at<now()-interval '1 day' AND NOT EXISTS(SELECT 1 FROM versions v WHERE v.id=uploads.id)",
		"UPDATE tasks SET state=CASE WHEN attempts>=4 THEN 'failed' ELSE 'queued' END,available_at=now()+make_interval(secs=>power(2,least(attempts,6))::int),finished_at=CASE WHEN attempts>=4 THEN now() ELSE NULL END,error='lease expired' WHERE state='running' AND lease_until<now()",
		"DELETE FROM tasks WHERE state IN ('done','failed') AND (finished_at<now()-interval '7 days' OR id<(SELECT coalesce(max(id),0)-10000 FROM tasks))",
		"DELETE FROM events WHERE created_at<now()-interval '30 days' OR id<(SELECT coalesce(max(id),0)-10000 FROM events)",
	}
	for _, q := range queries {
		if _, e := t.Exec(ctx, q); e != nil {
			return e
		}
	}
	if _, e := t.Exec(ctx, "UPDATE blocks SET state='reclaimable',gc_after=now()+make_interval(secs=>$1) WHERE refs=0 AND state IN ('uploading','available') AND coalesce(upload_until,'-infinity')<now()", s.cfg.GCSeconds); e != nil {
		return e
	}
	pool, e := s.routingPool(ctx, t)
	if e != nil {
		return e
	}
	fresh, writable := 0, 0
	for _, n := range pool {
		if n.fresh && !n.resource.Storage.ReadOnly {
			fresh++
		}
		if n.writable(chunkSize) {
			writable++
		}
	}
	if fresh >= s.cfg.Replicas && writable < s.cfg.Replicas {
		if e = s.reclaimCapacity(ctx, t); e != nil {
			return e
		}
	}
	rows, e = t.Query(ctx, "SELECT hash,size,generation,state,gc_after,last_access,heat,heat_at,last_move_at FROM blocks WHERE state<>'deleted' ORDER BY hash")
	if e != nil {
		return e
	}
	type block struct {
		h       string
		size, g int64
		state   string
		gc      *time.Time
		access  time.Time
		heat    float64
		heatAt  time.Time
		moved   *time.Time
	}
	all := []block{}
	for rows.Next() {
		var a block
		if e = rows.Scan(&a.h, &a.size, &a.g, &a.state, &a.gc, &a.access, &a.heat, &a.heatAt, &a.moved); e != nil {
			rows.Close()
			return e
		}
		all = append(all, a)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	state, e := loadRebalanceState(ctx, t)
	if e != nil {
		return e
	}
	for _, a := range all {
		if a.state == "reclaimable" && a.gc != nil && a.gc.Before(time.Now()) {
			if e = s.scheduleBlockGC(ctx, t, a.h, a.g, "gc"); e != nil {
				return e
			}
			continue
		}
		if a.state != "available" {
			continue
		}
		if e = s.rebalanceBlock(ctx, t, pool, routingBlock{a.h, a.size, a.g, a.access, a.heatAt, a.heat, a.moved}, state[routingBlockKey{a.h, a.g}]); e != nil {
			return e
		}
	}
	return nil
}
