package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type routingBlockKey struct {
	hash       string
	generation int64
}
type rebalanceState struct {
	busy  bool
	nodes []string
}

// One authoritative transaction-local snapshot; the global serialization lock
// remains held while planning. Each block/generation is visited only once.
func loadRebalanceState(ctx context.Context, t pgx.Tx) (map[routingBlockKey]rebalanceState, error) {
	state := map[routingBlockKey]rebalanceState{}
	rows, err := t.Query(ctx, "SELECT hash,generation,node_id FROM replicas")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key routingBlockKey
		var node string
		if err = rows.Scan(&key.hash, &key.generation, &node); err != nil {
			rows.Close()
			return nil, err
		}
		v := state[key]
		v.nodes = append(v.nodes, node)
		state[key] = v
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = t.Query(ctx, `SELECT DISTINCT hash,generation FROM tasks WHERE hash IS NOT NULL AND (state IN ('queued','running') OR (kind='copy' AND state='failed' AND finished_at>now()-interval '5 minutes'))`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key routingBlockKey
		if err = rows.Scan(&key.hash, &key.generation); err != nil {
			rows.Close()
			return nil, err
		}
		v := state[key]
		v.busy = true
		state[key] = v
	}
	err = rows.Err()
	rows.Close()
	return state, err
}
