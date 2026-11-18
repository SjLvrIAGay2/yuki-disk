package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
)

func openComponents(root *os.File, p string, dir bool) (*os.File, error) {
	if !safeRelative(p) {
		return nil, os.ErrNotExist
	}
	parts := strings.Split(p, "/")
	if p == "" {
		parts = []string{"."}
	}
	parent := root
	var owned *os.File
	for i, part := range parts {
		if part != "." && (!filepath.IsLocal(part) || strings.ContainsAny(part, ":?*") || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ")) {
			if owned != nil {
				owned.Close()
			}
			return nil, os.ErrNotExist
		}
		next, err := openRelativeNative(parent, part, windows.GENERIC_READ, i < len(parts)-1 || dir)
		if owned != nil {
			owned.Close()
		}
		if err != nil {
			return nil, err
		}
		owned = next
		parent = next
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
	for _, export := range n.cfg.Exports {
		if export.ID == "" || !safeRelative(export.ID) || strings.Contains(export.ID, "/") || !filepath.IsAbs(export.Root) || n.exports[export.ID] != nil {
			return errors.New("invalid export")
		}
		clean := filepath.Clean(export.Root)
		volume := filepath.VolumeName(clean)
		root, err := os.Open(volume + string(filepath.Separator))
		if err != nil {
			return err
		}
		relative := strings.TrimPrefix(strings.TrimPrefix(clean, volume), string(filepath.Separator))
		f, err := openComponents(root, filepath.ToSlash(relative), true)
		root.Close()
		if err != nil {
			return fmt.Errorf("export %s: %w", export.ID, err)
		}
		n.exports[export.ID] = f
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
