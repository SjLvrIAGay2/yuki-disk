package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"strings"
	"time"
)

type queryRow interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func visible(ctx context.Context, q queryRow, p string) (bool, error) {
	var mode string
	parents := []string{}
	for {
		parents = append(parents, p)
		at := strings.LastIndex(p, "/")
		if at < 0 {
			break
		}
		p = p[:at]
	}
	err := q.QueryRow(ctx, "SELECT mode FROM visibility WHERE path=ANY($1) ORDER BY length(path) DESC LIMIT 1", parents).Scan(&mode)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return mode == "public", err
}
func (s *Server) visibility(w http.ResponseWriter, r *http.Request, b []byte, _ string) error {
	p, mode := r.URL.Query().Get("path"), ""
	if r.Method == "POST" {
		var req struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		if err := parse(b, &req); err != nil {
			return err
		}
		p, mode = req.Path, req.Mode
		if mode != "public" && mode != "private" && mode != "inherit" {
			return fail(400, "invalid_visibility", "invalid visibility")
		}
	}
	if !validPath(p) {
		return fail(400, "invalid_path", "invalid path")
	}
	isDir := false
	m, mounted := s.mountFor(r.Context(), p)
	if mounted {
		var st ExportEntry
		if err := s.exportRequest(r.Context(), m, p, false, "", "", 1, &st); err != nil {
			return err
		}
		isDir = st.IsDir
	}
	effective := false
	own := "inherit"
	err := s.tx(r.Context(), func(t pgx.Tx) error {
		ctx := r.Context()
		if !mounted {
			e := t.QueryRow(ctx, "SELECT is_dir FROM paths WHERE path=$1 AND trashed_at IS NULL", p).Scan(&isDir)
			if e == pgx.ErrNoRows {
				return fail(404, "not_found", "entry not found")
			}
			if e != nil {
				return e
			}
		}
		if mode != "" {
			if mode == "inherit" {
				if _, e := t.Exec(ctx, "DELETE FROM visibility WHERE path=$1", p); e != nil {
					return e
				}
			} else {
				if _, e := t.Exec(ctx, "INSERT INTO visibility(path,mode) VALUES($1,$2) ON CONFLICT(owner_id,path) DO UPDATE SET mode=excluded.mode,updated_at=now()", p, mode); e != nil {
					return e
				}
			}
		}
		e := t.QueryRow(ctx, "SELECT mode FROM visibility WHERE path=$1", p).Scan(&own)
		if e != nil && e != pgx.ErrNoRows {
			return e
		}
		effective, e = visible(ctx, t, p)
		return e
	})
	if err != nil {
		return err
	}
	jsonOut(w, 200, map[string]any{"path": p, "mode": own, "public": effective, "is_dir": isDir, "content_url": publicContentPath(p)})
	return nil
}

// Capture only metadata produced by the authoritative browse implementation.
type metadataResponse struct {
	bytes.Buffer
	header http.Header
}

func (w *metadataResponse) Header() http.Header { return w.header }
func (w *metadataResponse) WriteHeader(int)     {}
func (s *Server) publicBrowse(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	p := r.URL.Query().Get("path")
	if p != "" && !validPath(p) {
		return fail(400, "invalid_path", "invalid path")
	}
	if r.URL.Query().Get("trash") != "" || r.URL.Query().Get("q") != "" {
		return fail(400, "invalid_query", "public lists only direct children")
	}
	if p != "" {
		ok, err := visible(r.Context(), s.query(r.Context()), p)
		if err != nil {
			return err
		}
		if !ok {
			return fail(404, "not_found", "folder not found")
		}
	}
	metadata := &metadataResponse{header: http.Header{}}
	if err := s.browse(metadata, r, nil, ""); err != nil {
		return err
	}
	var page struct {
		Items  []map[string]any `json:"items"`
		Cursor string           `json:"next_cursor"`
	}
	if err := json.Unmarshal(metadata.Bytes(), &page); err != nil {
		return err
	}
	items := []map[string]any{}
	err := s.tx(r.Context(), func(t pgx.Tx) error {
		if p != "" {
			ok, e := visible(r.Context(), t, p)
			if e != nil {
				return e
			}
			if !ok {
				return fail(404, "not_found", "folder not found")
			}
		}
		for _, item := range page.Items {
			path := item["path"].(string)
			ok, e := visible(r.Context(), t, path)
			if e != nil {
				return e
			}
			if ok {
				entry := map[string]any{"path": path, "name": item["name"], "is_dir": item["is_dir"], "size": item["size"], "modified_at": item["modified_at"]}
				if item["is_dir"] == false {
					entry["content_url"] = publicContentPath(path)
				}
				items = append(items, entry)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Cursor is opaque to avoid exposing a filtered private name.
	next := ""
	if page.Cursor != "" {
		next = sealCursor(s.cfg.SessionKey, publicCursor{p, page.Cursor, time.Now().Add(time.Hour).Unix()})
	}
	jsonOut(w, 200, map[string]any{"items": items, "next_cursor": next})
	return nil
}

type publicCursor struct {
	Path    string `json:"path"`
	After   string `json:"after"`
	Expires int64  `json:"exp"`
}

func (s *Server) publicList(w http.ResponseWriter, r *http.Request, b []byte, id string) error {
	q := r.URL.Query()
	if raw := q.Get("cursor"); raw != "" {
		var cursor publicCursor
		if openCursor(s.cfg.SessionKey, raw, &cursor) != nil || cursor.Path != q.Get("path") || cursor.Expires <= time.Now().Unix() {
			return fail(400, "invalid_cursor", "invalid cursor")
		}
		q.Set("cursor", cursor.After)
		r.URL.RawQuery = q.Encode()
	}
	return s.publicBrowse(w, r, b, id)
}

func cursorAEAD(key string) cipher.AEAD {
	k := sha256.Sum256([]byte("public-cursor:" + key))
	block, _ := aes.NewCipher(k[:])
	g, _ := cipher.NewGCM(block)
	return g
}
func sealCursor(key string, p publicCursor) string {
	g := cursorAEAD(key)
	nonce := make([]byte, g.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	b, _ := json.Marshal(p)
	return base64.RawURLEncoding.EncodeToString(g.Seal(nonce, nonce, b, []byte("public-list-v1")))
}
func openCursor(key, raw string, p *publicCursor) error {
	g := cursorAEAD(key)
	b, e := base64.RawURLEncoding.DecodeString(raw)
	if e != nil || len(b) < g.NonceSize() {
		return errors.New("invalid cursor")
	}
	b, e = g.Open(nil, b[:g.NonceSize()], b[g.NonceSize():], []byte("public-list-v1"))
	if e != nil {
		return e
	}
	return json.Unmarshal(b, p)
}
