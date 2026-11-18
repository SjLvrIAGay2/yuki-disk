package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func (s *Server) uploadTargets(ctx context.Context, t pgx.Tx, pool []*routingNode, hash string, size, generation int64, needed int) ([]Target, error) {
	rows, e := t.Query(ctx, `SELECT n.id,EXISTS(SELECT 1 FROM replicas r WHERE r.hash=$1 AND r.generation=$2 AND r.node_id=n.id),
 EXISTS(SELECT 1 FROM placement_reservations p WHERE p.hash=$1 AND p.generation=$2 AND p.node_id=n.id AND p.expires_at>now()) FROM nodes n`, hash, generation)
	if e != nil {
		return nil, e
	}
	have := map[string]bool{}
	reserved := map[string]bool{}
	for rows.Next() {
		var id string
		var exists, pending bool
		if e = rows.Scan(&id, &exists, &pending); e != nil {
			rows.Close()
			return nil, e
		}
		have[id] = exists
		reserved[id] = pending
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	selected := []*routingNode{}
	chosen := map[string]bool{}
	// Reissued capabilities keep their reservation; pending bytes were already deducted.
	for _, n := range pool {
		if len(selected) >= needed {
			break
		}
		if !have[n.id] && reserved[n.id] && n.fresh && !n.resource.Storage.ReadOnly && n.available >= n.floor() {
			selected = append(selected, n)
			chosen[n.id] = true
		}
	}
	candidates := []*routingNode{}
	for _, n := range pool {
		if !have[n.id] && !chosen[n.id] {
			candidates = append(candidates, n)
		}
	}
	rankNodes(candidates, size, false, "hot")
	for _, n := range candidates {
		if len(selected) >= needed {
			break
		}
		if !n.writable(size) {
			continue
		}
		if e = reservePlacement(ctx, t, pool, n, hash, generation, size); e != nil {
			return nil, e
		}
		selected = append(selected, n)
		chosen[n.id] = true
	}
	if len(selected) < needed {
		return nil, fail(503, "insufficient_capacity", "not enough fresh writable node capacity")
	}
	targets := []Target{}
	for _, n := range selected {
		if reserved[n.id] {
			if e = reservePlacement(ctx, t, pool, n, hash, generation, size); e != nil {
				return nil, e
			}
		}
		targets = append(targets, Target{s.hasP2P(n.id), n.id, n.url + "/v1/blocks/" + hash, s.cap(n.id, "put", hash, size, generation)})
	}
	return targets, nil
}
