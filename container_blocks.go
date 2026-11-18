package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"sync/atomic"
)

type containerBlocks struct {
	node                     *Node
	file                     *os.File
	dataStart                int64
	pageCount                int
	entryCount               int
	usageMu                  sync.RWMutex
	records                  map[string]containerRecord
	sequences                []uint64
	banks                    []int
	slots, pages             []bool
	verified                 map[string]uint64
	cursor                   int
	stored, allocated, count atomic.Int64
	poisoned                 atomic.Bool
	afterDataSync            func()
}

func (c *containerBlocks) Read(hash string, generation int64) ([]byte, error) {
	r, ok := c.records[blockName(hash, generation)]
	if !ok {
		return nil, os.ErrNotExist
	}
	data := make([]byte, r.size)
	offset := 0
	for _, page := range r.pages {
		length := min(len(data)-offset, int(containerPage))
		if _, e := c.file.ReadAt(data[offset:offset+length], c.dataStart+int64(page)*containerPage); e != nil {
			return nil, e
		}
		offset += length
	}
	return data, nil
}
func (c *containerBlocks) commit(r containerRecord) error {
	if c.sequences[r.slot] == math.MaxUint64 {
		return errors.New("container descriptor sequence exhausted")
	}
	r.sequence = c.sequences[r.slot] + 1
	r.bank = 1 - c.banks[r.slot]
	if _, e := c.file.WriteAt(encodeRecord(r), containerHeader+int64(r.slot)*1024+int64(r.bank)*512); e != nil {
		c.poisoned.Store(true)
		return e
	}
	if e := syncFile(c.file); e != nil {
		c.poisoned.Store(true)
		return e
	}
	c.usageMu.Lock()
	defer c.usageMu.Unlock()
	key := blockName(r.hash, r.generation)
	if old, ok := c.records[key]; ok {
		for _, page := range old.pages {
			c.pages[page] = false
		}
		c.stored.Add(-old.size)
		c.allocated.Add(-int64(len(old.pages)) * containerPage)
		c.count.Add(-1)
		delete(c.records, key)
	}
	c.sequences[r.slot], c.banks[r.slot], c.slots[r.slot] = r.sequence, r.bank, r.active
	if r.active {
		for _, page := range r.pages {
			c.pages[page] = true
		}
		c.records[key] = r
		c.stored.Add(r.size)
		c.allocated.Add(int64(len(r.pages)) * containerPage)
		c.count.Add(1)
	}
	delete(c.verified, key)
	return nil
}
func (c *containerBlocks) Write(hash string, generation int64, data []byte) error {
	if c.poisoned.Load() {
		return errors.New("container awaiting recovery")
	}
	key := blockName(hash, generation)
	slot := -1
	if old, ok := c.records[key]; ok {
		slot = old.slot
	} else {
		for i, used := range c.slots {
			if !used {
				slot = i
				break
			}
		}
	}
	if slot < 0 {
		return errors.New("container index capacity exhausted")
	}
	pages := []uint32{}
	needed := (len(data) + int(containerPage) - 1) / int(containerPage)
	for i, used := range c.pages {
		if !used {
			pages = append(pages, uint32(i))
			if len(pages) == needed {
				break
			}
		}
	}
	if len(pages) < needed {
		return errors.New("container capacity exhausted")
	}
	offset := 0
	for _, page := range pages {
		length := min(len(data)-offset, int(containerPage))
		if _, e := c.file.WriteAt(data[offset:offset+length], c.dataStart+int64(page)*containerPage); e != nil {
			return e
		}
		offset += length
	}
	if e := syncFile(c.file); e != nil {
		return e
	}
	if c.afterDataSync != nil {
		c.afterDataSync()
	}
	return c.commit(containerRecord{slot: slot, hash: hash, generation: generation, size: int64(len(data)), pages: pages, active: true})
}
func (c *containerBlocks) Delete(hash string, generation int64) error {
	if c.poisoned.Load() {
		return errors.New("container awaiting recovery")
	}
	old, ok := c.records[blockName(hash, generation)]
	if !ok {
		return nil
	}
	return c.commit(containerRecord{slot: old.slot, hash: hash, generation: generation, active: false})
}
func (c *containerBlocks) Inventory() ([]DiskItem, error) {
	if c.poisoned.Load() {
		if e := c.recover(); e != nil {
			return nil, e
		}
	}
	keys := make([]string, 0, len(c.records))
	for key := range c.records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := []DiskItem{}
	for i, key := range keys {
		r := c.records[key]
		force := scrubSelected(i, c.cursor, len(keys))
		if c.verified[key] != r.sequence || force {
			data, e := c.Read(r.hash, r.generation)
			if e != nil {
				return nil, e
			}
			if !validBlock(data, r.size, r.hash) {
				delete(c.verified, key)
				continue
			}
			c.verified[key] = r.sequence
		}
		items = append(items, DiskItem{r.hash, r.size, r.generation})
	}
	c.cursor = nextScrub(c.cursor, len(keys))
	return items, nil
}
func (c *containerBlocks) Usage() (int64, int64, int64) {
	c.usageMu.RLock()
	defer c.usageMu.RUnlock()
	allocated := c.allocated.Load()
	available := min(int64(c.pageCount)*containerPage-allocated, (int64(c.entryCount)-c.count.Load())*chunkSize)
	if c.poisoned.Load() {
		available = 0
	}
	return c.stored.Load(), allocated, available
}
func (c *containerBlocks) recover() error {
	c.file.Close()
	opened, e := openContainer(c.node)
	if e != nil {
		return e
	}
	next := opened.(*containerBlocks)
	c.file = next.file
	c.records = next.records
	c.sequences = next.sequences
	c.banks = next.banks
	c.slots = next.slots
	c.pages = next.pages
	c.verified = next.verified
	c.cursor = 0
	c.usageMu.Lock()
	defer c.usageMu.Unlock()
	c.stored.Store(next.stored.Load())
	c.allocated.Store(next.allocated.Load())
	c.count.Store(next.count.Load())
	c.poisoned.Store(false)
	return nil
}
func (c *containerBlocks) Close() error { return c.file.Close() }
func (c *containerBlocks) allocationKey(volume string) string {
	return hash([]byte(fmt.Sprintf("%s:%s:%s", volume, c.node.cfg.Data, c.node.cfg.ContainerFile)))
}
