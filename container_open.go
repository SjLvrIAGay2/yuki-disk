package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
)

func openContainer(n *Node) (blockBackend, error) {
	name := n.cfg.ContainerFile
	if st, e := n.root.Lstat(name); os.IsNotExist(e) {
		// Switching an existing directory node must be an explicit data migration.
		dir, e := n.root.Open("blocks")
		if e != nil {
			return nil, e
		}
		items, e := dir.ReadDir(1)
		dir.Close()
		if e != nil && e != io.EOF {
			return nil, e
		}
		if len(items) > 0 {
			return nil, errors.New("cannot switch nonempty filesystem storage to container")
		}
		tmp := "tmp/container-" + randomID()
		f, e := n.root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if e != nil {
			return nil, e
		}
		e = preallocateContainer(f, n.cfg.CapacityBytes)
		if e == nil {
			_, e = f.WriteAt(containerHeaderBytes(n.cfg.CapacityBytes), 0)
		}
		if e == nil {
			e = syncFile(f)
		}
		closeError := f.Close()
		if e == nil {
			e = closeError
		}
		if e == nil {
			e = n.root.Rename(tmp, name)
		}
		if e != nil {
			n.root.Remove(tmp)
			return nil, e
		}
		if e = n.syncDir("."); e != nil {
			return nil, e
		}
	} else if e != nil {
		return nil, e
	} else if !st.Mode().IsRegular() {
		return nil, errors.New("container must be a regular file")
	}
	file, e := openContainerFile(n.root, name)
	if e != nil {
		return nil, e
	}
	failed := true
	defer func() {
		if failed {
			file.Close()
		}
	}()
	if e = validateContainerFile(file, n.cfg.CapacityBytes); e != nil {
		return nil, e
	}
	if e = lockExclusive(file); e != nil {
		return nil, errors.New("container already in use")
	}
	header := make([]byte, containerHeader)
	if _, e = file.ReadAt(header, 0); e != nil {
		return nil, e
	}
	if !validContainerHeader(header, n.cfg.CapacityBytes) {
		return nil, errors.New("invalid container format or capacity")
	}
	if expected := containerHeaderBytes(n.cfg.CapacityBytes); !bytes.Equal(header, expected) {
		if _, e = file.WriteAt(expected, 0); e != nil {
			return nil, e
		}
		if e = syncFile(file); e != nil {
			return nil, e
		}
	}
	entries, start, pages := containerGeometry(n.cfg.CapacityBytes)
	c := &containerBlocks{node: n, file: file, dataStart: start, pageCount: pages, entryCount: entries, records: map[string]containerRecord{}, sequences: make([]uint64, entries), banks: make([]int, entries), slots: make([]bool, entries), pages: make([]bool, pages), verified: map[string]uint64{}}
	reader := bufio.NewReaderSize(io.NewSectionReader(file, containerHeader, int64(entries)*1024), 1<<20)
	pair := make([]byte, 1024)
	for slot := 0; slot < entries; slot++ {
		if _, e = io.ReadFull(reader, pair); e != nil {
			return nil, e
		}
		a, va, er := decodeRecord(pair[:512], pages)
		if er != nil {
			return nil, er
		}
		b, vb, er := decodeRecord(pair[512:], pages)
		if er != nil {
			return nil, er
		}
		if !va && !vb {
			if !bytes.Equal(pair, make([]byte, 1024)) {
				slog.Warn("uncommitted_container_descriptor", "slot", slot)
			}
			continue
		}
		r := a
		r.bank = 0
		if vb && (!va || b.sequence > a.sequence) {
			r = b
			r.bank = 1
		}
		r.slot = slot
		c.sequences[slot], c.banks[slot] = r.sequence, r.bank
		if !r.active {
			continue
		}
		key := blockName(r.hash, r.generation)
		if _, exists := c.records[key]; exists {
			return nil, errors.New("duplicate container block descriptor")
		}
		for _, page := range r.pages {
			if c.pages[page] {
				return nil, errors.New("overlapping committed container allocations")
			}
			c.pages[page] = true
		}
		c.slots[slot] = true
		c.records[key] = r
		c.stored.Add(r.size)
		c.allocated.Add(int64(len(r.pages)) * containerPage)
		c.count.Add(1)
	}
	failed = false
	return c, nil
}
