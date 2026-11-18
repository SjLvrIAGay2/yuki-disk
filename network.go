package main

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Payload counters include client transfers and node-to-node copies, not HTTP headers.
type NetworkStats struct {
	received, sent        atomic.Int64
	readers, writers      atomic.Int64
	mu                    sync.Mutex
	readSpeed, writeSpeed float64
}
type countedReader struct {
	io.Reader
	count *atomic.Int64
}

func (r countedReader) Read(p []byte) (int, error) {
	n, e := r.Reader.Read(p)
	r.count.Add(int64(n))
	return n, e
}
func (n *NetworkStats) observe(read bool, size int64, elapsed time.Duration) {
	if size < 65536 {
		return
	}
	seconds := elapsed.Seconds()
	if seconds < .001 {
		seconds = .001
	}
	speed := float64(size) / seconds
	n.mu.Lock()
	defer n.mu.Unlock()
	target := &n.writeSpeed
	if read {
		target = &n.readSpeed
	}
	if *target == 0 {
		*target = speed
	} else {
		*target = .8*(*target) + .2*speed
	}
}
func (n *NetworkStats) speeds() (float64, float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.readSpeed, n.writeSpeed
}
