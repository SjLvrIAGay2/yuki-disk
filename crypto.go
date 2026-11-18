package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func randomID() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func idempotencyKey(key string) string {
	if key == "" {
		return randomID()
	}
	return key
}
