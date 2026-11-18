package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type P2PConfig struct {
	Broker       string `json:"broker"`
	Identity     string `json:"identity"`
	Key          string `json:"key"`
	IssuerURL    string `json:"issuer_url,omitempty"`
	IssuerCAFile string `json:"issuer_ca_file,omitempty"`
}

var p2pID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func (c Config) validateP2P(keys map[string]bool) error {
	if c.P2P != nil && (!validURL(c.P2P.Broker, c.Development) || !p2pID.MatchString(c.P2P.Identity) || len(c.P2P.Key) < 32 || keys[c.P2P.Key] || len(c.Origins) == 0) {
		return errors.New("invalid P2P issuer configuration")
	}
	if c.P2P != nil {
		if e := c.P2P.validateIssuer(c.Development); e != nil {
			return e
		}
	}
	seen := map[string]bool{}
	for _, node := range c.Nodes {
		if node.P2PService != "" {
			if c.P2P == nil || !p2pID.MatchString(node.P2PService) || seen[node.P2PService] {
				return errors.New("invalid P2P node service")
			}
			seen[node.P2PService] = true
		}
	}
	return nil
}
func (s *Server) hasP2P(node string) bool {
	if s.cfg.P2P == nil {
		return false
	}
	n, ok := s.nodeConfig(node)
	return ok && n.P2PService != ""
}

// This endpoint only exchanges short-lived connection metadata. Block bytes remain
// on browser <-> agent <-> node paths, protected by the original node capability.
func (s *Server) p2pTicket(w http.ResponseWriter, r *http.Request, body []byte, _ string) error {
	if s.cfg.P2P == nil {
		return fail(404, "p2p_unavailable", "connection unavailable")
	}
	if e := parse(body, &struct{}{}); e != nil {
		return e
	}
	origin := r.Header.Get("Origin")
	allowed := false
	for _, v := range s.cfg.Origins {
		allowed = allowed || origin == v
	}
	if !allowed {
		return fail(403, "origin_denied", "origin required")
	}
	n, _ := s.nodeConfig(r.PathValue("node"))
	service := n.P2PService
	if service == "" {
		return fail(404, "p2p_unavailable", "connection unavailable")
	}
	payload, _ := json.Marshal(map[string]string{"service": service, "origin": origin})
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	issuer := s.cfg.P2P.Broker
	if s.cfg.P2P.IssuerURL != "" {
		issuer = s.cfg.P2P.IssuerURL
	}
	req, e := http.NewRequestWithContext(ctx, "POST", issuer+"/v1/browser-ticket", bytes.NewReader(payload))
	if e != nil {
		return e
	}
	ts, nonce := strconv.FormatInt(time.Now().Unix(), 10), randomID()
	msg := strings.Join([]string{s.cfg.P2P.Identity, req.Method, req.URL.EscapedPath(), "", digest(payload), ts, nonce}, "\n")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NAT-Identity", s.cfg.P2P.Identity)
	req.Header.Set("X-NAT-Time", ts)
	req.Header.Set("X-NAT-Nonce", nonce)
	req.Header.Set("X-NAT-Signature", hex.EncodeToString(mac(s.cfg.P2P.Key, msg)))
	s.p2pHTTPOnce.Do(func() { s.p2pHTTP, s.p2pHTTPErr = s.cfg.P2P.issuerClient(s.cfg.Development) })
	if s.p2pHTTPErr != nil {
		return fail(503, "p2p_unavailable", "connection unavailable")
	}
	client := s.p2pHTTP
	response, e := client.Do(req)
	if e != nil {
		return fail(503, "p2p_unavailable", "connection unavailable")
	}
	defer response.Body.Close()
	data, e := io.ReadAll(io.LimitReader(response.Body, 4097))
	if e != nil || len(data) > 4096 || response.StatusCode != 200 {
		return fail(503, "p2p_unavailable", "connection unavailable")
	}
	var ticket struct {
		Identity string `json:"identity"`
		Key      string `json:"key"`
		Service  string `json:"service"`
		Expires  int64  `json:"expires_at"`
	}
	if json.Unmarshal(data, &ticket) != nil || !p2pID.MatchString(ticket.Identity) || len(ticket.Key) < 32 || len(ticket.Key) > 256 || ticket.Service != service || ticket.Expires <= time.Now().Unix() || ticket.Expires > time.Now().Unix()+125 {
		return fail(503, "p2p_unavailable", "connection unavailable")
	}
	jsonOut(w, 200, map[string]any{"broker": s.cfg.P2P.Broker, "identity": ticket.Identity, "key": ticket.Key, "service": service, "expires_at": ticket.Expires})
	return nil
}
