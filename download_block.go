package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

func (c *Client) readBlock(ctx context.Context, id string, p Plan) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			var fresh Plan
			if e := c.request(ctx, "GET", "/v1/download/"+url.PathEscape(id)+"/blocks/"+p.Hash, nil, "", &fresh); e != nil {
				return nil, e
			}
			if fresh.Hash != p.Hash || fresh.Size != p.Size {
				return nil, errors.New("block metadata changed")
			}
			p = fresh
		}
		for _, target := range p.Targets {
			if !c.allowed(target.URL) {
				return nil, errors.New("node URL not allowlisted")
			}
			req, e := http.NewRequestWithContext(ctx, "GET", target.URL, nil)
			if e != nil {
				return nil, e
			}
			req.Header.Set("Authorization", "Bearer "+target.Token)
			response, e := c.blockHTTP().Do(req)
			if e != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				continue
			}
			data, e := io.ReadAll(io.LimitReader(response.Body, p.Size+1))
			response.Body.Close()
			if e == nil && response.StatusCode == 200 && int64(len(data)) == p.Size && digest(data) == p.Hash {
				return data, nil
			}
		}
	}
	return nil, errors.New("replicas unavailable or corrupt after retries")
}
