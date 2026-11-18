package main

import (
	"context"
	"errors"
	"fmt"

	"net/url"
	"os"
	"path/filepath"
)

func (c *Client) download(ctx context.Context, id, target string) error {
	abs, e := filepath.Abs(target)
	if e != nil {
		return e
	}
	parent := filepath.Dir(abs)
	resolved, e := filepath.EvalSymlinks(parent)
	if e != nil || resolved != parent {
		return errors.New("target parent must not contain symlinks")
	}
	root, e := os.OpenRoot(parent)
	if e != nil {
		return e
	}
	defer root.Close()
	name := filepath.Base(abs)
	if name == "." || name == string(filepath.Separator) {
		return errors.New("invalid destination")
	}
	if _, e = root.Lstat(name); !os.IsNotExist(e) {
		return errors.New("destination exists or cannot be inspected")
	}
	lockName := "." + name + ".yuki.lock"
	if st, err := root.Lstat(lockName); err == nil && !st.Mode().IsRegular() {
		return errors.New("unsafe destination lock")
	}
	lock, e := root.OpenFile(lockName, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return errors.New("destination already being written")
	}
	defer lock.Close()
	if e = lockExclusive(lock); e != nil {
		return errors.New("destination already being written")
	}
	tmp := "." + name + "." + randomID() + ".part"
	f, e := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	defer root.Remove(tmp)
	var plan struct {
		Source *FileSource `json:"source"`
		Size   int64       `json:"size"`
		Blocks []Plan      `json:"blocks"`
	}
	if e = c.request(ctx, "GET", "/v1/download/"+url.PathEscape(id), nil, "", &plan); e != nil {
		return e
	}
	var total int64
	if plan.Source != nil {
		total, e = c.readFile(ctx, *plan.Source, plan.Size, f)
		if e != nil {
			return e
		}
	}
	for _, p := range plan.Blocks {
		if p.Size <= 0 || p.Size > chunkSize {
			return errors.New("invalid chunk size")
		}
		data, err := c.readBlock(ctx, id, p)
		if err != nil {
			return err
		}
		if _, e = f.Write(data); e != nil {
			return e
		}
		total += int64(len(data))
	}
	if total != plan.Size {
		return errors.New("download size mismatch")
	}
	if e = syncFile(f); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = root.Link(tmp, name); e != nil {
		return fmt.Errorf("atomic no-clobber publish: %w", e)
	}
	return syncDirectory(root)
}
