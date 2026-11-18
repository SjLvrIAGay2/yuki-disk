package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// Refresh an enqueued copy's source and validate destination capacity at claim time.
// A temporarily unsuitable task yields the queue without consuming a retry attempt.
func (s *Server) routeCopy(ctx context.Context, t pgx.Tx, a *Task) (bool, error) {
	pool, e := s.routingPool(ctx, t)
	if e != nil {
		return false, e
	}
	var target *routingNode
	for _, n := range pool {
		if n.id == a.Node {
			target = n
			break
		}
	}
	if target == nil || !target.fresh || target.resource.Storage.ReadOnly || target.available < target.floor() {
		return false, nil
	}
	rows, e := t.Query(ctx, "SELECT node_id FROM replicas WHERE hash=$1 AND generation=$2 AND node_id<>$3", a.Hash, a.Generation, a.Node)
	if e != nil {
		return false, e
	}
	have := map[string]bool{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return false, e
		}
		have[id] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return false, e
	}
	sources := []*routingNode{}
	for _, n := range pool {
		if have[n.id] {
			sources = append(sources, n)
		}
	}
	if len(sources) == 0 {
		return false, nil
	}
	size, _ := a.Payload["size"].(float64)
	rankNodes(sources, int64(size), true, "")
	source := sources[0]
	a.Payload["source_node"], a.Payload["source_url"] = source.id, source.url
	if _, e = t.Exec(ctx, "UPDATE tasks SET payload=$2 WHERE id=$1", a.ID, a.Payload); e != nil {
		return false, e
	}
	if e = reservePlacement(ctx, t, pool, target, a.Hash, a.Generation, int64(size)); e != nil {
		return false, e
	}
	a.Payload["source_token"] = s.cap(source.id, "get", a.Hash, int64(size), a.Generation)
	return true, nil
}
