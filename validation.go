package main

import (
	"strings"
)

func checkIdempotencyKey(key string) error {
	if len(key) < 8 || len(key) > 128 {
		return fail(400, "invalid_idempotency_key", "8..128 bytes required")
	}
	return nil
}

func validPath(p string) bool {
	if len(p) == 0 || len(p) > 1024 || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\\x00<>:\"|?*") {
		return false
	}
	for _, ch := range p {
		if ch < 32 || ch == 127 {
			return false
		}
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return false
		}
	}
	return true
}

func validHash(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, c := range h {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
