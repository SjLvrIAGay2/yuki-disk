package main

import (
	"net/http"
	"strconv"
)

func pagination(r *http.Request) (int, int64, error) {
	limit := 100
	var after int64
	var e error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, e = strconv.Atoi(raw)
		if e != nil || limit < 1 || limit > 200 {
			return 0, 0, fail(400, "invalid_page", "limit must be 1..200")
		}
	}
	if raw := r.URL.Query().Get("after"); raw != "" {
		after, e = strconv.ParseInt(raw, 10, 64)
		if e != nil || after < 0 {
			return 0, 0, fail(400, "invalid_page", "invalid cursor")
		}
	}
	return limit, after, nil
}
