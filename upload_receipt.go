package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

// A durable node receipt publishes its replica and upload ownership proof in
// one existing authoritative transaction. It never accepts file bytes.
func (s *Server) uploadReceipt(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "node_required", "node identity required")
	}
	var p struct {
		Upload     string `json:"upload_id"`
		Hash       string `json:"hash"`
		Size       int64  `json:"size"`
		Generation int64  `json:"generation"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if p.Upload == "" || !validHash(p.Hash) || p.Size <= 0 || p.Size > chunkSize || p.Generation < 1 {
		return fail(400, "invalid_receipt", "invalid upload receipt")
	}
	report, _ := json.Marshal(Report{Items: []DiskItem{{p.Hash, p.Size, p.Generation}}})
	proof, _ := json.Marshal(map[string]any{"upload_id": p.Upload, "hash": p.Hash, "generation": p.Generation})
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		request := r.WithContext(context.WithValue(r.Context(), requestTransactionKey{}, t))
		buffered := &committedResponse{header: http.Header{}}
		if e := s.report(buffered, request, report, identity); e != nil {
			return e
		}
		return s.uploadProof(buffered, request, proof, identity)
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]string{"status": "confirmed"})
	return nil
}
