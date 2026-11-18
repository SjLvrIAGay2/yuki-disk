package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type ExportConfig struct {
	ID   string `json:"id"`
	Root string `json:"root"`
}
type Config struct {
	MonitorKey       string         `json:"monitor_key,omitempty"`
	Exports          []ExportConfig `json:"exports,omitempty"`
	ID               string         `json:"id"`
	Listen           string         `json:"listen"`
	Data             string         `json:"data"`
	Control          string         `json:"control"`
	ControlAddress   string         `json:"control_address,omitempty"`
	ControlCAFile    string         `json:"control_ca_file,omitempty"`
	Key              string         `json:"key"`
	CapabilityKey    string         `json:"capability_key"`
	MaintenanceKey   string         `json:"maintenance_key"`
	Development      bool           `json:"development"`
	Origins          []string       `json:"origins"`
	Sources          []string       `json:"sources"`
	TLSCert          string         `json:"tls_cert"`
	TLSKey           string         `json:"tls_key"`
	CrashAfterRename bool           `json:"crash_after_rename"`
	StorageMode      string         `json:"storage_mode"`
	CapacityBytes    int64          `json:"capacity_bytes"`
	ContainerFile    string         `json:"container_file"`
}

func validURL(raw string, dev bool) bool { return validURLPath(raw, dev, false) }

// validURLPath validates a service address. Node URLs must be bare origins
// because the control plane appends node routes to them, but the control plane's
// own address may carry a path prefix when it is published behind a reverse proxy
// at a subpath such as /api.
func validURLPath(raw string, dev bool, allowPath bool) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(raw, " \t\n\r\\") {
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

func readConfig(p string) (Config, error) {
	var c Config
	b, e := os.ReadFile(p)
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

	if c.MonitorKey != "" && (len(c.MonitorKey) < 32 || c.MonitorKey == c.Key || c.MonitorKey == c.CapabilityKey || c.MonitorKey == c.MaintenanceKey) {
		return c, errors.New("monitor_key must contain at least 32 bytes")
	}
	if e = c.storageDefaults(); e != nil {
		return c, e
	}
	if e = c.validateControlTransport(); e != nil {
		return c, e
	}
	if c.ID == "" || c.Listen == "" || !filepath.IsAbs(c.Data) || !validURLPath(c.Control, c.Development, true) || len(c.Key) < 32 || len(c.CapabilityKey) < 32 || len(c.MaintenanceKey) < 32 || c.Key == c.MaintenanceKey || c.Key == c.CapabilityKey || c.MaintenanceKey == c.CapabilityKey {
		return c, errors.New("invalid config")
	}
	if !c.Development && (c.TLSCert == "" || c.TLSKey == "" || c.CrashAfterRename) {
		return c, errors.New("production requires TLS and forbids failpoints")
	}
	if !c.Development {
		for _, key := range []string{c.Key, c.CapabilityKey, c.MaintenanceKey, c.MonitorKey} {
			if strings.Contains(strings.ToLower(key), "development") {
				return c, errors.New("production rejects development credentials")
			}
		}
	}
	for _, a := range append(c.Origins, c.Sources...) {
		if !validURL(a, c.Development) {
			return c, errors.New("invalid origin/source")
		}
	}
	return c, nil
}

func (c *Config) storageDefaults() error {
	if c.StorageMode == "" {
		c.StorageMode = "filesystem"
	}
	if c.StorageMode != "filesystem" && c.StorageMode != "container" {
		return errors.New("invalid storage_mode")
	}
	if c.CapacityBytes < 0 || c.CapacityBytes > 1<<40 {
		return errors.New("capacity_bytes must be between zero and 1 TiB")
	}
	if c.StorageMode == "filesystem" {
		if c.ContainerFile != "" {
			return errors.New("container_file requires container mode")
		}
		return nil
	}
	if c.CapacityBytes < 16<<20 {
		return errors.New("container requires capacity_bytes of at least 16 MiB")
	}
	if c.ContainerFile == "" {
		c.ContainerFile = "blocks.pack"
	}
	if filepath.Base(c.ContainerFile) != c.ContainerFile || c.ContainerFile == "." || c.ContainerFile == ".." || strings.ContainsAny(c.ContainerFile, "/\\\x00\r\n") || len(c.ContainerFile) > 128 {
		return errors.New("invalid container_file basename")
	}
	if runtime.GOOS == "windows" && (!filepath.IsLocal(c.ContainerFile) || strings.ContainsAny(c.ContainerFile, ":?*") || strings.HasSuffix(c.ContainerFile, ".") || strings.HasSuffix(c.ContainerFile, " ")) {
		return errors.New("invalid Windows container filename")
	}
	for _, reserved := range []string{"blocks", "state", "nonces", "tmp", "process.lock", "storage-format.json"} {
		if c.ContainerFile == reserved || (runtime.GOOS == "windows" || runtime.GOOS == "darwin") && strings.EqualFold(c.ContainerFile, reserved) {
			return errors.New("reserved container filename")
		}
	}
	return nil
}
