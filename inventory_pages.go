package main

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

type InventoryPage struct {
	Snapshot string     `json:"snapshot"`
	Page     int        `json:"page"`
	Final    bool       `json:"final"`
	Items    []DiskItem `json:"items"`
}

func (s *Server) inventoryPage(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "forbidden", "node identity required")
	}
	node := strings.TrimPrefix(identity, "node:")
	var p InventoryPage
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Snapshot) != 48 || p.Page < 0 || len(p.Items) > 1000 {
		return fail(400, "invalid_inventory_page", "invalid snapshot or page")
	}
	for _, a := range p.Items {
		if !validHash(a.Hash) || a.Size <= 0 || a.Size > chunkSize || a.Generation < 1 {
			return fail(400, "invalid_inventory", "invalid disk entry")
		}
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var snapshot, prior string
		var next int
		var completed bool
		e := t.QueryRow(ctx, "SELECT snapshot,next_page,last_digest,completed FROM inventory_sessions WHERE node_id=$1 FOR UPDATE", node).Scan(&snapshot, &next, &prior, &completed)
		if e != nil && e != pgx.ErrNoRows {
			return e
		}
		if e == nil && snapshot == p.Snapshot && next == p.Page+1 && prior == digest(b) {
			return nil
		}
		if p.Page == 0 && snapshot != p.Snapshot {
			if _, e = t.Exec(ctx, "DELETE FROM inventory_sessions WHERE node_id=$1", node); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "INSERT INTO inventory_sessions(node_id,snapshot) VALUES($1,$2)", node, p.Snapshot); e != nil {
				return e
			}
			snapshot = p.Snapshot
			next = 0
			completed = false
		}
		if snapshot != p.Snapshot || next != p.Page || completed {
			return fail(409, "inventory_sequence", "restart full snapshot")
		}
		values := make([][]any, 0, len(p.Items))
		for _, item := range p.Items {
			values = append(values, []any{node, item.Hash, item.Size, item.Generation})
		}
		if len(values) > 0 {
			if _, e = t.CopyFrom(ctx, pgx.Identifier{"inventory_entries"}, []string{"node_id", "hash", "size", "generation"}, pgx.CopyFromRows(values)); e != nil {
				return e
			}
		}
		if p.Final {
			var bad bool
			if e = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM inventory_entries i JOIN blocks b USING(hash) WHERE node_id=$1 AND i.size<>b.size)", node).Scan(&bad); e != nil {
				return e
			}
			if bad {
				return fail(409, "inventory_conflict", "disk size differs")
			}
			if _, e = t.Exec(ctx, `WITH added AS (
    INSERT INTO blocks(hash,size,state,generation,gc_after)
    SELECT DISTINCT ON(hash) hash,size,'reclaimable',generation,now()+make_interval(secs=>$2)
    FROM inventory_entries WHERE node_id=$1 ORDER BY hash,generation DESC ON CONFLICT DO NOTHING RETURNING hash)
    INSERT INTO events(kind,hash,detail) SELECT 'orphan_discovered',hash,jsonb_build_object('node',$1::text) FROM added`, node, s.cfg.GCSeconds); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, `INSERT INTO tasks(kind,hash,node_id,generation,payload)
    SELECT 'delete',i.hash,i.node_id,i.generation,'{"reason":"stale_generation"}'::jsonb
    FROM inventory_entries i JOIN blocks b USING(hash) WHERE i.node_id=$1 AND (i.generation<>b.generation OR b.state='deleted')
    ON CONFLICT DO NOTHING`, node); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "DELETE FROM replicas WHERE node_id=$1", node); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, `INSERT INTO replicas(hash,node_id,generation)
    SELECT i.hash,i.node_id,i.generation FROM inventory_entries i JOIN blocks b USING(hash)
    WHERE i.node_id=$1 AND i.generation=b.generation AND b.state<>'deleted'`, node); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "UPDATE nodes SET last_seen=now(),inventory_at=now() WHERE id=$1", node); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, `WITH ready AS (
    UPDATE blocks b SET state='available' WHERE state='uploading' AND
    (SELECT count(*) FROM replicas rp JOIN nodes n ON n.id=rp.node_id WHERE rp.hash=b.hash AND rp.generation=b.generation AND n.last_seen>now()-interval '30 seconds') >= $1
    RETURNING hash) INSERT INTO events(kind,hash,detail) SELECT 'block_available',hash,jsonb_build_object('replicas',$1::int) FROM ready`, s.cfg.Replicas); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "DELETE FROM inventory_entries WHERE node_id=$1", node); e != nil {
				return e
			}
		}
		_, e = t.Exec(ctx, "UPDATE inventory_sessions SET next_page=$2,last_digest=$3,completed=$4,updated_at=now() WHERE node_id=$1", node, p.Page+1, digest(b), p.Final)
		return e
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]bool{"accepted": true})
	return nil
}
