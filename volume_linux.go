package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func filesystemStatus(dir *os.File, cfg Config) (StorageStatus, string, error) {
	var result StorageStatus
	var e error
	var stat syscall.Statfs_t
	var inode syscall.Stat_t
	if e = syscall.Fstatfs(int(dir.Fd()), &stat); e != nil {
		return result, "", e
	}
	if e = syscall.Fstat(int(dir.Fd()), &inode); e != nil {
		return result, "", e
	}
	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bfree) * float64(stat.Bsize)
	available := float64(stat.Bavail) * float64(stat.Bsize)
	if stat.Bsize <= 0 || total > 9007199254740991 || available > free || free > total {
		return result, "", errors.New("invalid filesystem capacity")
	}
	host, e := os.ReadFile("/etc/machine-id")
	if e != nil || len(host) == 0 {
		host, e = os.ReadFile("/proc/sys/kernel/random/boot_id")
	}
	if e != nil {
		return result, "", e
	}
	result = StorageStatus{Total: int64(total), Free: int64(free), Available: int64(available), ReadOnly: stat.Flags&1 != 0, Mode: cfg.StorageMode, Configured: cfg.CapacityBytes, AllocationUnit: int64(stat.Bsize), VolumeAvailable: int64(available)}
	key := hash([]byte(fmt.Sprintf("%s:%d", host, inode.Dev)))
	return result, key, nil
}
