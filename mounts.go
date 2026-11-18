package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type ExportEntry struct {
	Name     string    `json:"name"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified_at"`
	Revision string    `json:"revision"`
}
type FileIdentity struct {
	Purpose  string `json:"purpose"`
	Path     string `json:"path"`
	Mount    string `json:"mount"`
	Revision string `json:"revision"`
	Node     string `json:"node"`
	Expires  int64  `json:"exp"`
	ID       string `json:"jti"`
	Owner    string `json:"owner"`
	Start    int64  `json:"start,omitempty"`
	Length   int64  `json:"length,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

func (s *Server) mountReadonly(ctx context.Context, p string) bool {
	_, ok := s.mountFor(ctx, p)
	return ok
}
func (s *Server) exportRequest(ctx context.Context, m MountConfig, p string, list bool, cursor, q string, limit int, out any) error {
	if err := currentMount(ctx, s.query(ctx), m); err != nil {
		return err
	}
	var source string
	if e := s.query(ctx).QueryRow(ctx, "SELECT source_path FROM mounts WHERE id=$1", m.ID).Scan(&source); e != nil {
		return e
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(p, m.Path), "/")
	if source != "" {
		relative = path.Join(source, relative)
	}
	return s.nodeExport(ctx, m.Node, m.Export, relative, list, cursor, q, limit, out)
}
func (s *Server) nodeExport(ctx context.Context, node, export, relative string, list bool, cursor, q string, limit int, out any) error {
	n, ok := s.nodeConfig(node)
	if !ok {
		return fail(503, "mount_unavailable", "node unavailable")
	}
	body, _ := json.Marshal(map[string]any{"export": export, "path": relative, "list": list, "cursor": cursor, "query": q, "limit": limit})
	req, _ := http.NewRequestWithContext(ctx, "POST", n.URL+"/internal/v1/export", bytes.NewReader(body))
	ts, nonce := strconv.FormatInt(time.Now().Unix(), 10), randomID()
	msg := strings.Join([]string{"control", req.Method, req.URL.EscapedPath(), req.URL.RawQuery, digest(body), ts, nonce}, "\n")
	req.Header.Set("X-Yuki-Identity", "control")
	req.Header.Set("X-Yuki-Time", ts)
	req.Header.Set("X-Yuki-Nonce", nonce)
	req.Header.Set("X-Yuki-Signature", hex.EncodeToString(mac(n.Key, msg)))
	client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect forbidden") }}
	resp, err := client.Do(req)
	if err != nil {
		return fail(503, "mount_unavailable", "server folder unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return fail(404, "not_found", "entry not found")
	}
	if resp.StatusCode != 200 {
		return fail(503, "mount_unavailable", "server folder unavailable")
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}
func (s *Server) mountID(m MountConfig, p string, e ExportEntry, owner string) string {
	return "m." + encodeToken(s.cfg.CapabilityKey, FileIdentity{Purpose: "mount-version", Path: p, Mount: m.ID, Node: m.Node, Revision: e.Revision, Owner: owner})
}
func (s *Server) browseMount(w http.ResponseWriter, r *http.Request, m MountConfig) error {
	limit, _, err := pagination(r)
	if err != nil {
		return err
	}
	dir := r.URL.Query().Get("path")
	cursor := r.URL.Query().Get("cursor")
	if cursor != "" {
		if parentPath(cursor) != dir {
			return fail(400, "invalid_cursor", "invalid cursor")
		}
		cursor = path.Base(cursor)
	}
	var page struct {
		Items  []ExportEntry `json:"items"`
		Cursor string        `json:"next_cursor"`
	}
	if err = s.exportRequest(r.Context(), m, dir, true, cursor, r.URL.Query().Get("q"), limit, &page); err != nil {
		return err
	}
	items := []map[string]any{}
	for _, e := range page.Items {
		p := path.Join(dir, e.Name)
		if !validPath(p) {
			continue
		}
		id := ""
		size := e.Size
		if !e.IsDir {
			id = s.mountID(m, p, e, r.Context().Value(principalKey{}).(principal).ID)
		} else {
			size = 0
		}
		items = append(items, map[string]any{"path": p, "name": e.Name, "is_dir": e.IsDir, "size": size, "version_id": id, "versions": 0, "modified_at": e.Modified, "created_at": e.Modified, "readonly": true})
	}
	next := ""
	if page.Cursor != "" {
		next = path.Join(dir, page.Cursor)
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_cursor": next, "readonly": true})
	return nil
}
func (s *Server) downloadMount(w http.ResponseWriter, r *http.Request, id string) error {
	var f FileIdentity
	if decodeToken(s.cfg.CapabilityKey, strings.TrimPrefix(id, "m."), &f) != nil || f.Purpose != "mount-version" || f.Owner != r.Context().Value(principalKey{}).(principal).ID {
		return fail(404, "not_found", "invalid file")
	}
	m, ok := s.mountFor(r.Context(), f.Path)
	if !ok || m.ID != f.Mount || m.Node != f.Node {
		return fail(404, "not_found", "mount changed")
	}
	var st ExportEntry
	if err := s.exportRequest(r.Context(), m, f.Path, false, "", "", 1, &st); err != nil {
		return err
	}
	if st.IsDir || st.Revision != f.Revision {
		return fail(409, "file_changed", "file changed; refresh folder")
	}
	f.Purpose = "file-read"
	f.Expires = time.Now().Add(time.Minute).Unix()
	f.ID = randomID()
	n, _ := s.nodeConfig(m.Node)
	token := encodeToken(s.cfg.CapabilityKey, f)
	jsonOut(w, 200, map[string]any{"version_id": id, "path": f.Path, "size": st.Size, "blocks": []Plan{}, "source": map[string]any{"url": n.URL + "/v1/content", "token": token, "revision": st.Revision, "node": n.ID, "p2p": s.hasP2P(n.ID), "ranges": true}})
	return nil
}
func (s *Server) mountList(w http.ResponseWriter, r *http.Request, _ []byte, owner string) error {
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT path FROM mounts WHERE owner_id=$1 ORDER BY path", owner)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var p string
		if e = rows.Scan(&p); e != nil {
			return e
		}
		items = append(items, map[string]any{"path": p, "readonly": true})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items})
	return nil
}

func publicContentPath(p string) string {
	return "/public/v1/content?" + url.Values{"path": {p}}.Encode()
}
