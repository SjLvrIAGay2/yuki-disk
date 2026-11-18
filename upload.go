package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func (c *Client) upload(ctx context.Context, local, remote, idem string) (string, error) {
	f, e := os.Open(local)
	if e != nil {
		return "", e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return "", errors.New("upload source must be a regular file")
	}
	chunks := []Chunk{}
	offsets := map[string]int64{}
	var size int64
	for {
		b := make([]byte, chunkSize)
		n, e := io.ReadFull(f, b)
		if e == io.EOF {
			break
		}
		if e != nil && e != io.ErrUnexpectedEOF {
			return "", e
		}
		b = b[:n]
		h := digest(b)
		offsets[h] = size
		size += int64(n)
		chunks = append(chunks, Chunk{h, int64(n)})
		if len(chunks) > 10000 {
			return "", errors.New("file exceeds 10000 chunks")
		}
		if e == io.ErrUnexpectedEOF {
			break
		}
	}
	idem = idempotencyKey(idem)
	manifest := map[string]any{"path": remote, "size": size, "chunks": chunks}
	var last error
	for attempt := 0; attempt < 8; attempt++ {
		var plan struct {
			ID        string `json:"upload_id"`
			Committed bool   `json:"committed"`
			Blocks    []Plan `json:"blocks"`
		}
		e = c.request(ctx, "POST", "/v1/uploads", manifest, idem, &plan)
		if e == nil && plan.Committed {
			return plan.ID, nil
		}
		if e == nil {
			for _, p := range plan.Blocks {
				if p.Skip {
					continue
				}
				b := make([]byte, p.Size)
				_, e = f.ReadAt(b, offsets[p.Hash])
				if e != nil {
					break
				}
				if digest(b) != p.Hash {
					return "", errors.New("source changed during upload")
				}
				for _, target := range p.Targets {
					if !c.allowed(target.URL) {
						return "", errors.New("node URL not allowlisted")
					}
					req, er := http.NewRequestWithContext(ctx, "PUT", target.URL, bytes.NewReader(b))
					if er != nil {
						return "", er
					}
					req.Header.Set("Authorization", "Bearer "+target.Token)
					resp, er := c.blockHTTP().Do(req)
					if er != nil {
						e = er
						break
					}
					data, er := io.ReadAll(io.LimitReader(resp.Body, 65536))
					resp.Body.Close()
					if er != nil {
						e = er
						break
					}
					if resp.StatusCode != 200 {
						e = &HTTPError{resp.StatusCode, string(data)}
						break
					}
				}
				if e != nil {
					break
				}
			}
			if e == nil {
				var out struct {
					ID string `json:"version_id"`
				}
				e = c.request(ctx, "POST", "/v1/uploads/"+plan.ID+"/commit", map[string]any{}, "", &out)
				if e == nil {
					return out.ID, nil
				}
			}
		}
		last = e
		if !retryableUploadError(e) {
			return "", e
		}
		if e = wait(ctx, time.Duration(1<<min(attempt, 3))*time.Second); e != nil {
			return "", e
		}
	}
	return "", fmt.Errorf("upload retry budget exhausted: %w", last)
}
