package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func preallocateContainer(f *os.File, size int64) error {
	allocation := unix.Fstore_t{Flags: unix.F_ALLOCATEALL, Posmode: unix.F_PEOFPOSMODE, Length: size}
	if err := unix.FcntlFstore(f.Fd(), unix.F_PREALLOCATE, &allocation); err != nil {
		return err
	}
	if allocation.Bytesalloc < size {
		return errors.New("incomplete physical container allocation")
	}
	// Establish logical length only after physical reservation succeeds.
	return f.Truncate(size)
}

func syncFile(f *os.File) error {
	_, err := unix.FcntlInt(f.Fd(), unix.F_FULLFSYNC, 0)
	return err
}
