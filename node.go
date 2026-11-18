package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	runtimeState   atomic.Pointer[nodeRuntimeSnapshot]
	exports        map[string]*os.File
	contentSlots   chan struct{}
	blockReadSlots chan struct{}
	copyActive     atomic.Bool
	cfg            Config
	root           *os.Root
	mu             sync.Mutex
	client         *http.Client
	controlClient  *http.Client
	ready          bool
	reconciled     atomic.Pointer[reconciliationState]
	lock           *os.File
	verified       map[string]os.FileInfo
	scrubCursor    int
	network        NetworkStats
	resourceBoot   string
	blocks         blockBackend
}

func openNode(c Config) (*Node, error) {
	controlClient, e := c.newControlClient()
	if e != nil {
		return nil, e
	}
	if e := c.storageDefaults(); e != nil {
		return nil, e
	}
	if e := os.MkdirAll(c.Data, 0700); e != nil {
		return nil, e
	}
	resolved, e := filepath.EvalSymlinks(c.Data)
	if e != nil || filepath.Clean(resolved) != filepath.Clean(c.Data) {
		return nil, errors.New("data root must not contain symlinks")
	}
	root, e := os.OpenRoot(c.Data)
	if e != nil {
		return nil, e
	}
	if e = validateStorageRoot(root); e != nil {
		root.Close()
		return nil, e
	}
	n := &Node{contentSlots: make(chan struct{}, 16), blockReadSlots: make(chan struct{}, 8), cfg: c, root: root, resourceBoot: randomID(), client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect forbidden") }}}
	n.controlClient = controlClient
	for _, dir := range []string{"blocks", "state", "nonces", "tmp"} {
		if e = root.Mkdir(dir, 0700); e != nil && !os.IsExist(e) {
			root.Close()
			return nil, e
		}
		st, e := root.Lstat(dir)
		if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, errors.New("unsafe data subdirectory")
		}
	}
	if st, err := root.Lstat("process.lock"); err == nil && !st.Mode().IsRegular() {
		root.Close()
		return nil, errors.New("unsafe process lock")
	}
	n.lock, e = root.OpenFile("process.lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		root.Close()
		return nil, e
	}
	if e = lockExclusive(n.lock); e != nil {
		n.lock.Close()
		root.Close()
		return nil, errors.New("data root already in use")
	}
	f, err := root.Open("tmp")
	if err != nil {
		n.close()
		return nil, err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		n.close()
		return nil, err
	}
	for _, a := range entries {
		if a.Type().IsRegular() {
			if err = root.Remove("tmp/" + a.Name()); err != nil {
				n.close()
				return nil, err
			}
		} else {
			n.close()
			return nil, errors.New("unsafe temporary entry")
		}
	}
	if err = n.syncDir("tmp"); err != nil {
		n.close()
		return nil, err
	}
	if e = n.checkBackendFormat(); e != nil {
		n.close()
		return nil, e
	}
	if c.StorageMode == "container" {
		n.blocks, e = openContainer(n)
	} else {
		d := &directoryBlocks{node: n}
		n.blocks = d
		_, e = d.Inventory()
		if e == nil && c.CapacityBytes > 0 && d.used.Load() > c.CapacityBytes {
			e = errors.New("existing blocks exceed capacity_bytes")
		}
	}
	if e != nil {
		n.close()
		return nil, e
	}
	if e = n.saveBackendFormat(); e != nil {
		n.close()
		return nil, e
	}
	if e = n.openExports(); e != nil {
		n.close()
		return nil, e
	}
	return n, nil
}

func (n *Node) close() {
	if n.controlClient != nil {
		n.controlClient.CloseIdleConnections()
	}
	for _, f := range n.exports {
		f.Close()
	}
	if n.blocks != nil {
		n.blocks.Close()
	}
	n.lock.Close()
	n.root.Close()
}
