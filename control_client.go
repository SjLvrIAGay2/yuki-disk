package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type controlError struct {
	Status int
	Body   string
}

func (e *controlError) Error() string { return fmt.Sprintf("control %d: %s", e.Status, e.Body) }
func (n *Node) signed(ctx context.Context, path string, p any) ([]byte, error) {
	b, e := json.Marshal(p)
	if e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", n.cfg.Control+path, bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	id := "node:" + n.cfg.ID
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomID()
	msg := internalMessage(req, id, b, ts, nonce)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Yuki-Identity", id)
	req.Header.Set("X-Yuki-Time", ts)
	req.Header.Set("X-Yuki-Nonce", nonce)
	req.Header.Set("X-Yuki-Signature", hex.EncodeToString(mac(n.cfg.Key, msg)))
	client := n.controlClient
	if client == nil {
		client = n.client
	}
	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	body, e := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if e != nil {
		return nil, e
	}
	if resp.StatusCode != 200 {
		return nil, &controlError{resp.StatusCode, string(body)}
	}
	return body, nil
}

func (n *Node) report(ctx context.Context, items []DiskItem, full bool) error {
	if full {
		snapshot := randomID()
		for page, offset := 0, 0; ; page, offset = page+1, offset+1000 {
			end := offset + 1000
			if end > len(items) {
				end = len(items)
			}
			_, e := n.signed(ctx, "/internal/v1/inventory-page", map[string]any{"snapshot": snapshot, "page": page, "final": end == len(items), "items": items[offset:end]})
			if e != nil {
				return e
			}
			if end == len(items) {
				return nil
			}
		}
	}
	_, e := n.signed(ctx, "/internal/v1/report", map[string]any{"items": items, "full": full})
	return e
}
