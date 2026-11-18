package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

func (s *Server) enqueue(ctx context.Context, t pgx.Tx, kind, hash, node string, g int64, p any) error {
	_, e := t.Exec(ctx, `INSERT INTO tasks(kind,hash,node_id,generation,payload) SELECT $1,NULLIF($2,''),$3,$4,$5 WHERE NOT EXISTS(SELECT 1 FROM tasks WHERE kind=$1 AND coalesce(hash,'')=$2 AND node_id=$3 AND generation=$4 AND (state IN ('queued','running') OR (state='failed' AND finished_at>now()-interval '5 minutes'))) ON CONFLICT DO NOTHING`, kind, hash, node, g, p)
	return e
}

type Task struct {
	ID         int64          `json:"id"`
	Kind       string         `json:"kind"`
	Hash       string         `json:"hash"`
	Node       string         `json:"node"`
	URL        string         `json:"url"`
	Generation int64          `json:"generation"`
	Fence      int64          `json:"fence"`
	Owner      string         `json:"owner"`
	Payload    map[string]any `json:"payload"`
}

func (s *Server) claim(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if identity != "maintenance" {
		return fail(403, "forbidden", "maintenance identity required")
	}
	var p struct {
		Owner string `json:"owner"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Owner) < 16 || len(p.Owner) > 128 {
		return fail(400, "invalid_owner", "owner length")
	}
	var task *Task
	var fence int64
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var owner string
		var active bool
		if e := t.QueryRow(ctx, "SELECT owner,fence,expires_at>now() FROM leader FOR UPDATE").Scan(&owner, &fence, &active); e != nil {
			return e
		}
		if active && owner != p.Owner {
			return fail(409, "leader_busy", "another maintainer holds lease")
		}
		if !active || owner != p.Owner {
			fence++
		}
		if _, e := t.Exec(ctx, "UPDATE leader SET owner=$1,fence=$2,expires_at=now()+interval '20 seconds'", p.Owner, fence); e != nil {
			return e
		}
		if e := s.sweep(ctx, t); e != nil {
			return e
		}
		var running int
		if e := t.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE state='running' AND lease_until>now()").Scan(&running); e != nil {
			return e
		}
		if running > 0 {
			return nil
		}
		a := &Task{Fence: fence, Owner: p.Owner}
		var raw []byte
		e := t.QueryRow(ctx, "SELECT t.id,t.kind,coalesce(t.hash,''),t.node_id,n.url,t.generation,t.payload FROM tasks t JOIN nodes n ON n.id=t.node_id WHERE t.state='queued' AND t.available_at<=now() ORDER BY CASE WHEN t.kind='delete' AND t.payload->>'reason'='capacity_gc' THEN 0 ELSE 1 END,t.id LIMIT 1 FOR UPDATE OF t").Scan(&a.ID, &a.Kind, &a.Hash, &a.Node, &a.URL, &a.Generation, &raw)
		if e == pgx.ErrNoRows {
			return nil
		}
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &a.Payload); e != nil {
			return e
		}
		if a.Kind == "copy" {
			ready, err := s.routeCopy(ctx, t, a)
			if err != nil {
				return err
			}
			if !ready {
				_, err = t.Exec(ctx, "UPDATE tasks SET available_at=now()+interval '3 seconds' WHERE id=$1", a.ID)
				return err
			}
		}
		if _, e = t.Exec(ctx, "UPDATE tasks SET state='running',owner=$2,fence=$3,attempts=attempts+1,lease_until=now()+interval '15 seconds' WHERE id=$1", a.ID, p.Owner, fence); e != nil {
			return e
		}
		task = a
		return nil
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"fence": fence, "task": task})
	return nil
}

type Result struct {
	ID    int64  `json:"id"`
	Owner string `json:"owner"`
	Fence int64  `json:"fence"`
	Error string `json:"error"`
}

func (s *Server) validateTask(ctx context.Context, t pgx.Tx, p Result) (Task, error) {
	var a Task
	var raw []byte
	e := t.QueryRow(ctx, "SELECT id,kind,coalesce(hash,''),node_id,generation,payload FROM tasks WHERE id=$1 AND owner=$2 AND fence=$3 AND state='running' AND lease_until>now() AND EXISTS(SELECT 1 FROM leader WHERE owner=$2 AND fence=$3 AND expires_at>now()) FOR UPDATE", p.ID, p.Owner, p.Fence).Scan(&a.ID, &a.Kind, &a.Hash, &a.Node, &a.Generation, &raw)
	if e == pgx.ErrNoRows {
		return a, fail(409, "stale_executor", "lease or fence is stale")
	}
	if e != nil {
		return a, e
	}
	e = json.Unmarshal(raw, &a.Payload)
	return a, e
}

func (s *Server) permit(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "forbidden", "node required")
	}
	var p Result
	if e := parse(b, &p); e != nil {
		return e
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		a, e := s.validateTask(r.Context(), t, p)
		if e != nil {
			return e
		}
		if identity != "node:"+a.Node {
			return fail(403, "wrong_node", "task targets another node")
		}
		var g int64
		var state string
		var refs int64
		if e = t.QueryRow(r.Context(), "SELECT generation,state,refs FROM blocks WHERE hash=$1", a.Hash).Scan(&g, &state, &refs); e != nil {
			return e
		}
		if a.Kind == "delete" && a.Generation == g && state != "reclaimable" && state != "deleted" {
			reason, _ := a.Payload["reason"].(string)
			n, e := s.healthy(r.Context(), t, a.Hash, g)
			if e != nil {
				return e
			}
			if reason != "migration_trim" || n <= s.cfg.Replicas {
				return fail(409, "delete_protected", "block is protected")
			}
		}
		if a.Kind == "copy" && (g != a.Generation || state != "available") {
			return fail(409, "copy_obsolete", "block generation changed")
		}
		return nil
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]bool{"allowed": true})
	return nil
}

func (s *Server) complete(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if identity != "maintenance" {
		return fail(403, "forbidden", "maintenance required")
	}
	var p Result
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Error) > 1024 {
		p.Error = p.Error[:1024]
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		a, e := s.validateTask(ctx, t, p)
		if e != nil {
			return e
		}
		if p.Error != "" {
			_, e = t.Exec(ctx, "UPDATE tasks SET state=CASE WHEN attempts>=4 THEN 'failed' ELSE 'queued' END,available_at=now()+make_interval(secs=>power(2,attempts)::int),finished_at=CASE WHEN attempts>=4 THEN now() ELSE NULL END,error=$2 WHERE id=$1", p.ID, p.Error)
			return e
		}
		if a.Kind == "delete" {
			if _, e = t.Exec(ctx, "DELETE FROM replicas WHERE hash=$1 AND node_id=$2 AND generation<=$3", a.Hash, a.Node, a.Generation); e != nil {
				return e
			}
			if a.Payload["reason"] != "migration_trim" {
				// Physical GC has persisted a tombstone for these generations.
				// Their old upload reservations must not mask reclaimed space.
				if _, e = t.Exec(ctx, "DELETE FROM placement_reservations WHERE hash=$1 AND node_id=$2 AND generation<=$3", a.Hash, a.Node, a.Generation); e != nil {
					return e
				}
			}
		} else if a.Kind == "copy" {
			tag, err := t.Exec(ctx, "INSERT INTO replicas(hash,node_id,generation) SELECT hash,$2,generation FROM blocks WHERE hash=$1 AND generation=$3 AND state='available' ON CONFLICT(hash,node_id) DO UPDATE SET generation=excluded.generation,confirmed_at=now()", a.Hash, a.Node, a.Generation)
			if err != nil {
				return err
			}
			if replaced, ok := a.Payload["replace_node"].(string); tag.RowsAffected() > 0 && ok && replaced != "" {
				if e = s.enqueue(ctx, t, "delete", a.Hash, replaced, a.Generation, map[string]any{"reason": "migration_trim"}); e != nil {
					return e
				}
			}
		}
		if _, e = t.Exec(ctx, "UPDATE tasks SET state='done',finished_at=now(),error='' WHERE id=$1", p.ID); e != nil {
			return e
		}
		return event(ctx, t, a.Kind+"_completed", a.Hash, map[string]any{"node": a.Node, "generation": a.Generation, "reason": a.Payload["reason"]})
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]bool{"accepted": true})
	return nil
}
