package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (c *P2PConfig) validateIssuer(development bool) error {
	if c.IssuerURL != "" && !validURL(c.IssuerURL, development) {
		return errors.New("invalid P2P issuer address")
	}
	if c.IssuerCAFile != "" && (c.IssuerURL == "" || !strings.HasPrefix(c.IssuerURL, "https://") || !filepath.IsAbs(c.IssuerCAFile)) {
		return errors.New("invalid P2P issuer trust configuration")
	}
	return nil
}

// Private trust is confined to the fixed issuer transport, never to browser
// destinations, node transfers, or the process-wide default transport.
func (c *P2PConfig) issuerClient(development bool) (*http.Client, error) {
	if err := c.validateIssuer(development); err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if c.IssuerCAFile != "" {
		f, err := os.Open(c.IssuerCAFile)
		if err != nil {
			return nil, errors.New("issuer trust unavailable")
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil || !st.Mode().IsRegular() || st.Size() > 1<<20 {
			return nil, errors.New("invalid issuer trust file")
		}
		pem, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
		if err != nil || len(pem) > 1<<20 {
			return nil, errors.New("invalid issuer trust file")
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("invalid issuer trust bundle")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	transport.MaxIdleConnsPerHost = 4
	return &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect forbidden") }}, nil
}
