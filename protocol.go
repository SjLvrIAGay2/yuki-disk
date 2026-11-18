package main

type Task struct {
	ID         int64          `json:"id"`
	Kind       string         `json:"kind"`
	Hash       string         `json:"hash"`
	Node       string         `json:"node"`
	URL        string         `json:"url"`
	Generation int64          `json:"generation"`
	Fence      int64          `json:"fence"`
	Owner      string         `json:"owner"`
	Payload    map[string]any `json:"payload"`
}
