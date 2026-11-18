package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Client struct {
	cfg     Config
	http    *http.Client
	token   string
	tokenMu sync.RWMutex
	loginMu sync.Mutex
}

type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body) }

func (c *Client) request(ctx context.Context, method, path string, p any, idem string, out any) error {
	return c.requestAttempt(ctx, method, path, p, idem, out, true)
}
func (c *Client) requestAttempt(ctx context.Context, method, path string, p any, idem string, out any, retry bool) error {
	var b []byte
	var e error
	if p != nil {
		b, e = json.Marshal(p)
		if e != nil {
			return e
		}
	}
	r, e := http.NewRequestWithContext(ctx, method, c.cfg.Control+path, bytes.NewReader(b))
	if e != nil {
		return e
	}
	r.Header.Set("Content-Type", "application/json")
	c.tokenMu.RLock()
	token := c.token
	c.tokenMu.RUnlock()
	if token != "" && path != "/v1/session" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if idem != "" {
		r.Header.Set("Idempotency-Key", idem)
	}
	resp, e := c.http.Do(r)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	body, e := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if e != nil {
		return e
	}
	if resp.StatusCode == 401 && retry && path != "/v1/session" {
		if e := c.loginFor(ctx, token); e != nil {
			return e
		}
		return c.requestAttempt(ctx, method, path, p, idem, out, false)
	}
	if resp.StatusCode != 200 {
		return &HTTPError{resp.StatusCode, string(body)}
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

func (c *Client) login(ctx context.Context) error {
	return c.loginFor(ctx, "")
}
func (c *Client) loginFor(ctx context.Context, rejected string) error {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	c.tokenMu.RLock()
	alreadyRenewed := rejected != "" && c.token != "" && c.token != rejected
	c.tokenMu.RUnlock()
	if alreadyRenewed {
		return nil
	}
	var out struct {
		Token string `json:"token"`
	}
	e := c.request(ctx, "POST", "/v1/session", map[string]string{"username": c.cfg.Username, "password": c.cfg.Password, "client": "cli"}, "", &out)
	c.tokenMu.Lock()
	c.token = out.Token
	c.tokenMu.Unlock()
	return e
}

func (c *Client) allowed(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	for _, base := range c.cfg.Nodes {
		if strings.HasPrefix(raw, base+"/v1/blocks/") && u.Scheme+"://"+u.Host == base {
			return true
		}
	}
	return false
}
