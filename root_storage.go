package main

import "os"

// Reject unsupported backing filesystems before creating node metadata.
func validateStorageRoot(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	_, _, err = filesystemStatus(dir, Config{})
	return err
}
