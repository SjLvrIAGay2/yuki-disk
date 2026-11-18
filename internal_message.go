package main

import (
	"net/http"
	"strings"
)

// Encoding only: each caller still checks its own role, key, time and nonce.
func internalMessage(r *http.Request, identity string, body []byte, timestamp, nonce string) string {
	return strings.Join([]string{identity, r.Method, r.URL.EscapedPath(), r.URL.RawQuery, hash(body), timestamp, nonce}, "\n")
}
