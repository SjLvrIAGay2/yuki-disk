package main

import (
	"fmt"
	"os"
	"syscall"
)

func fileRevision(_ *os.File, st os.FileInfo) string {
	v := st.Sys().(*syscall.Stat_t)
	return hash([]byte(fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d", v.Dev, v.Ino, st.Size(), v.Mtim.Sec, v.Mtim.Nsec, v.Ctim.Sec, v.Ctim.Nsec)))
}
