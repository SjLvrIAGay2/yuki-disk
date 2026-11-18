package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

const containerPage int64 = 65536
const containerHeader int64 = 4096
const descriptorSize = 512

type containerRecord struct {
	slot       int
	sequence   uint64
	bank       int
	hash       string
	generation int64
	size       int64
	pages      []uint32
	active     bool
}

func containerGeometry(size int64) (int, int64, int) {
	entries := min(int(size/containerPage), 1000000)
	start := (containerHeader + int64(entries)*2*descriptorSize + containerPage - 1) / containerPage * containerPage
	return entries, start, int((size - start) / containerPage)
}
func containerHeaderBytes(size int64) []byte {
	entries, start, pages := containerGeometry(size)
	b := make([]byte, containerHeader)
	for _, base := range []int{0, 2048} {
		p := b[base : base+2048]
		copy(p, "YUKIPACK0001")
		binary.LittleEndian.PutUint64(p[16:], uint64(size))
		binary.LittleEndian.PutUint64(p[24:], uint64(entries))
		binary.LittleEndian.PutUint64(p[32:], uint64(start))
		binary.LittleEndian.PutUint64(p[40:], uint64(pages))
		sum := sha256.Sum256(p[:2016])
		copy(p[2016:], sum[:])
	}
	return b
}
func validContainerHeader(b []byte, size int64) bool {
	expected := containerHeaderBytes(size)
	return len(b) == 4096 && (bytes.Equal(b[:2048], expected[:2048]) || bytes.Equal(b[2048:], expected[2048:]))
}
func encodeRecord(r containerRecord) []byte {
	b := make([]byte, descriptorSize)
	copy(b, "YKDREC01")
	binary.LittleEndian.PutUint64(b[8:], r.sequence)
	state := uint32(2)
	if r.active {
		state = 1
	}
	binary.LittleEndian.PutUint32(b[16:], state)
	binary.LittleEndian.PutUint64(b[24:], uint64(r.generation))
	binary.LittleEndian.PutUint64(b[32:], uint64(r.size))
	hash, _ := hex.DecodeString(r.hash)
	copy(b[40:72], hash)
	binary.LittleEndian.PutUint32(b[72:], uint32(len(r.pages)))
	for i, page := range r.pages {
		binary.LittleEndian.PutUint32(b[80+i*4:], page)
	}
	sum := sha256.Sum256(b[:480])
	copy(b[480:], sum[:])
	return b
}
func decodeRecord(b []byte, pageCount int) (containerRecord, bool, error) {
	var r containerRecord
	sum := sha256.Sum256(b[:480])
	if !bytes.Equal(b[:8], []byte("YKDREC01")) || !bytes.Equal(sum[:], b[480:]) {
		return r, false, nil
	}
	r.sequence = binary.LittleEndian.Uint64(b[8:])
	state := binary.LittleEndian.Uint32(b[16:])
	r.active = state == 1
	r.generation = int64(binary.LittleEndian.Uint64(b[24:]))
	r.size = int64(binary.LittleEndian.Uint64(b[32:]))
	r.hash = hex.EncodeToString(b[40:72])
	count := int(binary.LittleEndian.Uint32(b[72:]))
	if r.sequence == 0 || r.generation < 1 || (state != 1 && state != 2) || r.size < 0 || r.size > chunkSize || count > 64 || (r.active && (r.size == 0 || int64(count) != (r.size+containerPage-1)/containerPage)) || (!r.active && (count != 0 || r.size != 0)) {
		return r, false, errors.New("invalid committed container descriptor")
	}
	for i := 0; i < count; i++ {
		page := binary.LittleEndian.Uint32(b[80+i*4:])
		if uint64(page) >= uint64(pageCount) {
			return r, false, errors.New("container page outside configured space")
		}
		r.pages = append(r.pages, page)
	}
	return r, true, nil
}
