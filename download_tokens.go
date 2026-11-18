package main

import (
	"github.com/jackc/pgx/v5"
	"net/http"
)

// Issue fresh single-use capabilities just before a block is transferred.
func (s *Server) downloadBlock(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	id, h := r.PathValue("id"), r.PathValue("hash")
	if !validHash(h) {
		return fail(400, "invalid_hash", "invalid block hash")
	}
	p := Plan{Hash: h, Targets: []Target{}}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var g int64
		var expired bool
		e := t.QueryRow(ctx, `SELECT b.size,b.generation,coalesce(coalesce(v.expires_at,p.expires_at)<=now(),false)
  FROM versions v LEFT JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path JOIN version_blocks vb ON vb.version_id=v.id JOIN blocks b ON b.hash=vb.hash
  WHERE v.id=$1 AND b.hash=$2 LIMIT 1`, id, h).Scan(&p.Size, &g, &expired)
		if e == pgx.ErrNoRows {
			return fail(404, "not_found", "block does not belong to version")
		}
		if e != nil {
			return e
		}
		if expired {
			return fail(410, "trash_expired", "restore window expired")
		}
		p.Targets, e = s.readTargets(ctx, t, h, p.Size, g)
		if e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, p)
	return nil
}
