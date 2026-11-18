package main

import (
	"context"
	"errors"
	"strings"
)

func (c *Client) versionCommand(ctx context.Context, args []string, key string, out *any) error {
	if args[0] == "version-recycle" {
		if len(args) != 1 {
			return errors.New("version-recycle")
		}
		return c.request(ctx, "GET", "/v1/versions?trash=true", nil, "", out)
	}
	op := strings.TrimPrefix(args[0], "version-")
	items := []map[string]string{}
	if op == "trash" {
		if len(args) < 2 {
			return errors.New("version-trash ID...")
		}
		for _, id := range args[1:] {
			items = append(items, map[string]string{"id": id})
		}
	} else {
		if len(args) != 3 {
			return errors.New("version-restore/version-purge ID TRASH_ID")
		}
		items = append(items, map[string]string{"id": args[1], "trash_id": args[2]})
	}
	key = idempotencyKey(key)
	return c.request(ctx, "POST", "/v1/versions/actions", map[string]any{"operation": op, "items": items}, key, out)
}
