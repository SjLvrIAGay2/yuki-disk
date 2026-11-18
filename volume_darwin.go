package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

func filesystemStatus(dir *os.File, cfg Config) (StorageStatus, string, error) {
	var fs unix.Statfs_t
	var st unix.Stat_t
	if err := unix.Fstatfs(int(dir.Fd()), &fs); err != nil {
		return StorageStatus{}, "", err
	}
	if fs.Flags&unix.MNT_LOCAL == 0 {
		return StorageStatus{}, "", errors.New("network storage roots forbidden")
	}
	if err := unix.Fstat(int(dir.Fd()), &st); err != nil {
		return StorageStatus{}, "", err
	}
	total, free, available := float64(fs.Blocks)*float64(fs.Bsize), float64(fs.Bfree)*float64(fs.Bsize), float64(fs.Bavail)*float64(fs.Bsize)
	if fs.Bsize == 0 || total > 9007199254740991 || available > free || free > total {
		return StorageStatus{}, "", errors.New("invalid filesystem capacity")
	}
	host, err := unix.Sysctl("kern.uuid")
	if err != nil || host == "" {
		return StorageStatus{}, "", errors.New("host identity unavailable")
	}
	value := StorageStatus{Total: int64(total), Free: int64(free), Available: int64(available), ReadOnly: fs.Flags&unix.MNT_RDONLY != 0, Mode: cfg.StorageMode, Configured: cfg.CapacityBytes, AllocationUnit: int64(fs.Bsize), VolumeAvailable: int64(available)}
	return value, hash([]byte(fmt.Sprintf("%s:%d", host, st.Dev))), nil
}
