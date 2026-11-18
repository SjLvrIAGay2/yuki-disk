package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func (n *Node) atomic(path string, b []byte) error {
	tmp := "tmp/" + randomID()
	f, e := n.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer n.root.Remove(tmp)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = syncFile(f); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = n.root.Rename(tmp, path); e != nil {
		return e
	}
	return n.syncDir(filepath.Dir(path))
}

func (n *Node) tombstone(h string) (int64, error) {
	b, e := n.root.ReadFile("state/" + h)
	if os.IsNotExist(e) {
		return 0, nil
	}
	if e != nil {
		return 0, e
	}
	return strconv.ParseInt(string(b), 10, 64)
}

func blockName(h string, g int64) string { return "blocks/" + h + "." + strconv.FormatInt(g, 10) }

func (n *Node) put(ctx context.Context, p Capability, reader io.Reader) error {
	tomb, e := n.tombstone(p.Hash)
	if e != nil {
		return e
	}
	if p.Generation <= tomb {
		return errors.New("generation fenced")
	}
	started := time.Now()
	n.network.writers.Add(1)
	defer n.network.writers.Add(-1)
	data, e := io.ReadAll(io.LimitReader(countedReader{reader, &n.network.received}, p.Size+1))
	if e != nil {
		return e
	}
	if !validBlock(data, p.Size, p.Hash) {
		return errors.New("size/hash mismatch")
	}
	n.network.observe(false, int64(len(data)), time.Since(started))
	return n.putStored(p, data)
}

// Caller owns n.mu and supplies fully verified bytes. Recheck the durable fence
// even when bytes were received before taking the physical-operation lock.
func (n *Node) putStored(p Capability, data []byte) error {
	tomb, e := n.tombstone(p.Hash)
	if e != nil {
		return e
	}
	if p.Generation <= tomb {
		return errors.New("generation fenced")
	}
	if !validBlock(data, p.Size, p.Hash) {
		return errors.New("size/hash mismatch")
	}
	if prior, err := n.blocks.Read(p.Hash, p.Generation); err == nil && hash(prior) == p.Hash {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if e = n.blocks.Write(p.Hash, p.Generation, data); e != nil {
		return e
	}
	if n.cfg.CrashAfterRename {
		os.Exit(86)
	}
	return nil
}
