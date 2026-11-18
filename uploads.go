package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
)

const chunkSize int64 = 4194304

type Chunk struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}
type Manifest struct {
	Path   string  `json:"path"`
	Size   int64   `json:"size"`
	Chunks []Chunk `json:"chunks"`
}
type Target struct {
	P2P   bool   `json:"p2p,omitempty"`
	Node  string `json:"node"`
	URL   string `json:"url"`
	Token string `json:"token"`
}
type Plan struct {
	Hash    string   `json:"hash"`
	Size    int64    `json:"size"`
	Skip    bool     `json:"skip"`
	Targets []Target `json:"targets"`
}

func (m Manifest) validate() error {
	if !validPath(m.Path) || m.Size < 0 || len(m.Chunks) > 10000 {
		return fail(400, "invalid_manifest", "invalid path, size or chunk count")
	}
	var sum int64
	for i, c := range m.Chunks {
		if !validHash(c.Hash) || c.Size <= 0 || c.Size > chunkSize || (i < len(m.Chunks)-1 && c.Size != chunkSize) {
			return fail(400, "invalid_manifest", "invalid hash or chunk size")
		}
		sum += c.Size
	}
	if sum != m.Size {
		return fail(400, "invalid_manifest", "size mismatch")
	}
	return nil
}
func (s *Server) healthy(ctx context.Context, t pgx.Tx, h string, g int64) (int, error) {
	var n int
	e := t.QueryRow(ctx, "SELECT count(*) FROM replicas r JOIN nodes n ON n.id=r.node_id WHERE r.hash=$1 AND r.generation=$2 AND n.last_seen>now()-interval '30 seconds'", h, g).Scan(&n)
	return n, e
}
func (s *Server) prepare(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	var m Manifest
	if e := parse(b, &m); e != nil {
		return e
	}
	if e := m.validate(); e != nil {
		return e
	}
	idem := r.Header.Get("Idempotency-Key")
	if err := checkIdempotencyKey(idem); err != nil {
		return err
	}
	canonical, _ := json.Marshal(m)
	id := randomID()
	plans := []Plan{}
	committed := false
	capacity, e := s.planUpload(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var prior string
		var expiry time.Time
		e := t.QueryRow(ctx, "SELECT id,request_digest,committed,expires_at FROM uploads WHERE idem=$1", idem).Scan(&id, &prior, &committed, &expiry)
		if e != nil && e != pgx.ErrNoRows {
			return e
		}
		if e == nil {
			if prior != digest(canonical) {
				return fail(409, "idempotency_conflict", "key used with another manifest")
			}
			if !committed && expiry.Before(time.Now()) {
				return fail(410, "upload_expired", "use a new idempotency key")
			}
		} else {
			if e = t.QueryRow(ctx, "INSERT INTO uploads(id,idem,request_digest,path,manifest,size,expires_at) VALUES($1,$2,$3,$4,$5,$6,now()+make_interval(secs=>$7)) RETURNING expires_at", id, idem, digest(canonical), m.Path, canonical, m.Size, s.runtime(ctx).UploadSeconds).Scan(&expiry); e != nil {
				return e
			}
		}
		if committed {
			return nil
		}
		if e = checkUploadPath(ctx, t, m.Path); e != nil {
			return e
		}
		seen := map[string]bool{}
		// All placement updates remain under the same PG serialization lock.
		// reservePlacement updates this transaction-local view for shared volumes
		// as well as node quotas; do not rescan all reservations for every chunk.
		var pool []*routingNode
		for _, c := range m.Chunks {
			if seen[c.Hash] {
				continue
			}
			seen[c.Hash] = true
			_, e = t.Exec(ctx, "INSERT INTO blocks(hash,size,state,upload_until) VALUES($1,$2,'uploading',$3) ON CONFLICT(hash) DO NOTHING", c.Hash, c.Size, expiry)
			if e != nil {
				return e
			}
			var state string
			var owner string
			var owned bool
			var size, g int64
			if e = t.QueryRow(ctx, "SELECT state,size,generation,upload_owner,coalesce(upload_until>now(),false) FROM blocks WHERE hash=$1 FOR UPDATE", c.Hash).Scan(&state, &size, &g, &owner, &owned); e != nil {
				return e
			}
			if size != c.Size {
				return fail(409, "hash_size_conflict", "hash already has another size")
			}
			if state == "deleted" || state == "reclaimable" {
				// Revival can invalidate replica/reservation accounting. Rebuild the
				// view after the authoritative state transition, never reuse it across it.
				pool = nil
				var withinGrace bool
				if state == "reclaimable" {
					if e = t.QueryRow(ctx, "SELECT gc_after>now() FROM blocks WHERE hash=$1", c.Hash).Scan(&withinGrace); e != nil {
						return e
					}
				}
				count, err := s.healthy(ctx, t, c.Hash, g)
				if err != nil {
					return err
				}
				if withinGrace && count >= s.cfg.Replicas {
					state = "available"
					if _, e = t.Exec(ctx, "UPDATE blocks SET state='available',gc_after=NULL WHERE hash=$1", c.Hash); e != nil {
						return e
					}
				} else {
					g++
					state = "uploading"
					owned = false
					if _, e = t.Exec(ctx, "DELETE FROM replicas WHERE hash=$1", c.Hash); e != nil {
						return e
					}
					if _, e = t.Exec(ctx, "UPDATE blocks SET generation=$2,state='uploading',gc_after=NULL WHERE hash=$1", c.Hash, g); e != nil {
						return e
					}
				}
			}
			n, e := s.healthy(ctx, t, c.Hash, g)
			if e != nil {
				return e
			}
			p := Plan{Hash: c.Hash, Size: c.Size, Skip: n >= s.cfg.Replicas, Targets: []Target{}}
			if !p.Skip && owned && owner != "" && owner != id {
				plans = append(plans, p)
				continue
			}
			if _, e = t.Exec(ctx, "UPDATE blocks SET upload_owner=$3,upload_until=greatest(upload_until,$2),last_access=now() WHERE hash=$1", c.Hash, expiry, id); e != nil {
				return e
			}
			if !p.Skip {
				if pool == nil {
					pool, e = s.routingPool(ctx, t)
					if e != nil {
						return e
					}
				}
				p.Targets, e = s.uploadTargets(ctx, t, pool, c.Hash, c.Size, g, s.cfg.Replicas-n)
				if e != nil {
					return e
				}
			}
			allowed := false
			if p.Skip {
				allowed, e = canReuseBlock(ctx, t, id, c.Hash, g)
				if e != nil {
					return e
				}
			}
			if p.Skip && !allowed {
				// Full bytes must be verified at a current replica before cross-user dedup.
				candidates, e := s.proofTargets(ctx, t, c.Hash, c.Size, g)
				if e != nil {
					return e
				}
				if len(candidates) == 0 {
					return fail(503, "unavailable", "no fresh replica for ownership proof")
				}
				target := candidates[0]
				target.Token = s.cap(target.Node, "put", c.Hash, c.Size, g)
				p.Skip = false
				p.Targets = []Target{target}
			}
			p.Targets = s.bindUploadTargets(p.Targets, id)
			plans = append(plans, p)
		}
		return nil
	})
	if e != nil {
		return e
	}
	if capacity != nil {
		w.Header().Set("Retry-After", "2")
		jsonOut(w, capacity.Status, map[string]string{"code": capacity.Code, "message": "not enough writable capacity; eligible reclamation scheduled, retry safely"})
		return nil
	}
	jsonOut(w, 200, map[string]any{"upload_id": id, "committed": committed, "blocks": plans})
	return nil
}
func (s *Server) commit(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	id := r.PathValue("id")
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var m Manifest
		var raw []byte
		var done bool
		var expires time.Time
		e := t.QueryRow(ctx, "SELECT manifest,committed,expires_at FROM uploads WHERE id=$1 FOR UPDATE", id).Scan(&raw, &done, &expires)
		if e == pgx.ErrNoRows {
			return fail(404, "not_found", "upload not found")
		}
		if e != nil {
			return e
		}
		if done {
			return nil
		}
		if expires.Before(time.Now()) {
			return fail(410, "upload_expired", "upload expired")
		}
		if e = json.Unmarshal(raw, &m); e != nil {
			return e
		}
		if e = checkUploadPath(ctx, t, m.Path); e != nil {
			return e
		}
		if _, e = t.Exec(ctx, "INSERT INTO paths(path) VALUES($1) ON CONFLICT DO NOTHING", m.Path); e != nil {
			return e
		}
		var trashed *time.Time
		if e = t.QueryRow(ctx, "SELECT trashed_at FROM paths WHERE path=$1 FOR UPDATE", m.Path).Scan(&trashed); e != nil {
			return e
		}
		if trashed != nil {
			return fail(409, "path_trashed", "restore path before adding a version")
		}
		for _, c := range m.Chunks {
			var state string
			var g int64
			if e = t.QueryRow(ctx, "SELECT state,generation FROM blocks WHERE hash=$1", c.Hash).Scan(&state, &g); e != nil {
				return e
			}
			allowed, e := canReuseBlock(ctx, t, id, c.Hash, g)
			if e != nil {
				return e
			}
			if !allowed {
				return fail(409, "upload_proof_required", "full block ownership proof required")
			}
			n, e := s.healthy(ctx, t, c.Hash, g)
			if e != nil {
				return e
			}
			if state != "available" || n < s.cfg.Replicas {
				return fail(409, "blocks_not_ready", "all blocks require full healthy replicas")
			}
		}
		if _, e = t.Exec(ctx, "INSERT INTO versions(id,path,size) VALUES($1,$2,$3)", id, m.Path, m.Size); e != nil {
			return e
		}
		for i, c := range m.Chunks {
			if _, e = t.Exec(ctx, "INSERT INTO version_blocks(version_id,ordinal,hash) VALUES($1,$2,$3)", id, i, c.Hash); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "UPDATE blocks SET refs=refs+1,gc_after=NULL,last_access=now() WHERE hash=$1", c.Hash); e != nil {
				return e
			}
		}
		_, e = t.Exec(ctx, "UPDATE uploads SET committed=true WHERE id=$1", id)
		return e
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"version_id": id, "status": "committed"})
	return nil
}
