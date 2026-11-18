//go:build linux || darwin

package main

import (
	"errors"
	"os"
	"syscall"
)

func lockExclusive(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func openContainerFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDWR|syscall.O_NOFOLLOW, 0)
}

func validateContainerFile(f *os.File, size int64) error {
	var st syscall.Stat_t
	if err := syscall.Fstat(int(f.Fd()), &st); err != nil {
		return err
	}
	if st.Nlink != 1 || st.Size != size || st.Blocks*512 < size {
		return errors.New("container must have one link, matching size and physical allocation")
	}
	return nil
}

func (n *Node) syncDir(dir string) error {
	f, err := n.root.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return syncFile(f)
}
