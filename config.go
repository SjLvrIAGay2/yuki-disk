package main

import (
	"net"
	"net/url"
	"strings"
)

type Config struct {
	MonitorKey  string   `json:"monitor_key,omitempty"`
	Control     string   `json:"control"`
	Key         string   `json:"key"`
	Development bool     `json:"development"`
	Nodes       []string `json:"nodes"`
	Listen      string   `json:"listen"`
}

func validURL(raw string, dev bool) bool { return validURLPath(raw, dev, false) }

// validURLPath validates a service address. Node URLs must be bare origins since
// the worker appends node routes, but the control plane's address may carry a path
// prefix when it is published behind a reverse proxy at a subpath such as /api.
func validURLPath(raw string, dev bool, allowPath bool) bool {
	u, e := url.Parse(raw)
	if e != nil || u.User != nil || u.Hostname() == "" || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(raw, " \t\n\r\\") {
		return false
	}
	if allowPath {
		if u.Path == "/" || strings.HasSuffix(u.Path, "/") {
			return false
		}
	} else if raw != u.Scheme+"://"+u.Host {
		return false
	}
	return u.Scheme == "https" || dev && u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback())
}
