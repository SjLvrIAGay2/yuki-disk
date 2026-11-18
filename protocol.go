package main

const chunkSize = 4194304

type Chunk struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type Target struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type Plan struct {
	Hash    string   `json:"hash"`
	Size    int64    `json:"size"`
	Skip    bool     `json:"skip"`
	Targets []Target `json:"targets"`
}
