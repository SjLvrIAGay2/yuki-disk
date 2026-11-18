package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// Heat counts authorized download requests, never inventory or replication traffic.
// A five-minute half-life lets old bursts lose influence without a write per sweep.
func recordAccess(ctx context.Context, t pgx.Tx, hash string) error {
	_, e := t.Exec(ctx, `UPDATE blocks SET last_access=now(),heat=least(1000000,heat*power(0.5,greatest(0,extract(epoch FROM now()-heat_at))/300)+1),heat_at=now() WHERE hash=$1`, hash)
	return e
}
