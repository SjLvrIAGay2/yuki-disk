package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"math"
	"net/http"
	"strings"
	"time"
)

type StorageStatus struct {
	Total           int64  `json:"total_bytes"`
	Free            int64  `json:"free_bytes"`
	Available       int64  `json:"available_bytes"`
	ReadOnly        bool   `json:"read_only"`
	Mode            string `json:"mode"`
	Configured      int64  `json:"configured_bytes"`
	Stored          int64  `json:"stored_bytes"`
	AllocationUnit  int64  `json:"allocation_unit_bytes"`
	VolumeAvailable int64  `json:"volume_available_bytes"`
}
type ResourceReport struct {
	Version      int           `json:"version"`
	Boot         string        `json:"boot_id"`
	Sequence     int64         `json:"sequence"`
	SampledAt    time.Time     `json:"sampled_at"`
	Volume       string        `json:"volume"`
	Storage      StorageStatus `json:"storage"`
	Received     int64         `json:"received_bytes"`
	Sent         int64         `json:"sent_bytes"`
	ReceiveRate  float64       `json:"receive_bps"`
	SendRate     float64       `json:"send_bps"`
	ReadSpeed    float64       `json:"read_speed_bps"`
	WriteSpeed   float64       `json:"write_speed_bps"`
	ActiveReads  int64         `json:"active_reads"`
	ActiveWrites int64         `json:"active_writes"`
}

func (p ResourceReport) validate() error {
	if _, e := hex.DecodeString(p.Boot); e != nil {
		return fail(400, "invalid_resources", "invalid boot identity")
	}
	if p.Version != 1 || len(p.Boot) != 48 || p.Sequence < 1 || !validHash(p.Volume) || time.Since(p.SampledAt) > 30*time.Second || time.Until(p.SampledAt) > 5*time.Second {
		return fail(400, "invalid_resources", "invalid resource identity or sample time")
	}
	for _, v := range []int64{p.Storage.Total, p.Storage.Free, p.Storage.Available, p.Storage.Configured, p.Storage.Stored, p.Storage.VolumeAvailable, p.Received, p.Sent, p.ActiveReads, p.ActiveWrites} {
		if v < 0 || v > 9007199254740991 {
			return fail(400, "invalid_resources", "invalid resource counter")
		}
	}
	if (p.Storage.Mode != "filesystem" && p.Storage.Mode != "container") || p.Storage.AllocationUnit < 1 || p.Storage.AllocationUnit > 65536 || 4194304%p.Storage.AllocationUnit != 0 || p.Storage.Available > p.Storage.VolumeAvailable {
		return fail(400, "invalid_resources", "invalid storage allocation model")
	}
	if p.Storage.Available > p.Storage.Free || p.Storage.Free > p.Storage.Total {
		return fail(400, "invalid_resources", "invalid capacity")
	}
	for _, v := range []float64{p.ReceiveRate, p.SendRate, p.ReadSpeed, p.WriteSpeed} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1e12 {
			return fail(400, "invalid_resources", "invalid transfer rate")
		}
	}
	return nil
}
func (s *Server) resources(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if r.URL.RawQuery != "" {
		return fail(400, "invalid_query", "resource reports have no query parameters")
	}
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "forbidden", "node identity required")
	}
	var p ResourceReport
	if e := parse(b, &p); e != nil {
		return e
	}
	if e := p.validate(); e != nil {
		return e
	}
	canonical, e := json.Marshal(p)
	if e != nil {
		return e
	}
	e = s.tx(r.Context(), func(t pgx.Tx) error {
		tag, e := t.Exec(r.Context(), `INSERT INTO node_resources(node_id,sampled_at,boot_id,sequence,payload) VALUES($1,$2,$3,$4,$5)
   ON CONFLICT(node_id) DO UPDATE SET sampled_at=excluded.sampled_at,received_at=now(),boot_id=excluded.boot_id,sequence=excluded.sequence,payload=excluded.payload
   WHERE excluded.sampled_at>node_resources.sampled_at AND (excluded.boot_id<>node_resources.boot_id OR excluded.sequence>node_resources.sequence)`, strings.TrimPrefix(identity, "node:"), p.SampledAt, p.Boot, p.Sequence, canonical)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return fail(409, "stale_resources", "late or repeated sample")
		}
		return nil
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]bool{"accepted": true})
	return nil
}
