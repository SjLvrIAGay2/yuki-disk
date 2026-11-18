package main

import (
	"io"
	"strconv"
	"strings"
)

func boundedContentRange(value string) bool {
	if !strings.HasPrefix(value, "bytes=") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(parts) != 2 {
		return false
	}
	first, e := strconv.ParseInt(parts[0], 10, 64)
	last, le := strconv.ParseInt(parts[1], 10, 64)
	return e == nil && le == nil && first >= 0 && last >= first && last-first < chunkSize && last <= 9007199254740991 && value == "bytes="+strconv.FormatInt(first, 10)+"-"+strconv.FormatInt(last, 10)
}

// ServeContent's bounded reader still verifies revision before and after every
// source Read. A larger bounded copy buffer avoids two network-volume stat
// roundtrips for each default 32 KiB slice. No file is cached between requests.
func (w *contentWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.CopyBuffer(struct{ io.Writer }{w}, r, make([]byte, 256<<10))
}
