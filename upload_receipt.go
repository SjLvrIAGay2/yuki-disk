package main

import (
	"context"
	"errors"
	"net/http"
)

func (n *Node) confirmUpload(ctx context.Context, p Capability) error {
	if p.Upload != "" {
		_, err := n.signed(ctx, "/internal/v1/upload-receipt", map[string]any{"upload_id": p.Upload, "hash": p.Hash, "size": p.Size, "generation": p.Generation})
		if err == nil {
			return nil
		}
		var response *controlError
		// Only an older control without the endpoint uses the original two steps.
		// Ambiguous failures must surface for the existing upload retry identity.
		if !errors.As(err, &response) || response.Status != http.StatusNotFound {
			return err
		}
	}
	if err := n.report(ctx, []DiskItem{{p.Hash, p.Size, p.Generation}}, false); err != nil {
		return err
	}
	if p.Upload != "" {
		_, err := n.signed(ctx, "/internal/v1/upload-proof", map[string]any{"upload_id": p.Upload, "hash": p.Hash, "generation": p.Generation})
		return err
	}
	return nil
}
