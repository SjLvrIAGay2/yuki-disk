package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

func (w *Worker) call(ctx context.Context, endpoint string, p any, out any) error {
	b, e := json.Marshal(p)
	if e != nil {
		return e
	}
	r, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if e != nil {
		return e
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomID()
	sum := sha256.Sum256(b)
	msg := strings.Join([]string{"maintenance", r.Method, r.URL.EscapedPath(), r.URL.RawQuery, hex.EncodeToString(sum[:]), ts, nonce}, "\n")
	h := hmac.New(sha256.New, []byte(w.cfg.Key))
	h.Write([]byte(msg))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Yuki-Identity", "maintenance")
	r.Header.Set("X-Yuki-Time", ts)
	r.Header.Set("X-Yuki-Nonce", nonce)
	r.Header.Set("X-Yuki-Signature", hex.EncodeToString(h.Sum(nil)))
	resp, e := w.client.Do(r)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if strings.HasPrefix(endpoint, w.cfg.Control+"/") && (resp.StatusCode == 200 || resp.StatusCode == 409) {
		w.lastContact.Store(time.Now().Unix())
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if e != nil {
		return e
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
