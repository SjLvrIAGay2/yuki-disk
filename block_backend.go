package main

import (
	"errors"
	"os"
	"sync/atomic"
)

type blockBackend interface {
	Read(string, int64) ([]byte, error)
	Write(string, int64, []byte) error
	Delete(string, int64) error
	Inventory() ([]DiskItem, error)
	Usage() (stored, allocated, available int64)
	Close() error
}
type directoryBlocks struct {
	node *Node
	used atomic.Int64
}

func (d *directoryBlocks) Read(hash string, generation int64) ([]byte, error) {
	name := blockName(hash, generation)
	st, e := d.node.root.Lstat(name)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("unsafe block file")
	}
	return d.node.root.ReadFile(name)
}
func (d *directoryBlocks) Write(hash string, generation int64, b []byte) error {
	old := int64(0)
	if st, e := d.node.root.Lstat(blockName(hash, generation)); e == nil {
		if !st.Mode().IsRegular() {
			return errors.New("unsafe block file")
		}
		old = st.Size()
	} else if !os.IsNotExist(e) {
		return e
	}
	if limit := d.node.cfg.CapacityBytes; limit > 0 && int64(len(b))-old > limit-d.used.Load() {
		return errors.New("storage quota exceeded")
	}
	if e := d.node.atomic(blockName(hash, generation), b); e != nil {
		return e
	}
	d.used.Add(int64(len(b)) - old)
	return nil
}
func (d *directoryBlocks) Delete(hash string, generation int64) error {
	name := blockName(hash, generation)
	st, e := d.node.root.Lstat(name)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() {
		return errors.New("unsafe block file")
	}
	if e = d.node.root.Remove(name); e != nil {
		return e
	}
	d.used.Add(-st.Size())
	return d.node.syncDir("blocks")
}
func (d *directoryBlocks) Usage() (int64, int64, int64) {
	used := d.used.Load()
	limit := d.node.cfg.CapacityBytes
	if limit == 0 {
		limit = 9007199254740991
	}
	return used, used, max(0, limit-used)
}
func (d *directoryBlocks) Close() error { return nil }
