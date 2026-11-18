package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

type FileSource struct {
	URL      string `json:"url"`
	Token    string `json:"token"`
	Revision string `json:"revision"`
}

func (c *Client) readFile(ctx context.Context, source FileSource, size int64, out io.Writer) (int64, error) {
	u, e := url.Parse(source.URL)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/v1/content" || size < 0 || size > 9007199254740991 {
		return 0, errors.New("invalid file source")
	}
	allowed := false
	for _, base := range c.cfg.Nodes {
		if source.URL == base+"/v1/content" {
			allowed = true
		}
	}
	if !allowed {
		return 0, errors.New("untrusted file source")
	}
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if e != nil {
		return 0, e
	}
	req.Header.Set("Authorization", "Bearer "+source.Token)
	client := *c.http
	client.Timeout = 0
	resp, e := client.Do(req)
	if e != nil {
		return 0, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("ETag") != "\""+source.Revision+"\"" {
		return 0, errors.New("file changed or unavailable")
	}
	return io.Copy(out, io.LimitReader(resp.Body, size+1))
}
