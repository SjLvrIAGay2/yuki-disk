package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
)

type NodeConfig struct {
	ID             string `json:"id"`
	URL            string `json:"url"`
	Tier           string `json:"tier"`
	Key            string `json:"key"`
	ReadBandwidth  int64  `json:"read_bandwidth_bps"`
	WriteBandwidth int64  `json:"write_bandwidth_bps"`
	ReserveBytes   int64  `json:"reserve_bytes"`
	P2PService     string `json:"p2p_service,omitempty"`
}
type MountConfig struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Node   string `json:"node"`
	Export string `json:"export"`
}
type BootstrapAccount struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type Config struct {
	LogKeys        map[string]string `json:"log_keys,omitempty"`
	MonitorKey     string            `json:"monitor_key,omitempty"`
	BootstrapAdmin *BootstrapAccount `json:"bootstrap_admin,omitempty"`
	LegacyUsername string            `json:"legacy_username,omitempty"`
	Mounts         []MountConfig     `json:"mounts,omitempty"`
	Listen         string            `json:"listen"`
	// ControlURL is this control plane's public base address as clients reach it.
	// It may carry a path prefix when the API is published behind a reverse proxy
	// at a subpath (for example https://disk.example.com/api). Unlike node and
	// broker addresses, no route is appended from configuration, but browser
	// session cookies are scoped to its path.
	ControlURL     string            `json:"control_url,omitempty"`
	Database       string            `json:"database"`
	Development    bool              `json:"development"`
	Password       string            `json:"password"`
	SessionKey     string            `json:"session_key"`
	CapabilityKey  string            `json:"capability_key"`
	MaintenanceKey string            `json:"maintenance_key"`
	Nodes          []NodeConfig      `json:"nodes"`
	Origins        []string          `json:"origins"`
	Replicas       int               `json:"replicas"`
	TrashSeconds   int               `json:"trash_seconds"`
	GCSeconds      int               `json:"gc_seconds"`
	UploadSeconds  int               `json:"upload_seconds"`
	ColdSeconds    int               `json:"cold_seconds"`
	TLSCert        string            `json:"tls_cert"`
	TLSKey         string            `json:"tls_key"`
	P2P            *P2PConfig        `json:"p2p,omitempty"`
}

func loadConfig(path string) (Config, error) {
	c := Config{Listen: "127.0.0.1:8080", Replicas: 2, TrashSeconds: 604800, GCSeconds: 86400, UploadSeconds: 900, ColdSeconds: 604800}
	b, e := os.ReadFile(path)
	if e != nil {
		return c, e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&c); e != nil {
		return c, e
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return c, errors.New("trailing config data")
	}
	return c, c.validate()
}
func validURL(raw string, dev bool) bool { return validURLPath(raw, dev, false) }

// validURLPath validates a service address. When allowPath is false the value
// must be a bare origin (scheme://host[:port]) with no path, query or fragment,
// which is what every address the control plane appends a route to must be.
// The control plane's own public address may carry a path prefix, because it can
// be published behind a reverse proxy at a subpath such as /api.
func validURLPath(raw string, dev bool, allowPath bool) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if allowPath {
		if u.Path != "" && u.Path != "/" && strings.HasSuffix(u.Path, "/") {
			return false
		}
		if u.Path == "/" {
			return false
		}
	} else if raw != u.Scheme+"://"+u.Host {
		return false
	}
	// A path prefix is only meaningful for an absolute base; reject anything that
	// could be parsed differently by a client.
	if strings.ContainsAny(raw, " \t\n\r\\") {
		return false
	}
	return u.Scheme == "https" || dev && u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback())
}

// publicPrefix is the path component of the control plane's public address, and
// is empty when it serves at the root of its own origin. Session cookies are
// scoped to it so they are not sent to unrelated paths when the API is published
// under a subpath of a shared host (for example https://disk.example.com/api).
func (c Config) publicPrefix() string {
	u, e := url.Parse(c.ControlURL)
	if e != nil || u.Path == "" || u.Path == "/" {
		return ""
	}
	return strings.TrimSuffix(u.Path, "/")
}

func (c Config) validate() error {
	if c.MonitorKey != "" && len(c.MonitorKey) < 32 {
		return errors.New("monitor_key must contain at least 32 bytes")
	}
	if c.ControlURL != "" && !validURLPath(c.ControlURL, c.Development, true) {
		return errors.New("invalid public control address")
	}

	if c.Database == "" || c.Replicas < 1 || len(c.Nodes) < c.Replicas || c.TrashSeconds < 1 || c.GCSeconds < 1 || c.UploadSeconds < 120 || c.ColdSeconds < 1 {
		return errors.New("invalid database, replication or windows")
	}
	seen := map[string]bool{}
	keys := map[string]bool{}
	if c.MonitorKey != "" {
		if !c.Development && strings.Contains(strings.ToLower(c.MonitorKey), "development") {
			return errors.New("production rejects development monitoring credentials")
		}
		keys[c.MonitorKey] = true
	}
	for _, k := range []string{c.Password, c.SessionKey, c.CapabilityKey, c.MaintenanceKey} {
		if !c.Development && strings.Contains(strings.ToLower(k), "development") {
			return errors.New("production rejects development credentials")
		}
		if len(k) < 32 || keys[k] {
			return errors.New("secrets must be distinct and >=32 bytes")
		}
		keys[k] = true
	}
	for id, key := range c.LogKeys {
		if (id != "web" && id != "status") || len(key) < 32 || keys[key] || (!c.Development && strings.Contains(strings.ToLower(key), "development")) {
			return errors.New("invalid independent log credential")
		}
		keys[key] = true
	}
	for _, n := range c.Nodes {
		if n.ReadBandwidth < 0 || n.WriteBandwidth < 0 || n.ReadBandwidth > 1000000000000 || n.WriteBandwidth > 1000000000000 || n.ReserveBytes < 0 || n.ReserveBytes > 9007199254740991 {
			return errors.New("invalid node scheduling budget")
		}
		if n.ID == "" || strings.ContainsAny(n.ID, "/\\\x00 \n") || seen[n.ID] || !validURL(n.URL, c.Development) || (n.Tier != "hot" && n.Tier != "cold") || len(n.Key) < 32 || keys[n.Key] {
			return errors.New("invalid or duplicate node")
		}
		seen[n.ID] = true
		keys[n.Key] = true
	}
	for i, m := range c.Mounts {
		if !validPath(m.Path) || !validPath(m.ID) || strings.Contains(m.ID, "/") || !validPath(m.Export) || strings.Contains(m.Export, "/") || !seen[m.Node] {
			return errors.New("invalid mount")
		}
		for _, other := range c.Mounts[:i] {
			if m.ID == other.ID || m.Path == other.Path || strings.HasPrefix(m.Path, other.Path+"/") || strings.HasPrefix(other.Path, m.Path+"/") {
				return errors.New("overlapping mount")
			}
		}
	}
	for _, origin := range c.Origins {
		if !validURL(origin, c.Development) {
			return errors.New("invalid CORS origin")
		}
	}
	if e := c.validateP2P(keys); e != nil {
		return e
	}
	if !c.Development && c.BootstrapAdmin != nil && strings.Contains(strings.ToLower(c.BootstrapAdmin.Password), "development") {
		return errors.New("production rejects development administrator credentials")
	}
	if !c.Development && (c.TLSCert == "" || c.TLSKey == "") {
		return errors.New("production requires TLS")
	}
	return nil
}

func (s *Server) nodeConfig(id string) (NodeConfig, bool) {
	for _, n := range s.cfg.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return NodeConfig{}, false
}
