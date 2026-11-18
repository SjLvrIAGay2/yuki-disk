package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type logPolicy struct {
	Version       int64  `json:"version"`
	InfoLimit     int    `json:"info_limit"`
	WarningDays   int    `json:"warning_days"`
	UploadSeconds int    `json:"upload_seconds"`
	MinLevel      string `json:"min_level"`
}
type logRecord struct {
	Sequence int64     `json:"sequence"`
	Level    string    `json:"level"`
	Code     string    `json:"code"`
	Occurred time.Time `json:"occurred_at"`
	Count    int       `json:"count"`
}
type logBatch struct {
	PolicyVersion int64       `json:"policy_version"`
	Stream        string      `json:"stream"`
	Entries       []logRecord `json:"entries"`
}

var logStreamPattern = regexp.MustCompile(`^[a-f0-9]{48}$`)

func init() {
	for _, reason := range []string{"timeout", "cancelled", "control_auth_rejected", "control_conflict", "control_rate_limited", "control_unavailable", "control_rejected", "connection_closed", "network_failure"} {
		logCodes["reconcile_"+reason] = true
	}
}

var logCodes = map[string]bool{"started": true, "stopping": true, "request_failed": true, "reconcile_failed": true, "resource_report_failed": true, "storage_corrupt": true, "task_failed": true, "connection_failed": true, "connection_restored": true, "command_completed": true, "command_failed": true, "configuration_applied": true, "configuration_failed": true, "logs_dropped": true, "probe_failed": true, "probe_recovered": true}

