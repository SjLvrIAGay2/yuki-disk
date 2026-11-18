package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (d *directoryBlocks) Inventory() ([]DiskItem, error) {
	n := d.node
	f, e := n.root.Open("blocks")
	if e != nil {
		return nil, e
	}
	entries, e := f.ReadDir(-1)
	f.Close()
	if e != nil {
		return nil, e
	}
	items := []DiskItem{}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	verified := make(map[string]os.FileInfo, len(entries))
	var used int64
	for index, entry := range entries {
		parts := strings.Split(entry.Name(), ".")
		if len(parts) != 2 || !validHash(parts[0]) || entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil, errors.New("unexpected disk entry")
		}
		g, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil || g < 1 {
			return nil, errors.New("bad disk generation")
		}
		info, e := entry.Info()
		if e != nil {
			return nil, e
		}
		if info.Size() < 0 || info.Size() > 9007199254740991-used {
			return nil, errors.New("invalid disk usage")
		}
		used += info.Size()
		if info.Size() <= 0 || info.Size() > chunkSize {
			slog.Error("corrupt_block", "file", entry.Name())
			continue
		}
		old := n.verified[entry.Name()]
		force := scrubSelected(index, n.scrubCursor, len(entries))
		if old == nil || !os.SameFile(old, info) || old.Size() != info.Size() || !old.ModTime().Equal(info.ModTime()) || force {
			data, err := n.root.ReadFile("blocks/" + entry.Name())
			if err != nil {
				return nil, err
			}
			if !validBlock(data, info.Size(), parts[0]) {
				slog.Error("corrupt_block", "file", entry.Name())
				continue
			}
		}
		verified[entry.Name()] = info
		items = append(items, DiskItem{parts[0], info.Size(), g})
	}
	n.verified = verified
	d.used.Store(used)
	n.scrubCursor = nextScrub(n.scrubCursor, len(entries))
	return items, nil
}

func (n *Node) background(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		n.mu.Lock()
		items, e := n.blocks.Inventory()
		if e == nil {
			e = n.report(ctx, items, true)
		}
		n.ready = e == nil
		n.publishReadiness(n.ready)
		if e != nil {
			slog.Warn("reconcile", "error", e)
		}
		n.cleanNonces()
		n.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
