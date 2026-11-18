package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
)

func (n *Node) task(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	locked := true
	defer func() {
		if locked {
			n.mu.Unlock()
		}
	}()
	b, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	if e != nil {
		output(w, 413, map[string]string{"code": "too_large"})
		return
	}
	if e = n.internal(r, b); e != nil {
		output(w, 403, map[string]string{"code": "internal_rejected", "message": e.Error()})
		return
	}
	var t Task
	if e = json.Unmarshal(b, &t); e != nil || t.Node != n.cfg.ID || !validHash(t.Hash) || t.Generation < 1 {
		output(w, 400, map[string]string{"code": "invalid_task"})
		return
	}
	if t.Kind == "copy" {
		// Authentication/replay journal remains serialized. Slow source reads
		// stage bounded bytes only, with a fresh permit under the physical lock
		// before any destination state is changed.
		n.mu.Unlock()
		locked = false
		if e = n.copyTask(r.Context(), t); e != nil {
			output(w, 409, map[string]string{"code": "task_failed", "message": e.Error()})
			return
		}
		output(w, 200, map[string]bool{"done": true})
		return
	}
	_, e = n.signed(r.Context(), "/internal/v1/permit", map[string]any{"id": t.ID, "owner": t.Owner, "fence": t.Fence, "error": ""})
	if e != nil {
		output(w, 409, map[string]string{"code": "permit_rejected", "message": e.Error()})
		return
	}
	switch t.Kind {
	case "delete":
		reason, _ := t.Payload["reason"].(string)
		if reason != "migration_trim" {
			old, er := n.tombstone(t.Hash)
			if er != nil {
				e = er
				break
			}
			if t.Generation > old {
				if e = n.atomic("state/"+t.Hash, []byte(strconv.FormatInt(t.Generation, 10))); e != nil {
					break
				}
			}
		}
		e = n.blocks.Delete(t.Hash, t.Generation)
		if os.IsNotExist(e) {
			e = nil
		}
	default:
		e = errors.New("invalid task kind")
	}
	if e != nil {
		output(w, 409, map[string]string{"code": "task_failed", "message": e.Error()})
		return
	}
	output(w, 200, map[string]bool{"done": true})
}
