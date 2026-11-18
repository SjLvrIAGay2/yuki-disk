package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Metadata keeps its short deadline. A block uses the same transport, redirect
// policy and cancellation context, with a bounded allowance for real link speed.
func (c *Client) blockHTTP() *http.Client {
	client := *c.http
	client.Timeout = 2 * time.Minute
	return &client
}

func retryableUploadError(err error) bool {
	var response *HTTPError
	if !errors.As(err, &response) {
		return true
	}
	if response.Status == 409 || response.Status == 503 {
		return true
	}
	var body struct {
		Code string `json:"code"`
	}
	return response.Status == 403 && json.Unmarshal([]byte(response.Body), &body) == nil && body.Code == "capability_rejected"
}
