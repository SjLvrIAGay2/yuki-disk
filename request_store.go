package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type requestTransactionKey struct{}
type databaseQuery interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Never set session-level pool state: the tenant scope lives only in a transaction.
func (s *Server) query(ctx context.Context) databaseQuery {
	if t, ok := ctx.Value(requestTransactionKey{}).(pgx.Tx); ok {
		return t
	}
	return s.db
}
