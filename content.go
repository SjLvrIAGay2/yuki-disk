package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
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
type ContentBlock struct {
	Hash    string `json:"hash"`
	Size    int64  `json:"size"`
	Targets []struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	} `json:"targets"`
}
type ContentPlan struct {
	Path     string         `json:"path"`
	Version  string         `json:"version"`
	Size     int64          `json:"size"`
	Modified time.Time      `json:"modified_at"`
	Export   string         `json:"export"`
	Relative string         `json:"relative"`
	Revision string         `json:"revision"`
	Nonce    string         `json:"nonce"`
	Expires  int64          `json:"expires"`
	Range    string         `json:"range,omitempty"`
	Blocks   []ContentBlock `json:"blocks"`
}

func (n *Node) serveFile(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/public/v1/content" {
		output(w, http.StatusGone, map[string]string{"code": "legacy_share_expired"})
		return
	}
	if len(r.Header.Get("Range")) > 1024 || strings.Count(r.Header.Get("Range"), ",") > 7 {
		output(w, 416, map[string]string{"code": "invalid_range"})
		return
	}
	select {
	case n.contentSlots <- struct{}{}:
		defer func() { <-n.contentSlots }()
	default:
		output(w, 503, map[string]string{"code": "busy"})
		return
	}
	req := ContentRequest{Path: r.URL.Query().Get("path")}
	if r.URL.Path == "/v1/content" || r.URL.Path == "/v1/content/range" {
		req.Path = ""
		req.Token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if req.Token == "" {
			output(w, 403, map[string]string{"code": "unauthorized"})
			return
		}
	}
	if r.URL.Path == "/v1/content/range" {
		if !boundedContentRange(r.Header.Get("Range")) || r.Header.Get("If-Range") != "" {
			output(w, 416, map[string]string{"code": "invalid_range"})
			return
		}
		req.Range = r.Header.Get("Range")
	}
	body, err := n.signed(r.Context(), "/internal/v1/content", req)
	if err != nil {
		status := 503
		var ce *controlError
		if errors.As(err, &ce) {
			if ce.Status == 404 || ce.Status == 403 || ce.Status == 409 || ce.Status == 400 {
				status = ce.Status
			}
		}
		output(w, status, map[string]string{"code": "content_unavailable"})
		return
	}
	var plan ContentPlan
	if json.Unmarshal(body, &plan) != nil {
		output(w, 503, map[string]string{"code": "content_unavailable"})
		return
	}
	if req.Range != "" && (plan.Export == "" || plan.Range != req.Range || plan.Revision == "" || plan.Nonce == "") {
		output(w, 403, map[string]string{"code": "content_unavailable"})
		return
	}
	var reader io.ReadSeeker
	revision := plan.Version
	if plan.Export != "" {
		if plan.Nonce != "" && n.consume("file-read", plan.Nonce, plan.Expires) != nil {
			output(w, 403, map[string]string{"code": "replay_rejected"})
			return
		}
		f, e := n.openExport(plan.Export, plan.Relative)
		if e != nil {
			output(w, 404, map[string]string{"code": "not_found"})
			return
		}
		defer f.Close()
		st, e := f.Stat()
		if e != nil || !st.Mode().IsRegular() {
			output(w, 404, map[string]string{"code": "not_found"})
			return
		}
		revision = fileRevision(f, st)
		if revision == "" {
			output(w, 503, map[string]string{"code": "content_unavailable"})
			return
		}
		if plan.Revision != "" && plan.Revision != revision {
			output(w, 409, map[string]string{"code": "file_changed"})
			return
		}
		if req.Range != "" && plan.Size != st.Size() {
			output(w, 409, map[string]string{"code": "file_changed"})
			return
		}
		plan.Size = st.Size()
		plan.Modified = st.ModTime()
		reader = &stableExportReader{f, revision}
	} else {
		reader, err = newContentReader(n, r, plan)
		if err != nil {
			output(w, 503, map[string]string{"code": "content_unavailable"})
			return
		}
	}
	if plan.Size < 0 || plan.Size > 9007199254740991 {
		output(w, 413, map[string]string{"code": "file_too_large"})
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("%q", revision))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	kind := mime.TypeByExtension(path.Ext(plan.Path))
	if kind == "" {
		kind = "application/octet-stream"
	}
	w.Header().Set("Content-Type", kind)
	n.network.readers.Add(1)
	defer n.network.readers.Add(-1)
	started := time.Now()
	count := &contentWriter{ResponseWriter: w}
	defer func() { n.network.sent.Add(count.bytes); n.network.observe(true, count.bytes, time.Since(started)) }()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(30 * time.Second))
	http.ServeContent(count, r, path.Base(plan.Path), plan.Modified, reader)
}

type contentWriter struct {
	http.ResponseWriter
	bytes int64
}

func (w *contentWriter) Write(b []byte) (int, error) {
	_ = http.NewResponseController(w.ResponseWriter).SetWriteDeadline(time.Now().Add(30 * time.Second))
	n, e := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, e
}

type stableExportReader struct {
	*os.File
	revision string
}

func (f *stableExportReader) Read(b []byte) (int, error) {
	st, e := f.Stat()
	if e != nil || (f.revision == "" || fileRevision(f.File, st) != f.revision) {
		return 0, errors.New("file changed")
	}
	n, e := f.File.Read(b)
	after, se := f.Stat()
	if se != nil || fileRevision(f.File, after) != f.revision {
		return 0, errors.New("file changed")
	}
	return n, e
}
