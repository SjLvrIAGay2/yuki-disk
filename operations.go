package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"path"
	"sort"
	"strings"
)

type Operation struct {
	Operation string   `json:"operation"`
	Paths     []string `json:"paths"`
	Target    string   `json:"target,omitempty"`
}

func (s *Server) operate(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	var p Operation
	if e := parse(b, &p); e != nil {
		return e
	}
	if len(p.Paths) == 0 || len(p.Paths) > 100 {
		return fail(400, "invalid_selection", "select 1..100 entries")
	}
	switch p.Operation {
	case "rename", "move", "copy", "trash", "restore":
	default:
		return fail(400, "invalid_operation", "unsupported operation")
	}
	if p.Operation == "rename" && len(p.Paths) != 1 {
		return fail(400, "invalid_selection", "rename requires one entry")
	}
	for _, p := range p.Paths {
		if !validPath(p) {
			return fail(400, "invalid_path", "invalid source path")
		}
	}
	if p.Target != "" && !validPath(p.Target) {
		return fail(400, "invalid_path", "invalid target path")
	}
	if p.Operation == "rename" && p.Target == "" {
		return fail(400, "invalid_path", "target required")
	}
	key := r.Header.Get("Idempotency-Key")
	if err := checkIdempotencyKey(key); err != nil {
		return err
	}
	canonical, _ := json.Marshal(p)
	var result json.RawMessage
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		return operationResult(ctx, t, key, digest(canonical), &result, func() error {
			var e error
			roots := compactPaths(p.Paths)
			if (p.Operation == "move" || p.Operation == "copy") && p.Target != "" {
				var ok bool
				if e = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1 AND is_dir AND trashed_at IS NULL)", p.Target).Scan(&ok); e != nil {
					return e
				}
				if !ok {
					return fail(409, "target_unavailable", "target folder does not exist")
				}
			}
			for _, src := range roots {
				if p.Operation == "trash" || p.Operation == "restore" {
					if e = s.changeTree(ctx, t, src, p.Operation == "restore"); e != nil {
						return e
					}
					continue
				}
				dest := path.Join(p.Target, path.Base(src))
				if p.Operation == "rename" {
					dest = p.Target
				}
				if !validPath(dest) || dest == src || strings.HasPrefix(dest, src+"/") {
					return fail(409, "invalid_target", "target overlaps source")
				}
				if e = s.relocateTree(ctx, t, src, dest, p.Operation == "copy"); e != nil {
					return e
				}
			}
			result, _ = json.Marshal(map[string]any{"operation": p.Operation, "paths": roots, "target": p.Target})
			return nil
		})
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, result)
	return nil
}
func compactPaths(paths []string) []string {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	out := []string{}
	for _, p := range sorted {
		covered := false
		for _, root := range out {
			if p == root || strings.HasPrefix(p, root+"/") {
				covered = true
				break
			}
		}
		if !covered {
			out = append(out, p)
		}
	}
	return out
}

type treeEntry struct {
	Path    string
	IsDir   bool
	Trashed bool
}

func (s *Server) relocateTree(ctx context.Context, t pgx.Tx, src, dest string, copy bool) error {
	if err := readonlyPath(ctx, t, src, true); err != nil {
		return err
	}
	if err := readonlyPath(ctx, t, dest, true); err != nil {
		return err
	}
	rows, e := t.Query(ctx, "SELECT path,is_dir,trashed_at IS NOT NULL FROM paths WHERE path=$1 OR starts_with(path,$1||'/') ORDER BY path", src)
	if e != nil {
		return e
	}
	items := []treeEntry{}
	for rows.Next() {
		var item treeEntry
		if e = rows.Scan(&item.Path, &item.IsDir, &item.Trashed); e != nil {
			rows.Close()
			return e
		}
		items = append(items, item)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(items) == 0 || items[0].Path != src {
		return fail(404, "not_found", "source not found")
	}
	if items[0].Trashed {
		return fail(409, "source_trashed", "restore source first")
	}
	if !copy {
		if _, e = t.Exec(ctx, "UPDATE uploads SET expires_at=now() WHERE NOT committed AND (path=$1 OR starts_with(path,$1||'/'))", src); e != nil {
			return e
		}
	}
	if e = ensureParents(ctx, t, dest); e != nil {
		return e
	}
	for _, item := range items {
		if item.Trashed {
			if copy {
				continue
			}
			return fail(409, "subtree_trashed", "restore deleted descendants before moving folder")
		}
		target := dest + strings.TrimPrefix(item.Path, src)
		if !validPath(target) {
			return fail(400, "invalid_path", "resulting path is too long")
		}
		var exists bool
		if e = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1)", target).Scan(&exists); e != nil {
			return e
		}
		if exists {
			return fail(409, "path_exists", "destination already exists")
		}
		if copy {
			if _, e = t.Exec(ctx, "INSERT INTO paths(path,is_dir) VALUES($1,$2)", target, item.IsDir); e != nil {
				return e
			}
			if !item.IsDir {
				if e = s.copyLatest(ctx, t, item.Path, target); e != nil {
					return e
				}
			}
		} else {
			if _, e = t.Exec(ctx, "UPDATE paths SET path=$2 WHERE path=$1", item.Path, target); e != nil {
				return e
			}
			if _, e = t.Exec(ctx, "UPDATE uploads SET expires_at=now() WHERE path=$1 AND NOT committed", item.Path); e != nil {
				return e
			}
		}
	}
	return nil
}
