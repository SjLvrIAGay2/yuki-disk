package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"path"
	"strings"
	"time"
)

type ContentRequest struct {
	Path    string `json:"path"`
	Token   string `json:"token"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
	Range   string `json:"range,omitempty"`
}
type ContentPlan struct {
	Path     string    `json:"path"`
	Version  string    `json:"version"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified_at"`
	Export   string    `json:"export,omitempty"`
	Relative string    `json:"relative,omitempty"`
	Revision string    `json:"revision,omitempty"`
	Nonce    string    `json:"nonce,omitempty"`
	Expires  int64     `json:"expires,omitempty"`
	Range    string    `json:"range,omitempty"`
	Blocks   []Plan    `json:"blocks"`
}

func (s *Server) publicFile(ctx context.Context, t pgx.Tx, p string) (string, int64, time.Time, error) {
	var id string
	var size int64
	var modified time.Time
	ok, e := visible(ctx, t, p)
	if e != nil {
		return id, size, modified, e
	}
	if !ok {
		return id, size, modified, fail(404, "not_found", "file not found")
	}
	e = t.QueryRow(ctx, `SELECT v.id,v.size,v.created_at FROM versions v JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path WHERE p.path=$1 AND p.trashed_at IS NULL AND NOT p.is_dir ORDER BY v.created_at DESC,v.id DESC LIMIT 1`, p).Scan(&id, &size, &modified)
	if e == pgx.ErrNoRows {
		e = fail(404, "not_found", "file not found")
	}
	return id, size, modified, e
}
func (s *Server) publicContent(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	p := r.URL.Query().Get("path")
	if !validPath(p) {
		return fail(400, "invalid_path", "invalid path")
	}
	node := ""
	err := s.tx(r.Context(), func(t pgx.Tx) error {
		if m, ok := s.mountFor(r.Context(), p); ok {
			if e := currentMount(r.Context(), t, m); e != nil {
				return e
			}
			allowed, e := visible(r.Context(), t, p)
			if e != nil {
				return e
			}
			if !allowed {
				return fail(404, "not_found", "file not found")
			}
			node = m.Node
			return nil
		}
		_, _, _, e := s.publicFile(r.Context(), t, p)
		if e != nil {
			return e
		}
		pool, e := s.routingPool(r.Context(), t)
		if e != nil {
			return e
		}
		rankNodes(pool, chunkSize, true, "")
		if len(pool) == 0 {
			return fail(503, "unavailable", "no live gateway")
		}
		node = pool[0].id
		return nil
	})
	if err != nil {
		return err
	}
	n, ok := s.nodeConfig(node)
	if !ok {
		return fail(503, "unavailable", "node unavailable")
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, n.URL+publicContentPath(p), http.StatusTemporaryRedirect)
	return nil
}
func (s *Server) contentPlan(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	node := strings.TrimPrefix(identity, "node:")
	if identity == node {
		return fail(403, "node_required", "node identity required")
	}
	var req ContentRequest
	if err := parse(b, &req); err != nil {
		return err
	}
	if req.Token == "" {
		return legacyPublicGone(w, r, b, identity)
	}
	var out any
	err := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		if req.Token != "" {
			var f FileIdentity
			if decodeToken(s.cfg.CapabilityKey, req.Token, &f) != nil || (f.Purpose != "file-read" && f.Purpose != "file-range") || f.Node != node || f.Expires <= time.Now().Unix() || f.Expires > time.Now().Unix()+90 || len(f.ID) != 48 {
				return fail(403, "capability_rejected", "invalid file capability")
			}
			if (f.Purpose == "file-read" && req.Range != "") || (f.Purpose == "file-range" && !validFileRange(f, req.Range)) {
				return fail(403, "capability_rejected", "invalid file scope")
			}
			if f.Owner == "" {
				return fail(403, "capability_rejected", "unbound mount capability")
			}
			var allowed bool
			if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND role='user' AND NOT disabled AND deleted_at IS NULL)", f.Owner).Scan(&allowed); e != nil {
				return e
			}
			if !allowed {
				return fail(403, "owner_unavailable", "owner unavailable")
			}
			if _, e := t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", f.Owner); e != nil {
				return e
			}
			ctx = context.WithValue(ctx, requestTransactionKey{}, t)
			m, ok := s.mountFor(ctx, f.Path)
			if !ok || m.ID != f.Mount || m.Node != node {
				return fail(404, "not_found", "mount changed")
			}
			if e := currentMount(ctx, t, m); e != nil {
				return e
			}
			var source string
			if e := t.QueryRow(ctx, "SELECT source_path FROM mounts WHERE id=$1", m.ID).Scan(&source); e != nil {
				return e
			}
			relative := strings.TrimPrefix(strings.TrimPrefix(f.Path, m.Path), "/")
			if source != "" {
				relative = path.Join(source, relative)
			}
			out = ContentPlan{Path: f.Path, Export: m.Export, Relative: relative, Revision: f.Revision, Nonce: f.ID, Expires: f.Expires, Range: req.Range, Size: f.Size}
			return nil
		}
		if !validPath(req.Path) {
			return fail(400, "invalid_path", "invalid path")
		}
		if m, ok := s.mountFor(r.Context(), req.Path); ok {
			if e := currentMount(ctx, t, m); e != nil {
				return e
			}
			allowed, e := visible(ctx, t, req.Path)
			if e != nil {
				return e
			}
			if !allowed || m.Node != node {
				return fail(404, "not_found", "file not found")
			}
			out = ContentPlan{Path: req.Path, Export: m.Export, Relative: strings.TrimPrefix(strings.TrimPrefix(req.Path, m.Path), "/")}
			return nil
		}
		id, size, modified, e := s.publicFile(ctx, t, req.Path)
		if e != nil {
			return e
		}
		if req.Version != "" && req.Version != id {
			return fail(409, "file_changed", "file changed")
		}
		if req.Hash != "" {
			if !validHash(req.Hash) {
				return fail(400, "invalid_hash", "invalid hash")
			}
			var g int64
			p := Plan{Hash: req.Hash}
			if e = t.QueryRow(ctx, "SELECT b.size,b.generation FROM version_blocks v JOIN blocks b ON b.hash=v.hash WHERE v.version_id=$1 AND b.hash=$2 LIMIT 1", id, req.Hash).Scan(&p.Size, &g); e == pgx.ErrNoRows {
				return fail(404, "not_found", "block not found")
			}
			if e != nil {
				return e
			}
			p.Targets, e = s.readTargets(ctx, t, p.Hash, p.Size, g)
			out = p
			return e
		}
		plan := ContentPlan{Path: req.Path, Version: id, Size: size, Modified: modified, Blocks: []Plan{}}
		rows, e := t.Query(ctx, "SELECT b.hash,b.size FROM version_blocks v JOIN blocks b ON b.hash=v.hash WHERE v.version_id=$1 ORDER BY v.ordinal", id)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			p := Plan{}
			if e = rows.Scan(&p.Hash, &p.Size); e != nil {
				return e
			}
			plan.Blocks = append(plan.Blocks, p)
		}
		out = plan
		return rows.Err()
	})
	if err != nil {
		return err
	}
	jsonOut(w, 200, out)
	return nil
}
