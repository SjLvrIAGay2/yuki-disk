package main

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// The caller already holds s.tx's shared namespace/restore/delete lock.
// Keep the request's original digest domain and store the result atomically
// with its changes so a lost response can replay the exact committed result.
func operationResult(ctx context.Context, t pgx.Tx, key, signature string, result *json.RawMessage, apply func() error) error {
	var prior string
	err := t.QueryRow(ctx, "SELECT request_digest,result FROM operations WHERE key=$1", key).Scan(&prior, result)
	if err == nil {
		if prior != signature {
			return fail(409, "idempotency_conflict", "key used for another operation")
		}
		return nil
	}
	if err != pgx.ErrNoRows {
		return err
	}
	if err = apply(); err != nil {
		return err
	}
	_, err = t.Exec(ctx, "INSERT INTO operations(key,request_digest,result) VALUES($1,$2,$3)", key, signature, *result)
	return err
}
