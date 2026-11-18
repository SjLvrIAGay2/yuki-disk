package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

func (s *Server) addMountGrant(ctx context.Context, t pgx.Tx, source, owner, target, id string) error {
	if !validPath(target) {
		return fail(400, "invalid_path", "invalid mount path")
	}
	var active bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND role='user' AND NOT disabled AND deleted_at IS NULL)", owner).Scan(&active); e != nil {
		return e
	}
	if !active {
		return fail(409, "owner_unavailable", "ordinary account unavailable")
	}
	if _, e := t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", owner); e != nil {
		return e
	}
	defer t.Exec(ctx, "SELECT set_config('yuki.owner','',true)")
	var occupied bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1 OR starts_with(path,$1||'/')) OR EXISTS(SELECT 1 FROM uploads WHERE NOT committed AND expires_at>now() AND (path=$1 OR starts_with(path,$1||'/')))", target).Scan(&occupied); e != nil {
		return e
	}
	if occupied {
		return fail(409, "mount_conflict", "target namespace is occupied")
	}
	if e := ensureParents(ctx, t, target); e != nil {
		return e
	}
	if _, e := t.Exec(ctx, "INSERT INTO paths(path,is_dir) VALUES($1,true)", target); e != nil {
		return e
	}
	_, e := t.Exec(ctx, "INSERT INTO mounts(id,owner_id,path,node_id,export_id,source_id,source_path) SELECT $1,$2,$3,node_id,export_id,id,source_path FROM mount_sources WHERE id=$4", id, owner, target, source)
	return e
}

func (s *Server) adminMounts(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	ctx := r.Context()
	if r.Method == "POST" {
		var p struct {
			Name       string `json:"name"`
			Node       string `json:"node"`
			Export     string `json:"export"`
			SourcePath string `json:"source_path"`
		}
		if e := parse(b, &p); e != nil {
			return e
		}
		if len(p.Name) < 1 || len(p.Name) > 120 || !validPath(p.Export) || strings.Contains(p.Export, "/") || (p.SourcePath != "" && !validPath(p.SourcePath)) {
			return fail(400, "invalid_mount", "invalid source")
		}
		key := r.Header.Get("Idempotency-Key")
		if e := checkIdempotencyKey(key); e != nil {
			return e
		}
		canonical, _ := json.Marshal(p)
		var result json.RawMessage
		e := s.tx(ctx, func(t pgx.Tx) error {
			if _, e := t.Exec(ctx, "SELECT set_config('yuki.owner',$1,true)", actor); e != nil {
				return e
			}
			defer t.Exec(ctx, "SELECT set_config('yuki.owner','',true)")
			return operationResult(ctx, t, key, digest(append([]byte("mount-create:"), canonical...)), &result, func() error {
				var entry ExportEntry
				if e := s.nodeExport(ctx, p.Node, p.Export, p.SourcePath, false, "", "", 1, &entry); e != nil {
					return e
				}
				if !entry.IsDir {
					return fail(400, "not_folder", "mount source must be directory")
				}
				id := randomID()
				if _, e := t.Exec(ctx, "INSERT INTO mount_sources(id,name,node_id,export_id,source_path) VALUES($1,$2,$3,$4,$5)", id, p.Name, p.Node, p.Export, p.SourcePath); e != nil {
					return e
				}
				result, _ = json.Marshal(map[string]string{"id": id})
				return audit(ctx, t, actor, "mount.create", id)
			})
		})
		if e != nil {
			return e
		}
		jsonOut(w, 201, result)
		return nil
	}
	rows, e := s.query(ctx).Query(ctx, "SELECT id,name,node_id,export_id,source_path,version FROM mount_sources ORDER BY name,id")
	if e != nil {
		return e
	}
	items := []map[string]any{}
	for rows.Next() {
		var id, name, node, export, source string
		var version int64
		if e = rows.Scan(&id, &name, &node, &export, &source, &version); e != nil {
			rows.Close()
			return e
		}
		items = append(items, map[string]any{"id": id, "name": name, "node": node, "export": export, "source_path": source, "version": version})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	grants, e := s.query(ctx).Query(ctx, "SELECT m.id,m.source_id,m.owner_id,m.path,a.username FROM mounts m JOIN accounts a ON a.id=m.owner_id ORDER BY a.username,m.path")
	if e != nil {
		return e
	}
	defer grants.Close()
	bindings := []map[string]string{}
	for grants.Next() {
		var id, source, owner, path, name string
		if e = grants.Scan(&id, &source, &owner, &path, &name); e != nil {
			return e
		}
		bindings = append(bindings, map[string]string{"id": id, "source_id": source, "owner_id": owner, "path": path, "username": name})
	}
	if e = grants.Err(); e != nil {
		return e
	}
	nodes := []map[string]string{}
	for _, n := range s.cfg.Nodes {
		nodes = append(nodes, map[string]string{"id": n.ID})
	}
	jsonOut(w, 200, map[string]any{"items": items, "grants": bindings, "nodes": nodes})
	return nil
}

func (s *Server) adminMountChange(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	var p struct {
		Version   int64  `json:"version"`
		Operation string `json:"operation"`
		Owner     string `json:"owner_id"`
		Path      string `json:"path"`
		Grant     string `json:"grant_id"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if p.Operation != "grant" && p.Operation != "revoke" && p.Operation != "remove" {
		return fail(400, "invalid_operation", "invalid mount operation")
	}
	id := r.PathValue("id")
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		var version int64
		if e := t.QueryRow(ctx, "SELECT version FROM mount_sources WHERE id=$1 FOR UPDATE", id).Scan(&version); e == pgx.ErrNoRows {
			return fail(404, "not_found", "mount source missing")
		} else if e != nil {
			return e
		}
		if version != p.Version {
			return fail(409, "version_conflict", "mount changed")
		}
		if p.Operation == "grant" {
			if e := s.addMountGrant(ctx, t, id, p.Owner, p.Path, randomID()); e != nil {
				return e
			}
		} else {
			rows, e := t.Query(ctx, "DELETE FROM mounts WHERE source_id=$1 AND ($2='remove' OR id=$3) RETURNING owner_id,path", id, p.Operation, p.Grant)
			if e != nil {
				return e
			}
			type item struct{ owner, path string }
			removed := []item{}
			for rows.Next() {
				var v item
				if e = rows.Scan(&v.owner, &v.path); e != nil {
					rows.Close()
					return e
				}
				removed = append(removed, v)
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				return e
			}
			if p.Operation == "revoke" && len(removed) == 0 {
				return fail(404, "not_found", "grant missing")
			}
			for _, v := range removed {
				if _, e = t.Exec(ctx, "DELETE FROM paths WHERE owner_id=$1 AND path=$2", v.owner, v.path); e != nil {
					return e
				}
			}
		}
		sql := "UPDATE mount_sources SET version=version+1 WHERE id=$1"
		if p.Operation == "remove" {
			sql = "DELETE FROM mount_sources WHERE id=$1"
		}
		if _, e := t.Exec(ctx, sql, id); e != nil {
			return e
		}
		return audit(ctx, t, actor, "mount."+p.Operation, id)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"status": "updated"})
	return nil
}

func (s *Server) adminExportBrowse(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	q := r.URL.Query()
	relative := q.Get("path")
	if relative != "" && !validPath(relative) {
		return fail(400, "invalid_path", "invalid source path")
	}
	var result any
	if e := s.nodeExport(r.Context(), q.Get("node"), q.Get("export"), relative, true, q.Get("cursor"), "", 100, &result); e != nil {
		return e
	}
	jsonOut(w, 200, result)
	return nil
}
