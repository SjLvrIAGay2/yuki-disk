package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type contentReader struct {
	node   *Node
	ctx    context.Context
	plan   ContentPlan
	offset int64
	cached int
	data   []byte
	starts []int64
}

func newContentReader(n *Node, r *http.Request, p ContentPlan) (*contentReader, error) {
	reader := &contentReader{node: n, ctx: r.Context(), plan: p, cached: -1}
	var total int64
	for _, b := range p.Blocks {
		if !validHash(b.Hash) || b.Size <= 0 || b.Size > chunkSize || total > 9007199254740991-b.Size {
			return nil, errors.New("invalid manifest")
		}
		reader.starts = append(reader.starts, total)
		total += b.Size
	}
	if total != p.Size {
		return nil, errors.New("invalid manifest size")
	}
	return reader, nil
}
func (r *contentReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += r.offset
	case io.SeekEnd:
		offset += r.plan.Size
	default:
		return 0, errors.New("invalid seek")
	}
	if offset < 0 || offset > r.plan.Size {
		return 0, errors.New("invalid offset")
	}
	r.offset = offset
	return offset, nil
}
func (r *contentReader) Read(out []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.offset >= r.plan.Size {
		return 0, io.EOF
	}
	index := 0
	for index+1 < len(r.starts) && r.starts[index+1] <= r.offset {
		index++
	}
	if r.cached != index {
		data, e := r.fetch(index)
		if e != nil {
			return 0, e
		}
		r.data = data
		r.cached = index
	}
	n := copy(out, r.data[r.offset-r.starts[index]:])
	r.offset += int64(n)
	return n, nil
}
func (r *contentReader) fetch(index int) ([]byte, error) {
	b := r.plan.Blocks[index]
	raw, e := r.node.signed(r.ctx, "/internal/v1/content", ContentRequest{Path: r.plan.Path, Version: r.plan.Version, Hash: b.Hash})
	if e != nil {
		return nil, e
	}
	var fresh ContentBlock
	if json.Unmarshal(raw, &fresh) != nil || fresh.Hash != b.Hash || fresh.Size != b.Size {
		return nil, errors.New("invalid block plan")
	}
	for _, target := range fresh.Targets {
		valid := false
		for _, origin := range r.node.cfg.Sources {
			if target.URL == origin+"/v1/blocks/"+b.Hash {
				valid = true
				break
			}
		}
		if !valid {
			continue
		}
		req, e := http.NewRequestWithContext(r.ctx, "GET", target.URL, nil)
		if e != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+target.Token)
		resp, e := r.node.client.Do(req)
		if e != nil {
			continue
		}
		data, e := io.ReadAll(io.LimitReader(resp.Body, b.Size+1))
		resp.Body.Close()
		if e == nil && resp.StatusCode == 200 && validBlock(data, b.Size, b.Hash) {
			return data, nil
		}
	}
	return nil, fmt.Errorf("no readable replica for %s", strings.Repeat("*", 8))
}
