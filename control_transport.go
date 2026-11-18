package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func (c Config) validateControlTransport() error {
	if c.ControlAddress == "" && c.ControlCAFile == "" {
		return nil
	}
	u, e := url.Parse(c.Control)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || !filepath.IsAbs(c.ControlCAFile) {
		return errors.New("fixed control transport requires HTTPS and absolute CA file")
	}
	host, port, e := net.SplitHostPort(c.ControlAddress)
	number, pe := strconv.Atoi(port)
	if e != nil || net.ParseIP(host) == nil || pe != nil || number < 1 || number > 65535 {
		return errors.New("control_address must be a fixed IP and port")
	}
	return nil
}

// Trust and fixed dialing apply only to control requests. Copy sources retain
// their existing URL allowlist and independent standard certificate validation.
func (c Config) newControlClient() (*http.Client, error) {
	if e := c.validateControlTransport(); e != nil {
		return nil, e
	}
	if c.ControlAddress == "" {
		return nil, nil
	}
	f, e := os.Open(c.ControlCAFile)
	if e != nil {
		return nil, errors.New("control CA unavailable")
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Size() > 1<<20 {
		return nil, errors.New("invalid control CA file")
	}
	pem, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(pem) > 1<<20 {
		return nil, errors.New("invalid control CA file")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, errors.New("invalid control CA bundle")
	}
	u, _ := url.Parse(c.Control)
	port := u.Port()
	if port == "" {
		port = "443"
	}
	expected := net.JoinHostPort(u.Hostname(), port)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != expected {
			return nil, errors.New("control destination rejected")
		}
		return dialer.DialContext(ctx, network, c.ControlAddress)
	}
	return &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect forbidden") }}, nil
}
