package main

import (
	"encoding/json"
	"errors"
	"os"
)

type backendFormat struct {
	Mode      string `json:"storage_mode"`
	Container string `json:"container_file"`
}

func (n *Node) checkBackendFormat() error {
	info, e := n.root.Lstat("storage-format.json")
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return errors.New("storage format must be a regular file")
	}
	raw, e := n.root.ReadFile("storage-format.json")
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	var stored backendFormat
	if e = json.Unmarshal(raw, &stored); e != nil {
		return e
	}
	if stored != (backendFormat{n.cfg.StorageMode, n.cfg.ContainerFile}) {
		return errors.New("data directory belongs to another storage mode or container; migrate to a new node instead")
	}
	return nil
}
func (n *Node) saveBackendFormat() error {
	if _, e := n.root.Lstat("storage-format.json"); e == nil {
		return n.checkBackendFormat()
	} else if !os.IsNotExist(e) {
		return e
	}
	raw, _ := json.Marshal(backendFormat{n.cfg.StorageMode, n.cfg.ContainerFile})
	return n.atomic("storage-format.json", raw)
}
