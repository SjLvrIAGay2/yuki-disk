package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func (s *Server) routingStatus(ctx context.Context) (map[string]map[string]any, error) {
	t, e := s.db.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if e != nil {
		return nil, e
	}
	defer t.Rollback(context.Background())
	pool, e := s.routingPool(ctx, t)
	if e != nil {
		return nil, e
	}
	result := map[string]map[string]any{}
	for _, n := range pool {
		state := "ready"
		if !n.fresh {
			state = "unavailable"
		} else if n.resource.Storage.ReadOnly {
			state = "read_only"
		} else if !n.writable(chunkSize) {
			state = "full"
		}
		data := map[string]any{"routing_state": state, "read_bandwidth_bps": n.readBudget, "write_bandwidth_bps": n.writeBudget}
		if n.fresh {
			data["effective_read_bps"] = bandwidth(n.readBudget, n.resource.ReadSpeed)
			data["effective_write_bps"] = bandwidth(n.writeBudget, n.resource.WriteSpeed)
			data["receive_bps"] = n.resource.ReceiveRate
			data["send_bps"] = n.resource.SendRate
			data["reserved_bytes"] = n.reserved
			data["schedulable_bytes"] = max(0, n.available-n.floor())
			data["resource_at"] = n.resource.SampledAt
		}
		result[n.id] = data
	}
	return result, nil
}
