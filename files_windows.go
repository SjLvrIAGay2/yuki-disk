package main

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

func lockExclusive(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
}
func syncFile(f *os.File) error { return f.Sync() }

// Reopen the pinned directory with write access. A read-only directory handle
// cannot FlushFileBuffers on Windows; ignoring that failure loses durability.
func flushDirectory(f *os.File) error {
	handle, err := openRelativeNative(f, "", windows.GENERIC_READ|windows.GENERIC_WRITE, true)
	if err != nil {
		return err
	}
	defer handle.Close()
	return windows.FlushFileBuffers(windows.Handle(handle.Fd()))
}
func (n *Node) syncDir(name string) error {
	f, err := n.root.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return flushDirectory(f)
}

func preallocateContainer(f *os.File, size int64) error {
	if err := windows.SetFileInformationByHandle(windows.Handle(f.Fd()), windows.FileAllocationInfo, (*byte)(unsafe.Pointer(&size)), uint32(unsafe.Sizeof(size))); err != nil {
		return err
	}
	if err := f.Truncate(size); err != nil {
		return err
	}
	return validateContainerFile(f, size)
}

func validateContainerFile(f *os.File, size int64) error {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &info); err != nil {
		return err
	}
	var standard struct {
		AllocationSize, EndOfFile int64
		NumberOfLinks             uint32
		DeletePending, Directory  byte
		Padding                   [2]byte
	}
	if err := windows.GetFileInformationByHandleEx(windows.Handle(f.Fd()), windows.FileStandardInfo, (*byte)(unsafe.Pointer(&standard)), uint32(unsafe.Sizeof(standard))); err != nil {
		return err
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_SPARSE_FILE|windows.FILE_ATTRIBUTE_COMPRESSED) != 0 || standard.Directory != 0 || standard.NumberOfLinks != 1 || standard.EndOfFile != size || standard.AllocationSize < size {
		return errors.New("container must be regular, uncompressed, fully allocated, single-link and match configured size")
	}
	return nil
}

func openContainerFile(root *os.Root, name string) (*os.File, error) {
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return openRelativeNative(dir, name, windows.GENERIC_READ|windows.GENERIC_WRITE, false)
}

func openRelativeNative(parent *os.File, name string, access uint32, directory bool) (*os.File, error) {
	if name == "." {
		name = ""
	}
	value, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, err
	}
	attributes := windows.OBJECT_ATTRIBUTES{RootDirectory: windows.Handle(parent.Fd()), ObjectName: value, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	options := uint32(windows.FILE_SYNCHRONOUS_IO_NONALERT | windows.FILE_OPEN_REPARSE_POINT | windows.FILE_OPEN_FOR_BACKUP_INTENT)
	if directory {
		options |= windows.FILE_DIRECTORY_FILE
	}
	var handle windows.Handle
	var io windows.IO_STATUS_BLOCK
	if err := windows.NtCreateFile(&handle, access|windows.SYNCHRONIZE, &attributes, &io, nil, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN, options, 0, 0); err != nil {
		return nil, err
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		windows.CloseHandle(handle)
		return nil, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(handle)
		return nil, errors.New("reparse points forbidden")
	}
	return os.NewFile(uintptr(handle), name), nil
}
