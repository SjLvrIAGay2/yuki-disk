package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func hash(b []byte) string { a := sha256.Sum256(b); return hex.EncodeToString(a[:]) }

func mac(k, s string) []byte {
	h := hmac.New(sha256.New, []byte(k))
	h.Write([]byte(s))
	return h.Sum(nil)
}

func randomID() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
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

func (n *Node) consume(scope, id string, exp int64) error {
	if len(id) != 48 {
		return errors.New("invalid nonce")
	}
	name := "nonces/" + hash([]byte(scope+":"+id))
	f, e := n.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return errors.New("replay rejected")
	}
	_, e = f.WriteString(strconv.FormatInt(exp, 10))
	if e == nil {
		e = syncFile(f)
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e == nil {
		e = n.syncDir("nonces")
	}
	return e
}

func (n *Node) capability(r *http.Request, purpose string) (Capability, error) {
	var p Capability
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	a := strings.Split(raw, ".")
	if len(a) != 2 {
		return p, errors.New("invalid capability")
	}
	sig, e := base64.RawURLEncoding.DecodeString(a[1])
	if e != nil || !hmac.Equal(sig, mac(n.cfg.CapabilityKey, a[0])) {
		return p, errors.New("bad signature")
	}
	b, e := base64.RawURLEncoding.DecodeString(a[0])
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal(b, &p); e != nil {
		return p, e
	}
	if p.Version != 1 || p.Purpose != purpose || p.Node != n.cfg.ID || p.Hash != r.PathValue("hash") || !validHash(p.Hash) || p.Expires <= time.Now().Unix() || p.Expires > time.Now().Unix()+90 || p.Size <= 0 || p.Size > chunkSize || p.Generation < 1 {
		return p, errors.New("capability binding failed")
	}
	return p, n.consume("cap", p.ID, p.Expires)
}

func (n *Node) internal(r *http.Request, b []byte) error {
	id := r.Header.Get("X-Yuki-Identity")
	ts := r.Header.Get("X-Yuki-Time")
	nonce := r.Header.Get("X-Yuki-Nonce")
	t, e := strconv.ParseInt(ts, 10, 64)
	if id != "maintenance" || e != nil || t < time.Now().Unix()-30 || t > time.Now().Unix()+30 {
		return errors.New("maintenance identity required")
	}
	msg := internalMessage(r, id, b, ts, nonce)
	sig, e := hex.DecodeString(r.Header.Get("X-Yuki-Signature"))
	if e != nil || !hmac.Equal(sig, mac(n.cfg.MaintenanceKey, msg)) {
		return errors.New("invalid signature")
	}
	return n.consume(id, nonce, t+61)
}

func (n *Node) cleanNonces() {
	f, e := n.root.Open("nonces")
	if e != nil {
		return
	}
	entries, e := f.ReadDir(-1)
	f.Close()
	if e != nil {
		return
	}
	for _, a := range entries {
		p := "nonces/" + a.Name()
		b, e := n.root.ReadFile(p)
		if e != nil {
			continue
		}
		exp, e := strconv.ParseInt(string(b), 10, 64)
		if e == nil && exp < time.Now().Unix() {
			n.root.Remove(p)
		}
	}
}
