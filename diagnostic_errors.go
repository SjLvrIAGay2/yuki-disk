package main

import (
	"context"
	"errors"
	"io"
	"net"
)

// Closed vocabulary only: errors may contain URLs, response bodies or paths.
func diagnosticReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	var remote *controlError
	if errors.As(err, &remote) {
		switch remote.Status {
		case 401, 403:
			return "control_auth_rejected"
		case 409:
			return "control_conflict"
		case 429:
			return "control_rate_limited"
		case 503:
			return "control_unavailable"
		default:
			return "control_rejected"
		}
	}
	var network net.Error
	if errors.As(err, &network) && network.Timeout() {
		return "timeout"
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "connection_closed"
	}
	var operation *net.OpError
	if errors.As(err, &operation) {
		return "network_failure"
	}
	return "unclassified"
}
