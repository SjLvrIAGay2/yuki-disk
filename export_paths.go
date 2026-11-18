package main

import (
	"strings"
	"unicode/utf8"
)

func safeRelative(p string) bool {
	if p == "" {
		return true
	}
	if len(p) > 1024 || !utf8.ValidString(p) || strings.ContainsAny(p, "\\\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, c := range part {
			if c < 32 || c == 127 {
				return false
			}
		}
	}
	return true
}
