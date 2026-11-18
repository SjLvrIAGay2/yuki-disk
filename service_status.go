package main

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) serviceStatus(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows, err := s.db.Query(ctx, `SELECT n.id,n.url,n.tier,n.last_seen,n.inventory_at,
 (SELECT count(*) FROM replicas rp WHERE rp.node_id=n.id)
 FROM nodes n ORDER BY n.id`)
	if err != nil {
		return err
	}
	nodes := []map[string]any{}
	for rows.Next() {
		var id, url, tier string
		var seen, inventory *time.Time
		var replicas int64
		if err = rows.Scan(&id, &url, &tier, &seen, &inventory, &replicas); err != nil {
			rows.Close()
			return err
		}
		nodes = append(nodes, map[string]any{"id": id, "url": url, "tier": tier, "last_seen": seen, "inventory_at": inventory, "replicas": replicas})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	scheduling, err := s.routingStatus(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		for key, value := range scheduling[node["id"].(string)] {
			node[key] = value
		}
	}
	var blocks, versions, queued, running, failed int64
	err = s.db.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM blocks WHERE state<>'deleted'),
 (SELECT count(*) FROM versions),
 (SELECT count(*) FROM tasks WHERE state='queued'),
 (SELECT count(*) FROM tasks WHERE state='running'),
 (SELECT count(*) FROM tasks WHERE state='failed')`).Scan(&blocks, &versions, &queued, &running, &failed)
	if err != nil {
		return err
	}
	pool := s.db.Stat()
	jsonOut(w, 200, map[string]any{"version": 1, "nodes": nodes, "control": map[string]any{"blocks": blocks, "versions": versions, "queued_tasks": queued, "running_tasks": running, "failed_tasks": failed, "database_connections": pool.TotalConns(), "database_in_use": pool.AcquiredConns(), "database_limit": pool.MaxConns(), "replication_factor": s.cfg.Replicas}})
	return nil
}