func validLogLevel(v string) bool { return v == "info" || v == "warn" || v == "error" }
func readLogPolicy(ctx context.Context, q databaseQuery) (p logPolicy, e error) {
	e = q.QueryRow(ctx, "SELECT version,info_limit,warning_days,upload_seconds,min_level FROM log_policy WHERE singleton").Scan(&p.Version, &p.InfoLimit, &p.WarningDays, &p.UploadSeconds, &p.MinLevel)
	return
}
func trimRuntimeLogs(ctx context.Context, t pgx.Tx, p logPolicy) error {
	if _, e := t.Exec(ctx, `DELETE FROM runtime_logs WHERE (level='info' AND id NOT IN(SELECT id FROM runtime_logs WHERE level='info' ORDER BY id DESC LIMIT $1)) OR (level IN('warn','error') AND received_at<now()-make_interval(days=>$2))`, p.InfoLimit, p.WarningDays); e != nil {
		return e
	}
	_, e := t.Exec(ctx, "DELETE FROM log_streams WHERE last_seen<now()-interval '8 days'")
	if e == nil {
		_, e = t.Exec(ctx, "DELETE FROM log_rate_limits WHERE window_start<now()-interval '8 days'")
	}
	return e
}
func (s *Server) acceptLogs(ctx context.Context, module, source string, b logBatch) (int64, logPolicy, error) {
	var ack int64
	var policy logPolicy
	if b.PolicyVersion < 0 || !logStreamPattern.MatchString(b.Stream) || len(b.Entries) > 100 {
		return 0, policy, fail(400, "invalid_logs", "invalid log batch")
	}
	for i, v := range b.Entries {
		if v.Sequence < 1 || !validLogLevel(v.Level) || !logCodes[v.Code] || v.Count < 1 || v.Count > 1000000 || v.Occurred.Before(time.Now().Add(-7*24*time.Hour)) || v.Occurred.After(time.Now().Add(5*time.Minute)) || (i > 0 && v.Sequence <= b.Entries[i-1].Sequence) {
			return 0, policy, fail(400, "invalid_logs", "invalid log record")
		}
		if module == "cli" && v.Code != "command_completed" && v.Code != "command_failed" && v.Code != "logs_dropped" {
			return 0, policy, fail(400, "invalid_logs", "invalid client event")
		}
	}
	e := s.logTransaction(ctx, func(t pgx.Tx) error {
		var err error
		policy, err = readLogPolicy(ctx, t)
		if err != nil {
			return err
		}
		limit := 6000
		if module == "cli" {
			limit = 120
		}
		var count int
		if err = t.QueryRow(ctx, `INSERT INTO log_rate_limits(source,events) VALUES($1,$2) ON CONFLICT(source) DO UPDATE SET events=CASE WHEN log_rate_limits.window_start<now()-interval '1 minute' THEN $2 ELSE log_rate_limits.events+$2 END,window_start=CASE WHEN log_rate_limits.window_start<now()-interval '1 minute' THEN now() ELSE log_rate_limits.window_start END RETURNING events`, source, max(1, len(b.Entries))).Scan(&count); err != nil {
			return err
		}
		if count > limit {
			return fail(429, "rate_limited", "log rate exceeded")
		}
		if b.PolicyVersion > policy.Version {
			return fail(409, "version_conflict", "future log policy")
		}
		// Identity comes from authenticated transport, never from a submitted module name.
		var streams int
		if err = t.QueryRow(ctx, "SELECT count(*) FROM log_streams WHERE source=$1 AND last_seen>now()-interval '1 hour'", source).Scan(&streams); err != nil {
			return err
		}
		if streams >= 128 {
			var exists bool
			if err = t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM log_streams WHERE source=$1 AND stream=$2)", source, b.Stream).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return fail(429, "rate_limited", "too many log producers")
			}
		}
		if _, err = t.Exec(ctx, "INSERT INTO log_streams(source,stream) VALUES($1,$2) ON CONFLICT DO NOTHING", source, b.Stream); err != nil {
			return err
		}
		if err = t.QueryRow(ctx, "SELECT last_sequence FROM log_streams WHERE source=$1 AND stream=$2 FOR UPDATE", source, b.Stream).Scan(&ack); err != nil {
			return err
		}
		for _, v := range b.Entries {
			if v.Sequence <= ack {
				continue
			}
			if _, err = t.Exec(ctx, "INSERT INTO runtime_logs(module,source,stream,sequence,level,code,occurred_at,count) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", module, source, b.Stream, v.Sequence, v.Level, v.Code, v.Occurred, v.Count); err != nil {
				return err
			}
			ack = v.Sequence
		}
		if _, err = t.Exec(ctx, "UPDATE log_streams SET last_sequence=$3,last_seen=now(),policy_version=$4 WHERE source=$1 AND stream=$2", source, b.Stream, ack, b.PolicyVersion); err != nil {
			return err
		}
		return trimRuntimeLogs(ctx, t, policy)
	})
	return ack, policy, e
}

type logTransactionKey struct{}

