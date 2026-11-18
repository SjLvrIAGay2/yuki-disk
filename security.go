package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func randomID() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func mac(key, msg string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return h.Sum(nil)
}
func encodeToken(key string, p any) string {
	b, _ := json.Marshal(p)
	s := base64.RawURLEncoding.EncodeToString(b)
	return s + "." + base64.RawURLEncoding.EncodeToString(mac(key, s))
}
func decodeToken(key, s string, p any) error {
	a := strings.Split(s, ".")
	if len(a) != 2 {
		return errors.New("invalid token")
	}
	sig, e := base64.RawURLEncoding.DecodeString(a[1])
	if e != nil || !hmac.Equal(sig, mac(key, a[0])) {
		return errors.New("invalid signature")
	}
	b, e := base64.RawURLEncoding.DecodeString(a[0])
	if e != nil {
		return e
	}
	return json.Unmarshal(b, p)
}

type Session struct {
	Version int    `json:"v"`
	Purpose string `json:"purpose"`
	Expires int64  `json:"exp"`
	ID      string `json:"jti"`
}
type Capability struct {
	Upload     string `json:"upload_id,omitempty"`
	Version    int    `json:"v"`
	Purpose    string `json:"purpose"`
	Node       string `json:"node"`
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Generation int64  `json:"generation"`
	Expires    int64  `json:"exp"`
	ID         string `json:"jti"`
}

func (s *Server) cap(n, p, h string, size, g int64) string {
	return encodeToken(s.cfg.CapabilityKey, Capability{"", 1, p, n, h, size, g, time.Now().Add(60 * time.Second).Unix(), randomID()})
}
func (s *Server) internal(r *http.Request, body []byte) (string, error) {
	id := r.Header.Get("X-Yuki-Identity")
	key := ""
	if (id == "web" || id == "status") && r.URL.Path == "/internal/v1/logs" {
		key = s.cfg.LogKeys[id]
	} else if id == "maintenance" {
		key = s.cfg.MaintenanceKey
	} else if strings.HasPrefix(id, "node:") {
		if n, ok := s.nodeConfig(strings.TrimPrefix(id, "node:")); ok {
			key = n.Key
		}
	}
	ts := r.Header.Get("X-Yuki-Time")
	nonce := r.Header.Get("X-Yuki-Nonce")
	t, e := strconv.ParseInt(ts, 10, 64)
	if key == "" || e != nil || time.Now().Unix()-t > 30 || t-time.Now().Unix() > 30 || len(nonce) != 48 {
		return "", errors.New("invalid internal identity/time/nonce")
	}
	// Internal callers build the signed URL from the control plane's public base, so
	// when that base carries a mount prefix the signed path includes it while the
	// routed path does not. Accept either exact form and nothing else.
	signed := func(p string) []byte {
		return mac(key, strings.Join([]string{id, r.Method, p, r.URL.RawQuery, digest(body), ts, nonce}, "\n"))
	}
	sig, e := hex.DecodeString(r.Header.Get("X-Yuki-Signature"))
	if e != nil {
		return "", errors.New("invalid internal signature")
	}
	routed := r.URL.EscapedPath()
	onWire := wirePath(r)
	if !hmac.Equal(sig, signed(onWire)) && (onWire == routed || !hmac.Equal(sig, signed(routed))) {
		return "", errors.New("invalid internal signature")
	}
	_, e = s.db.Exec(r.Context(), "INSERT INTO replay_nonces(identity,nonce,expires_at) VALUES($1,$2,now()+interval '61 seconds')", id, nonce)
	if e != nil {
		return "", fmt.Errorf("replayed request or database unavailable: %w", e)
	}
	return id, nil
}
