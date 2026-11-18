package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"unsafe"
)

func filesystemStatus(dir *os.File, cfg Config) (StorageStatus, string, error) {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(dir.Fd()), &info); err != nil {
		return StorageStatus{}, "", err
	}
	var size struct {
		Total, CallerAvailable, ActualAvailable  int64
		SectorsPerAllocationUnit, BytesPerSector uint32
	}
	var io windows.IO_STATUS_BLOCK
	query := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtQueryVolumeInformationFile")
	status, _, _ := query.Call(dir.Fd(), uintptr(unsafe.Pointer(&io)), uintptr(unsafe.Pointer(&size)), unsafe.Sizeof(size), 7 /* FileFsFullSizeInformation */)
	if int32(status) < 0 {
		return StorageStatus{}, "", windows.NTStatus(status)
	}
	unit := uint64(size.SectorsPerAllocationUnit) * uint64(size.BytesPerSector)
	if unit == 0 || size.Total < 0 || size.CallerAvailable < 0 || size.ActualAvailable < 0 || size.CallerAvailable > size.ActualAvailable || size.ActualAvailable > size.Total || float64(size.Total)*float64(unit) > 9007199254740991 {
		return StorageStatus{}, "", errors.New("invalid filesystem capacity")
	}
	total, free, available := uint64(size.Total)*unit, uint64(size.ActualAvailable)*unit, uint64(size.CallerAvailable)*unit
	var device struct{ Type, Characteristics uint32 }
	status, _, _ = query.Call(dir.Fd(), uintptr(unsafe.Pointer(&io)), uintptr(unsafe.Pointer(&device)), unsafe.Sizeof(device), 4 /* FileFsDeviceInformation */)
	if int32(status) < 0 {
		return StorageStatus{}, "", windows.NTStatus(status)
	}
	if device.Characteristics&0x10 != 0 {
		return StorageStatus{}, "", errors.New("network storage roots forbidden")
	}
	var flags uint32
	name := make([]uint16, 64)
	if err := windows.GetVolumeInformationByHandle(windows.Handle(dir.Fd()), nil, 0, nil, nil, &flags, &name[0], uint32(len(name))); err != nil {
		return StorageStatus{}, "", err
	}
	if windows.UTF16ToString(name) != "NTFS" {
		return StorageStatus{}, "", errors.New("native Windows storage requires NTFS")
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return StorageStatus{}, "", err
	}
	defer key.Close()
	machine, _, err := key.GetStringValue("MachineGuid")
	if err != nil || machine == "" {
		return StorageStatus{}, "", errors.New("host identity unavailable")
	}
	value := StorageStatus{Total: int64(total), Free: int64(free), Available: int64(available), ReadOnly: flags&windows.FILE_READ_ONLY_VOLUME != 0, Mode: cfg.StorageMode, Configured: cfg.CapacityBytes, AllocationUnit: int64(unit), VolumeAvailable: int64(available)}
	return value, hash([]byte(fmt.Sprintf("%s:%d", machine, info.VolumeSerialNumber))), nil
}