// Logging uses its own short serialized transaction, never the file/task lock.
func (s *Server) logTransaction(ctx context.Context, f func(pgx.Tx) error) error {
	if tx, ok := ctx.Value(logTransactionKey{}).(pgx.Tx); ok {
		return f(tx)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='1500ms'; SET LOCAL lock_timeout='500ms'; SELECT pg_advisory_xact_lock(904218)"); e != nil {
		return e
	}
	if e = f(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Server) ingestLogs(w http.ResponseWriter, r *http.Request, body []byte, id string) error {
	if len(body) > 65536 {
		return fail(413, "body_too_large", "log batch exceeds limit")
	}
	module := strings.SplitN(id, ":", 2)[0]
	if module != "node" && module != "maintenance" && module != "web" && module != "status" {
		return fail(403, "role_denied", "log producer rejected")
	}
	var b logBatch
	if e := parse(body, &b); e != nil {
		return e
	}
	ack, p, e := s.acceptLogs(r.Context(), module, id, b)
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"ack": ack, "policy": p})
	return nil
}
func (s *Server) clientLogs(w http.ResponseWriter, r *http.Request, body []byte, owner string) error {
	if len(body) > 65536 {
		return fail(413, "body_too_large", "log batch exceeds limit")
	}
	var b logBatch
	if e := parse(body, &b); e != nil {
		return e
	}
	ack, p, e := s.acceptLogs(r.Context(), "cli", "cli:"+owner, b)
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"ack": ack, "policy": p})
	return nil
}
func (s *Server) adminLogs(w http.ResponseWriter, r *http.Request, _ []byte, _ string) error {
	q := r.URL.Query()
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	level, module := q.Get("level"), q.Get("module")
	if level != "" && !validLogLevel(level) {
		return fail(400, "invalid_query", "invalid level")
	}
	if module != "" && module != "control" && module != "node" && module != "maintenance" && module != "web" && module != "status" && module != "cli" {
		return fail(400, "invalid_query", "invalid module")
	}
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT id,module,CASE WHEN module='cli' THEN 'cli' ELSE source END,level,code,occurred_at,received_at,count FROM runtime_logs WHERE ($1::bigint=0 OR id<$1) AND ($2='' OR level=$2) AND ($3='' OR module=$3) ORDER BY id DESC LIMIT 100", before, level, module)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var m, source, l, code string
		var occurred, received time.Time
		var count int
		if e = rows.Scan(&id, &m, &source, &l, &code, &occurred, &received, &count); e != nil {
			return e
		}
		items = append(items, map[string]any{"id": id, "module": m, "source": source, "level": l, "code": code, "occurred_at": occurred, "received_at": received, "count": count})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items})
	return nil
}
func (s *Server) adminLogPolicy(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	if r.Method == "POST" {
		if e := requireConfigurationFields(b, "version", "info_limit", "warning_days", "upload_seconds", "min_level"); e != nil {
			return e
		}
		var p logPolicy
		if e := parse(b, &p); e != nil {
			return e
		}
		if p.Version < 1 || p.InfoLimit < 0 || p.InfoLimit > 100000 || p.WarningDays < 1 || p.WarningDays > 365 || p.UploadSeconds < 2 || p.UploadSeconds > 60 || !validLogLevel(p.MinLevel) {
			return fail(400, "invalid_configuration", "invalid log policy")
		}
		e := s.logTransaction(r.Context(), func(t pgx.Tx) error {
			tag, e := t.Exec(r.Context(), "UPDATE log_policy SET version=version+1,info_limit=$2,warning_days=$3,upload_seconds=$4,min_level=$5 WHERE singleton AND version=$1", p.Version, p.InfoLimit, p.WarningDays, p.UploadSeconds, p.MinLevel)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return fail(409, "version_conflict", "log policy changed")
			}
			if e = audit(r.Context(), t, actor, "log_policy.update", ""); e != nil {
				return e
			}
			return trimRuntimeLogs(r.Context(), t, p)
		})
		if e != nil {
			return e
		}
	}
	p, e := readLogPolicy(r.Context(), s.query(r.Context()))
	if e != nil {
		return e
	}
	rows, e := s.query(r.Context()).Query(r.Context(), "SELECT source,stream,policy_version,last_seen FROM log_streams WHERE source IN ('control','web','status','maintenance') AND last_seen>now()-interval '1 day' ORDER BY source,stream LIMIT 1000")
	if e != nil {
		return e
	}
	defer rows.Close()
	instances := []map[string]any{}
	for rows.Next() {
		var source, stream string
		var applied int64
		var observed time.Time
		if e = rows.Scan(&source, &stream, &applied, &observed); e != nil {
			return e
		}
		state := "applied"
		if applied != p.Version {
			state = "pending"
		}
		if time.Since(observed) > 150*time.Second {
			state = "stale"
		}
		instances = append(instances, map[string]any{"source": source, "id": stream, "applied_version": applied, "state": state, "observed_at": observed})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"version": p.Version, "info_limit": p.InfoLimit, "warning_days": p.WarningDays, "upload_seconds": p.UploadSeconds, "min_level": p.MinLevel, "instances": instances})
	return nil
}
