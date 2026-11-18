package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// At most one bounded block is staged. No physical state is changed while the
// network is outside mu; the final permit and persistent generation checks run
// again after taking the same lock as uploads, deletion and inventory.
func (n *Node) copyTask(ctx context.Context, t Task) error {
	if !n.copyActive.CompareAndSwap(false, true) {
		return errors.New("copy busy")
	}
	defer n.copyActive.Store(false)
	source, _ := t.Payload["source_url"].(string)
	token, _ := t.Payload["source_token"].(string)
	size, _ := t.Payload["size"].(float64)
	if size < 1 || size > chunkSize || size != float64(int64(size)) || token == "" {
		return errors.New("invalid copy size or authority")
	}
	allowed := false
	for _, u := range n.cfg.Sources {
		allowed = allowed || source == u
	}
	if !allowed {
		return errors.New("source not allowlisted")
	}
	permit := func() error {
		_, e := n.signed(ctx, "/internal/v1/permit", map[string]any{"id": t.ID, "owner": t.Owner, "fence": t.Fence, "error": ""})
		return e
	}
	if e := permit(); e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, "GET", source+"/v1/blocks/"+t.Hash, nil)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, e := n.client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("copy source unavailable")
	}
	started := time.Now()
	n.network.writers.Add(1)
	data, e := io.ReadAll(io.LimitReader(countedReader{resp.Body, &n.network.received}, int64(size)+1))
	n.network.writers.Add(-1)
	if e != nil {
		return e
	}
	if !validBlock(data, int64(size), t.Hash) {
		return errors.New("copy size/hash mismatch")
	}
	n.network.observe(false, int64(len(data)), time.Since(started))
	n.mu.Lock()
	defer n.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := permit(); e != nil {
		return e
	}
	// putStored has the same generation, backend and durable commit boundary as PUT.
	return n.putStored(Capability{Hash: t.Hash, Size: int64(size), Generation: t.Generation}, data)
}
