package main

import (
	"crypto/subtle"
	"net/http"
)

// Monitoring has a dedicated read-only credential, separate from user and node identities.
func monitorAllowed(w http.ResponseWriter, r *http.Request, key string) bool {
	if len(key) < 32 || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Yuki-Monitor")), []byte(key)) != 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		w.Write([]byte(`{"code":"monitor_auth_required"}`))
		return false
	}
	return true
}
