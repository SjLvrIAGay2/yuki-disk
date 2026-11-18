package main

const chunkSize = 4194304

type Capability struct {
	Upload     string `json:"upload_id,omitempty"`
	Version    int    `json:"v"`
	Purpose    string `json:"purpose"`
	Node       string `json:"node"`
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Generation int64  `json:"generation"`
	Expires    int64  `json:"exp"`
	ID         string `json:"jti"`
}

type DiskItem struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Generation int64  `json:"generation"`
}

type Task struct {
	ID         int64          `json:"id"`
	Kind       string         `json:"kind"`
	Hash       string         `json:"hash"`
	Node       string         `json:"node"`
	Generation int64          `json:"generation"`
	Fence      int64          `json:"fence"`
	Owner      string         `json:"owner"`
	Payload    map[string]any `json:"payload"`
}
