package main

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

type DiskItem struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Generation int64  `json:"generation"`
}

type Report struct {
	Items []DiskItem `json:"items"`
	Full  bool       `json:"full"`
}

func (s *Server) report(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "forbidden", "node identity required")
	}
	node := strings.TrimPrefix(identity, "node:")
	var p Report
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Items) > 10000 {
		return fail(400, "inventory_too_large", "inventory exceeds deployment limit")
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		if _, e := t.Exec(ctx, "UPDATE nodes SET last_seen=now(),inventory_at=CASE WHEN $2 THEN now() ELSE inventory_at END WHERE id=$1", node, p.Full); e != nil {
			return e
		}
		// Full inventories are snapshots serialized on the node against writes/deletes.
		if p.Full {
			if _, e := t.Exec(ctx, "DELETE FROM replicas WHERE node_id=$1", node); e != nil {
				return e
			}
		}
		for _, a := range p.Items {
			if !validHash(a.Hash) || a.Size <= 0 || a.Size > chunkSize || a.Generation < 1 {
				return fail(400, "invalid_inventory", "invalid disk entry")
			}
			var state string
			var size, g int64
			e := t.QueryRow(ctx, "SELECT state,size,generation FROM blocks WHERE hash=$1", a.Hash).Scan(&state, &size, &g)
			if e == pgx.ErrNoRows {
				_, e = t.Exec(ctx, "INSERT INTO blocks(hash,size,state,generation,gc_after) VALUES($1,$2,'reclaimable',$3,now()+make_interval(secs=>$4))", a.Hash, a.Size, a.Generation, s.cfg.GCSeconds)
				if e != nil {
					return e
				}
				state = "reclaimable"
				size = a.Size
				g = a.Generation
				if e = event(ctx, t, "orphan_discovered", a.Hash, map[string]any{"node": node}); e != nil {
					return e
				}
			} else if e != nil {
				return e
			}
			if size != a.Size {
				return fail(409, "inventory_conflict", "disk size differs")
			}
			if a.Generation != g || state == "deleted" {
				if e = s.enqueue(ctx, t, "delete", a.Hash, node, a.Generation, map[string]any{"reason": "stale_generation"}); e != nil {
					return e
				}
				continue
			}
			if _, e = t.Exec(ctx, "INSERT INTO replicas(hash,node_id,generation) VALUES($1,$2,$3) ON CONFLICT(hash,node_id) DO UPDATE SET generation=excluded.generation,confirmed_at=now()", a.Hash, node, g); e != nil {
				return e
			}
			if state == "uploading" {
				n, e := s.healthy(ctx, t, a.Hash, g)
				if e != nil {
					return e
				}
				if n >= s.cfg.Replicas {
					if _, e = t.Exec(ctx, "UPDATE blocks SET state='available' WHERE hash=$1", a.Hash); e != nil {
						return e
					}
					if e = event(ctx, t, "block_available", a.Hash, map[string]any{"replicas": n}); e != nil {
						return e
					}
				}
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]bool{"accepted": true})
	return nil
}
