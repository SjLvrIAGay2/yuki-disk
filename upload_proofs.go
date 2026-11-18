package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

func canReuseBlock(ctx context.Context, t pgx.Tx, upload, hash string, generation int64) (bool, error) {
	var allowed bool
	e := t.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM upload_proofs p JOIN uploads prior ON prior.id=p.upload_id JOIN uploads current ON current.owner_id=prior.owner_id WHERE current.id=$1 AND p.hash=$2 AND p.generation=$3)
 OR EXISTS(SELECT 1 FROM version_blocks vb JOIN versions v ON v.id=vb.version_id JOIN uploads u ON u.owner_id=v.owner_id WHERE u.id=$1 AND vb.hash=$2)`, upload, hash, generation).Scan(&allowed)
	return allowed, e
}
func (s *Server) uploadProof(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "node_required", "node identity required")
	}
	var p struct {
		Upload     string `json:"upload_id"`
		Hash       string `json:"hash"`
		Generation int64  `json:"generation"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		var valid bool
		e := t.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM uploads u JOIN blocks b ON b.hash=$2 JOIN replicas rp ON rp.hash=b.hash AND rp.generation=b.generation WHERE u.id=$1 AND u.expires_at>now() AND b.generation=$3 AND rp.node_id=$4 AND u.manifest->'chunks' @> jsonb_build_array(jsonb_build_object('hash',b.hash,'size',b.size)))`, p.Upload, p.Hash, p.Generation, strings.TrimPrefix(identity, "node:")).Scan(&valid)
		if e != nil {
			return e
		}
		if !valid {
			return fail(409, "proof_rejected", "upload proof no longer applicable")
		}
		_, e = t.Exec(r.Context(), "INSERT INTO upload_proofs(upload_id,hash,generation) VALUES($1,$2,$3) ON CONFLICT(upload_id,hash) DO UPDATE SET generation=excluded.generation,created_at=now()", p.Upload, p.Hash, p.Generation)
		return e
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"status": "confirmed"})
	return nil
}
func (s *Server) bindUploadTargets(targets []Target, upload string) []Target {
	for i := range targets {
		var c Capability
		if decodeToken(s.cfg.CapabilityKey, targets[i].Token, &c) == nil {
			c.Upload = upload
			targets[i].Token = encodeToken(s.cfg.CapabilityKey, c)
		}
	}
	return targets
}

// Proof writes reuse existing locations but still require signed fresh resources.
// No new allocation is made; bandwidth is only a scheduling weight.
func (s *Server) proofTargets(ctx context.Context, t pgx.Tx, hash string, size, generation int64) ([]Target, error) {
	pool, e := s.routingPool(ctx, t)
	if e != nil {
		return nil, e
	}
	candidates := []*routingNode{}
	for _, n := range pool {
		if !n.fresh || n.resource.Storage.ReadOnly {
			continue
		}
		var present bool
		if e = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM replicas WHERE hash=$1 AND generation=$2 AND node_id=$3)", hash, generation, n.id).Scan(&present); e != nil {
			return nil, e
		}
		if present {
			candidates = append(candidates, n)
		}
	}
	rankNodes(candidates, size, false, "")
	if len(candidates) == 0 {
		return nil, fail(503, "unavailable", "no fresh replica for ownership proof")
	}
	n := candidates[0]
	if e = reserveBandwidth(ctx, t, n, size, false); e != nil {
		return nil, e
	}
	return []Target{{s.hasP2P(n.id), n.id, n.url + "/v1/blocks/" + hash, s.cap(n.id, "put", hash, size, generation)}}, nil
}
