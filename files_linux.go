package main

import (
	"os"
	"syscall"
)

func preallocateContainer(f *os.File, size int64) error {
	return syscall.Fallocate(int(f.Fd()), 0, 0, size)
}
func syncFile(f *os.File) error { return f.Sync() }
