//go:build linux || darwin

package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
)

// Open every component relative to a pinned directory descriptor, never following links.
func openComponents(root *os.File, p string, dir bool) (*os.File, error) {
	parts := strings.Split(p, "/")
	if p == "" {
		parts = []string{"."}
	}
	fd := int(root.Fd())
	var owned *os.File
	for i, part := range parts {
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
		if i < len(parts)-1 || dir {
			flags |= unix.O_DIRECTORY
		}
		next, err := unix.Openat(fd, part, flags, 0)
		if owned != nil {
			owned.Close()
		}
		if err != nil {
			return nil, err
		}
		owned = os.NewFile(uintptr(next), part)
		fd = next
	}
	st, err := owned.Stat()
	if err != nil || (!st.Mode().IsRegular() && !st.IsDir()) {
		owned.Close()
		return nil, errors.New("unsupported export entry")
	}
	return owned, nil
}
func (n *Node) openExports() error {
	n.exports = map[string]*os.File{}
	for _, e := range n.cfg.Exports {
		if e.ID == "" || !safeRelative(e.ID) || strings.Contains(e.ID, "/") || !filepath.IsAbs(e.Root) || n.exports[e.ID] != nil {
			return errors.New("invalid export")
		}
		root, err := os.Open("/")
		if err != nil {
			return err
		}
		f, err := openComponents(root, strings.TrimPrefix(filepath.Clean(e.Root), "/"), true)
		root.Close()
		if err != nil {
			return fmt.Errorf("export %s: %w", e.ID, err)
		}
		n.exports[e.ID] = f
	}
	return nil
}
func (n *Node) openExport(id, p string) (*os.File, error) {
	root := n.exports[id]
	if root == nil || !safeRelative(p) {
		return nil, os.ErrNotExist
	}
	return openComponents(root, p, false)
}
