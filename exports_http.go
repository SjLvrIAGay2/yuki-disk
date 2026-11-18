package main

import (
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"sort"
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

func (n *Node) controlRead(r *http.Request, b []byte) bool {
	ts, nonce := r.Header.Get("X-Yuki-Time"), r.Header.Get("X-Yuki-Nonce")
	v, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || v < time.Now().Unix()-30 || v > time.Now().Unix()+30 || r.Header.Get("X-Yuki-Identity") != "control" {
		return false
	}
	msg := internalMessage(r, "control", b, ts, nonce)
	sig, err := hex.DecodeString(r.Header.Get("X-Yuki-Signature"))
	return err == nil && hmac.Equal(sig, mac(n.cfg.Key, msg)) && n.consume("control", nonce, v+61) == nil
}
func (n *Node) exportMetadata(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8192))
	if err != nil || !n.controlRead(r, b) {
		output(w, 403, map[string]string{"code": "unauthorized"})
		return
	}
	var req struct {
		Export string `json:"export"`
		Path   string `json:"path"`
		List   bool   `json:"list"`
		Cursor string `json:"cursor"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	if json.Unmarshal(b, &req) != nil || req.Limit < 1 || req.Limit > 200 || len(req.Query) > 256 {
		output(w, 400, map[string]string{"code": "invalid_request"})
		return
	}
	if req.Export == "" && req.Path == "" && req.List {
		items := []map[string]string{}
		for _, export := range n.cfg.Exports {
			items = append(items, map[string]string{"id": export.ID})
		}
		output(w, 200, map[string]any{"exports": items})
		return
	}
	f, err := n.openExport(req.Export, req.Path)
	if err != nil {
		output(w, 404, map[string]string{"code": "not_found"})
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		output(w, 503, map[string]string{"code": "unavailable"})
		return
	}
	revision := fileRevision(f, st)
	if revision == "" {
		output(w, 503, map[string]string{"code": "unavailable"})
		return
	}
	item := ExportEntry{path.Base(req.Path), st.IsDir(), st.Size(), st.ModTime(), revision}
	if st.Size() < 0 || st.Size() > 9007199254740991 {
		output(w, 413, map[string]string{"code": "file_too_large"})
		return
	}
	if !req.List {
		output(w, 200, item)
		return
	}
	if !st.IsDir() {
		output(w, 404, map[string]string{"code": "not_folder"})
		return
	}
	items := []ExportEntry{}
	seen := 0
	for {
		entries, e := f.ReadDir(128)
		for _, entry := range entries {
			seen++
			if seen > 100000 {
				output(w, 413, map[string]string{"code": "directory_too_large"})
				return
			}
			if entry.Name() <= req.Cursor || !safeRelative(entry.Name()) || !strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(req.Query)) {
				continue
			}
			child, ce := openComponents(f, entry.Name(), false)
			if ce != nil {
				continue
			}
			info, ce := child.Stat()
			revision := ""
			if ce == nil {
				revision = fileRevision(child, info)
			}
			child.Close()
			if ce != nil || revision == "" || info.Size() < 0 || info.Size() > 9007199254740991 {
				continue
			}
			items = append(items, ExportEntry{entry.Name(), info.IsDir(), info.Size(), info.ModTime(), revision})
			sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
			if len(items) > req.Limit+1 {
				items = items[:req.Limit+1]
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			output(w, 503, map[string]string{"code": "unavailable"})
			return
		}
		if r.Context().Err() != nil {
			return
		}
	}
	cursor := ""
	if len(items) > req.Limit {
		items = items[:req.Limit]
		cursor = items[len(items)-1].Name
	}
	output(w, 200, map[string]any{"items": items, "next_cursor": cursor})
}
