package main

import (
	"os"
	"syscall"
)

func syncFile(f *os.File) error {
	_, _, err := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), 51 /* F_FULLFSYNC */, 0)
	if err != 0 {
		return err
	}
	return nil
}
