package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

func fileRevision(f *os.File, st os.FileInfo) string {
	var info windows.ByHandleFileInformation
	if windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &info) != nil {
		return ""
	}
	var basic struct {
		CreationTime, LastAccessTime, LastWriteTime, ChangeTime int64
		FileAttributes                                          uint32
		Padding                                                 uint32
	}
	if windows.GetFileInformationByHandleEx(windows.Handle(f.Fd()), windows.FileBasicInfo, (*byte)(unsafe.Pointer(&basic)), uint32(unsafe.Sizeof(basic))) != nil {
		return ""
	}
	return hash([]byte(fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.FileSizeHigh, info.FileSizeLow, basic.LastWriteTime, basic.ChangeTime)))
}
