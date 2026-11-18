package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Do not recreate deleted parents: require the entire ancestor chain to be active.
func restoreParent(ctx context.Context, t pgx.Tx, source string) (string, error) {
	parent := parentPath(source)
	for parent != "" {
		var usable bool
		if err := t.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM paths WHERE path=$1 AND is_dir AND trashed_at IS NULL)
  AND NOT EXISTS(SELECT 1 FROM paths WHERE starts_with($1,path||'/') AND (NOT is_dir OR trashed_at IS NOT NULL))`, parent).Scan(&usable); err != nil {
			return "", err
		}
		if usable {
			break
		}
		parent = parentPath(parent)
	}
	return parent, nil
}

func numberedPath(base string, number int) string {
	if number == 0 {
		return base
	}
	ext := path.Ext(base)
	return strings.TrimSuffix(base, ext) + fmt.Sprintf(" (%d)", number) + ext
}
