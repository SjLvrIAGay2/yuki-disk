package main

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
)

func (s *Server) block(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	hash := r.PathValue("hash")
	if !validHash(hash) {
		return fail(400, "invalid_hash", "invalid hash")
	}
	var state string
	var size, g, refs int64
	var gc *time.Time
	var heat float64
	var heatAt, access time.Time
	var moved *time.Time
	e := s.db.QueryRow(r.Context(), "SELECT state,size,generation,refs,gc_after,heat,heat_at,last_access,last_move_at FROM blocks WHERE hash=$1", hash).Scan(&state, &size, &g, &refs, &gc, &heat, &heatAt, &access, &moved)
	if e == pgx.ErrNoRows {
		return fail(404, "not_found", "block not found")
	}
	if e != nil {
		return e
	}
	rows, e := s.db.Query(r.Context(), "SELECT n.id,n.tier,r.generation,n.last_seen>now()-interval '30 seconds' FROM replicas r JOIN nodes n ON n.id=r.node_id WHERE r.hash=$1", hash)
	if e != nil {
		return e
	}
	defer rows.Close()
	replicas := []map[string]any{}
	healthy := 0
	for rows.Next() {
		var id, tier string
		var generation int64
		var live bool
		if e = rows.Scan(&id, &tier, &generation, &live); e != nil {
			return e
		}
		if live && generation == g {
			healthy++
		}
		replicas = append(replicas, map[string]any{"node": id, "tier": tier, "generation": generation, "live": live})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	heat = heatValue(heat, heatAt)
	temperature := "warm"
	if time.Since(access) > time.Duration(s.cfg.ColdSeconds)*time.Second {
		temperature = "cold"
	} else if heat >= 8 {
		temperature = "hot"
	}
	jsonOut(w, 200, map[string]any{"hash": hash, "size": size, "state": state, "generation": g, "refs": refs, "gc_after": gc, "healthy": healthy >= s.cfg.Replicas, "replicas": replicas, "heat": heat, "temperature": temperature, "last_move_at": moved})
	return nil
}

func (s *Server) events(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	limit, after, e := pagination(r)
	if e != nil {
		return e
	}
	rows, e := s.db.Query(r.Context(), "SELECT id,kind,coalesce(hash,''),detail,created_at FROM events WHERE id>$1 ORDER BY id LIMIT $2", after, limit)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var kind, hash string
		var detail any
		var created time.Time
		if e = rows.Scan(&id, &kind, &hash, &detail, &created); e != nil {
			return e
		}
		items = append(items, map[string]any{"id": id, "kind": kind, "hash": hash, "detail": detail, "created_at": created})
		after = id
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_after": after})
	return nil
}

func (s *Server) tasks(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	limit, after, e := pagination(r)
	if e != nil {
		return e
	}
	rows, e := s.db.Query(r.Context(), "SELECT id,kind,coalesce(hash,''),node_id,state,attempts,error FROM tasks WHERE id>$1 ORDER BY id LIMIT $2", after, limit)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var kind, hash, node, state, msg string
		var attempts int
		if e = rows.Scan(&id, &kind, &hash, &node, &state, &attempts, &msg); e != nil {
			return e
		}
		items = append(items, map[string]any{"id": id, "kind": kind, "hash": hash, "node": node, "state": state, "attempts": attempts, "error": msg})
		after = id
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_after": after})
	return nil
}
