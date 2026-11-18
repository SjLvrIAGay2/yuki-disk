package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Bounded windows issue independent single-use capabilities. The node verifies
// the actual size/revision before reading; authorization never requires control
// to fetch file bytes or to make a remote stat for each range.
func (s *Server) mountRanges(w http.ResponseWriter, r *http.Request, b []byte, owner string) error {
	var p struct {
		Offsets []int64 `json:"offsets"`
		Size    int64   `json:"size"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if p.Size < 1 || p.Size > 9007199254740991 || len(p.Offsets) < 1 || len(p.Offsets) > 6 {
		return fail(400, "invalid_ranges", "invalid range window")
	}
	id := r.PathValue("id")
	var f FileIdentity
	if !strings.HasPrefix(id, "m.") || decodeToken(s.cfg.CapabilityKey, strings.TrimPrefix(id, "m."), &f) != nil || f.Purpose != "mount-version" || f.Owner != owner {
		return fail(404, "not_found", "file not found")
	}
	m, ok := s.mountFor(r.Context(), f.Path)
	if !ok || m.ID != f.Mount || m.Node != f.Node {
		return fail(404, "not_found", "mount changed")
	}
	if e := currentMount(r.Context(), s.query(r.Context()), m); e != nil {
		return e
	}
	n, ok := s.nodeConfig(m.Node)
	if !ok {
		return fail(503, "unavailable", "file unavailable")
	}
	seen := map[int64]bool{}
	items := []map[string]any{}
	for _, start := range p.Offsets {
		if start < 0 || start >= p.Size || start%chunkSize != 0 || seen[start] {
			return fail(400, "invalid_ranges", "invalid range window")
		}
		seen[start] = true
		length := min(int64(chunkSize), p.Size-start)
		cap := f
		cap.Purpose = "file-range"
		cap.Start = start
		cap.Length = length
		cap.Size = p.Size
		cap.Expires = time.Now().Add(time.Minute).Unix()
		cap.ID = randomID()
		items = append(items, map[string]any{"offset": start, "size": length, "revision": f.Revision, "range": fmt.Sprintf("bytes=%d-%d", start, start+length-1), "target": map[string]any{"node": n.ID, "url": n.URL + "/v1/content/range", "p2p": s.hasP2P(n.ID), "token": encodeToken(s.cfg.CapabilityKey, cap)}})
	}
	jsonOut(w, 200, map[string]any{"ranges": items})
	return nil
}

func validFileRange(f FileIdentity, value string) bool {
	return f.Size > 0 && f.Size <= 9007199254740991 && f.Start >= 0 && f.Start < f.Size && f.Start%chunkSize == 0 && f.Length > 0 && f.Length <= chunkSize && f.Length <= f.Size-f.Start && value == fmt.Sprintf("bytes=%d-%d", f.Start, f.Start+f.Length-1)
}
