package main

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

func (s *Server) download(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	id := r.PathValue("id")
	if strings.HasPrefix(id, "m.") {
		return s.downloadMount(w, r, id)
	}
	var path string
	var size int64
	plans := []Plan{}
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var expiry *time.Time
		e := t.QueryRow(ctx, "SELECT coalesce(v.path,v.trash_path),v.size,coalesce(v.expires_at,p.expires_at) FROM versions v LEFT JOIN paths p ON p.owner_id=v.owner_id AND p.path=v.path WHERE v.id=$1", id).Scan(&path, &size, &expiry)
		if e == pgx.ErrNoRows {
			return fail(404, "not_found", "version not found")
		}
		if e != nil {
			return e
		}
		if expiry != nil && expiry.Before(time.Now()) {
			return fail(410, "trash_expired", "restore window expired")
		}
		rows, e := t.Query(ctx, "SELECT b.hash,b.size,b.generation FROM version_blocks vb JOIN blocks b ON b.hash=vb.hash WHERE vb.version_id=$1 ORDER BY vb.ordinal", id)
		if e != nil {
			return e
		}
		type item struct {
			h    string
			s, g int64
		}
		list := []item{}
		for rows.Next() {
			var a item
			if e = rows.Scan(&a.h, &a.s, &a.g); e != nil {
				rows.Close()
				return e
			}
			list = append(list, a)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, a := range list {
			p := Plan{Hash: a.h, Size: a.s, Targets: []Target{}}
			if r.URL.Query().Get("manifest") == "true" {
				plans = append(plans, p)
				continue
			}
			p.Targets, e = s.readTargets(ctx, t, a.h, a.s, a.g)
			if e != nil {
				return e
			}
			plans = append(plans, p)
		}
		return nil
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"version_id": id, "path": path, "size": size, "blocks": plans})
	return nil
}
