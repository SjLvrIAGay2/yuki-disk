package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Server struct {
	p2pHTTPOnce  sync.Once
	p2pHTTP      *http.Client
	p2pHTTPErr   error
	cfg          Config
	db           *pgxpool.Pool
	instanceID   string
	runtimeState atomic.Pointer[runtimeSnapshot]
}

func openStore(ctx context.Context, c Config) (*Server, error) {
	poolConfig, e := pgxpool.ParseConfig(c.Database)
	if e != nil {
		return nil, e
	}
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, e := pgxpool.NewWithConfig(ctx, poolConfig)
	if e != nil {
		return nil, e
	}
	if e = pool.Ping(ctx); e != nil {
		pool.Close()
		return nil, e
	}
	s := &Server{cfg: c, db: pool, instanceID: randomID()}
	if c.P2P != nil {
		s.p2pHTTPOnce.Do(func() { s.p2pHTTP, s.p2pHTTPErr = c.P2P.issuerClient(c.Development) })
		if s.p2pHTTPErr != nil {
			pool.Close()
			return nil, s.p2pHTTPErr
		}
	}
	e = s.tx(ctx, func(t pgx.Tx) error {
		var exists bool
		if e := t.QueryRow(ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists); e != nil {
			return e
		}
		if !exists {
			b, _ := migrations.ReadFile("migrations/001_initial.sql")
			if _, e := t.Exec(ctx, string(b)); e != nil {
				return e
			}
		}
		var version int
		if e := t.QueryRow(ctx, "SELECT max(version) FROM schema_migrations").Scan(&version); e != nil {
			return e
		}
		if version < 1 || version > 17 {
			return errors.New("unsupported schema version")
		}
		if err := upgradeStore(ctx, t, version); err != nil {
			return err
		}

		for _, n := range c.Nodes {
			reserve := n.ReserveBytes
			if _, e := t.Exec(ctx, "INSERT INTO nodes(id,url,tier,read_bandwidth_bps,write_bandwidth_bps,reserve_bytes) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id) DO UPDATE SET url=excluded.url,tier=excluded.tier", n.ID, n.URL, n.Tier, n.ReadBandwidth, n.WriteBandwidth, reserve); e != nil {
				return e
			}

			raw, _ := json.Marshal(nodeRuntimeValues{ReadBandwidth: n.ReadBandwidth, WriteBandwidth: n.WriteBandwidth, ReserveBytes: n.ReserveBytes, ResourceInterval: 3, LogUploadSeconds: 10, LogMinLevel: "info"})
			if _, e := t.Exec(ctx, "INSERT INTO node_configuration(node_id,settings) VALUES($1,$2) ON CONFLICT DO NOTHING", n.ID, raw); e != nil {
				return e
			}
		}
		if e := s.bootstrapIdentity(ctx, t); e != nil {
			return e
		}
		if e := s.initializeRuntime(ctx, t); e != nil {
			return e
		}
		return s.syncMounts(ctx, t)
	})
	if e != nil {
		pool.Close()
		return nil, e
	}
	return s, nil
}
func (s *Server) tx(ctx context.Context, f func(pgx.Tx) error) error {
	if t, ok := ctx.Value(requestTransactionKey{}).(pgx.Tx); ok {
		return f(t)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	t, e := s.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer t.Rollback(context.Background())
	if _, e = t.Exec(ctx, "SET LOCAL statement_timeout='10s'; SET LOCAL lock_timeout='10s'; SET LOCAL idle_in_transaction_session_timeout='15s'"); e != nil {
		return e
	}
	if _, e = t.Exec(ctx, "SELECT pg_advisory_xact_lock(904217)"); e != nil {
		return e
	}
	if e = f(t); e != nil {
		return e
	}
	return t.Commit(ctx)
}

type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string             { return e.Message }
func fail(status int, code, msg string) error { return &apiError{status, code, msg} }
func event(ctx context.Context, t pgx.Tx, kind, hash string, detail any) error {
	_, e := t.Exec(ctx, "INSERT INTO events(kind,hash,detail) VALUES($1,NULLIF($2,''),$3)", kind, hash, detail)
	return e
}
