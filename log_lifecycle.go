package main

import (
	"context"
)

func (c *Client) startCommandLogs() *logBuffer {
	logs := newLogBuffer(func(ctx context.Context, b bufferedBatch) (v bufferedReply, e error) {
		e = c.request(ctx, "POST", "/v1/client-logs", b, "", &v)
		return
	})
	logs.quietStop = true
	return logs
}
