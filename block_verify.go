package main

func validBlock(data []byte, size int64, expectedHash string) bool {
	return int64(len(data)) == size && hash(data) == expectedHash
}

const scrubWindow = 8

func scrubSelected(index, cursor, total int) bool {
	return total > 0 && (index-cursor+total)%total < scrubWindow
}

func nextScrub(cursor, total int) int {
	if total == 0 {
		return 0
	}
	return (cursor + scrubWindow) % total
}
