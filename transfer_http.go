package main

import (
	"errors"
	"net/http"
	"time"
)

// Extend only an already authenticated, size-bounded block operation. Header
// deadlines, capability admission expiry and all metadata deadlines stay intact.
func blockTransferDeadline(w http.ResponseWriter) error {
	controller := http.NewResponseController(w)
	deadline := time.Now().Add(2 * time.Minute)
	for _, set := range []func(time.Time) error{controller.SetReadDeadline, controller.SetWriteDeadline} {
		if err := set(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return err
		}
	}
	return nil
}
