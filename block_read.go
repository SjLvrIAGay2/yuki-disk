package main

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

func (n *Node) serveBlock(w http.ResponseWriter, r *http.Request) {
	// Bound detached response buffers to eight blocks (32 MiB). Backpressure
	// returns a retryable response without consuming a capability or holding mu.
	select {
	case n.blockReadSlots <- struct{}{}:
		defer func() { <-n.blockReadSlots }()
	default:
		output(w, 503, map[string]string{"code": "transfer_busy"})
		return
	}
	n.mu.Lock()
	p, e := n.capability(r, "get")
	if e != nil {
		n.mu.Unlock()
		output(w, 403, map[string]string{"code": "capability_rejected"})
		return
	}
	if e = blockTransferDeadline(w); e != nil {
		n.mu.Unlock()
		output(w, 503, map[string]string{"code": "transfer_unavailable"})
		return
	}
	b, e := n.blocks.Read(p.Hash, p.Generation)
	if os.IsNotExist(e) {
		n.mu.Unlock()
		output(w, 404, map[string]string{"code": "missing"})
		return
	}
	if e != nil || !validBlock(b, p.Size, p.Hash) {
		n.mu.Unlock()
		output(w, 503, map[string]string{"code": "corrupt"})
		return
	}
	// All physical reads and hash verification completed under the original lock.
	// This authorized immutable response can finish even if its source is deleted.
	n.mu.Unlock()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	n.network.readers.Add(1)
	started := time.Now()
	written, err := w.Write(b)
	n.network.sent.Add(int64(written))
	n.network.readers.Add(-1)
	if err == nil {
		n.network.observe(true, int64(written), time.Since(started))
	}
}
