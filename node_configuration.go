package main

import (
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

type nodeRuntimeValues struct {
	ReadBandwidth    int64  `json:"read_bandwidth_bps"`
	WriteBandwidth   int64  `json:"write_bandwidth_bps"`
	ReserveBytes     int64  `json:"reserve_bytes"`
	ResourceInterval int    `json:"resource_interval_seconds"`
	LogUploadSeconds int    `json:"log_upload_seconds"`
	LogMinLevel      string `json:"log_min_level"`
}

func (v nodeRuntimeValues) valid() bool {
	return v.ReadBandwidth >= 0 && v.ReadBandwidth <= 1000000000000 && v.WriteBandwidth >= 0 && v.WriteBandwidth <= 1000000000000 && v.ReserveBytes >= 0 && v.ReserveBytes <= 9007199254740991 && v.ResourceInterval >= 1 && v.ResourceInterval <= 5 && v.LogUploadSeconds >= 2 && v.LogUploadSeconds <= 60 && validLogLevel(v.LogMinLevel)
}
func (s *Server) nodeConfigurations(w http.ResponseWriter, r *http.Request, b []byte, actor string) error {
	ctx := r.Context()
	if r.Method == "POST" {
		if e := requireConfigurationFields(b, "version", "settings"); e != nil {
			return e
		}
		var object map[string]json.RawMessage
		if e := json.Unmarshal(b, &object); e != nil {
			return fail(400, "invalid_configuration", "invalid configuration")
		}
		if e := requireConfigurationFields(object["settings"], "read_bandwidth_bps", "write_bandwidth_bps", "reserve_bytes", "resource_interval_seconds", "log_upload_seconds", "log_min_level"); e != nil {
			return e
		}
		var p struct {
			Version  int64             `json:"version"`
			Settings nodeRuntimeValues `json:"settings"`
		}
		if e := parse(b, &p); e != nil {
			return e
		}
		id := r.PathValue("id")
		if _, ok := s.nodeConfig(id); !ok {
			return fail(404, "not_found", "node not configured")
		}
		if !p.Settings.valid() {
			return fail(400, "invalid_configuration", "invalid node configuration")
		}
		raw, _ := json.Marshal(p.Settings)
		e := s.tx(ctx, func(t pgx.Tx) error {
			tag, e := t.Exec(ctx, "UPDATE node_configuration SET version=version+1,settings=$2 WHERE node_id=$1 AND version=$3", id, raw, p.Version)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return fail(409, "version_conflict", "node configuration changed")
			}
			if _, e = t.Exec(ctx, "UPDATE nodes SET read_bandwidth_bps=$2,write_bandwidth_bps=$3,reserve_bytes=$4 WHERE id=$1", id, p.Settings.ReadBandwidth, p.Settings.WriteBandwidth, p.Settings.ReserveBytes); e != nil {
				return e
			}
			return audit(ctx, t, actor, "node_configuration.update", id)
		})
		if e != nil {
			return e
		}
	}
	rows, e := s.query(ctx).Query(ctx, "SELECT c.node_id,c.version,c.settings,n.tier,r.payload->'storage'->>'mode',r.payload->'storage'->'configured_bytes' FROM node_configuration c JOIN nodes n ON n.id=c.node_id LEFT JOIN node_resources r ON r.node_id=c.node_id ORDER BY c.node_id")
	if e != nil {
		return e
	}
	items := []map[string]any{}
	for rows.Next() {
		var id, tier string
		var version int64
		var raw json.RawMessage
		var mode *string
		var capacity json.RawMessage
		if e = rows.Scan(&id, &version, &raw, &tier, &mode, &capacity); e != nil {
			rows.Close()
			return e
		}
		if _, ok := s.nodeConfig(id); ok {
			items = append(items, map[string]any{"id": id, "version": version, "settings": raw, "tier": tier, "storage_mode": mode, "capacity_bytes": capacity})
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	statuses, e := s.query(ctx).Query(ctx, "SELECT node_id,instance_id,target_version,applied_version,state,observed_at,max(observed_at) OVER (PARTITION BY node_id) FROM node_configuration_instances ORDER BY node_id,observed_at DESC,instance_id")
	if e != nil {
		return e
	}
	defer statuses.Close()
	instances := []map[string]any{}
	for statuses.Next() {
		var node, id, state string
		var target, applied int64
		var observed, latest time.Time
		if e = statuses.Scan(&node, &id, &target, &applied, &state, &observed, &latest); e != nil {
			return e
		}
		if time.Since(observed) > 20*time.Second {
			state = "stale"
			// One storage root has an exclusive owner. Keep old process reports as
			// history; never let them override the latest node report. Concurrent
			// fresh reports still participate, including failed acknowledgements.
			if observed.Before(latest) {
				state = "superseded"
			}
		}
		instances = append(instances, map[string]any{"node_id": node, "id": id, "target_version": target, "applied_version": applied, "state": state, "observed_at": observed})
	}
	if e = statuses.Err(); e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"items": items, "instances": instances})
	return nil
}
func (s *Server) nodeConfigurationPoll(w http.ResponseWriter, r *http.Request, b []byte, identity string) error {
	if !strings.HasPrefix(identity, "node:") {
		return fail(403, "role_denied", "node only")
	}
	id := strings.TrimPrefix(identity, "node:")
	var p struct {
		Instance string `json:"instance_id"`
		Applied  int64  `json:"applied_version"`
		State    string `json:"state"`
	}
	if e := parse(b, &p); e != nil {
		return e
	}
	if !logStreamPattern.MatchString(p.Instance) || p.Applied < 0 || (p.State != "applied" && p.State != "failed") {
		return fail(400, "invalid_configuration", "invalid node acknowledgement")
	}
	var version int64
	var settings json.RawMessage
	e := s.tx(r.Context(), func(t pgx.Tx) error {
		if e := t.QueryRow(r.Context(), "SELECT version,settings FROM node_configuration WHERE node_id=$1", id).Scan(&version, &settings); e != nil {
			return e
		}
		if p.Applied > version {
			return fail(409, "version_conflict", "future acknowledgement")
		}
		_, e := t.Exec(r.Context(), "INSERT INTO node_configuration_instances(node_id,instance_id,target_version,applied_version,state) VALUES($1,$2,$3,$4,$5) ON CONFLICT(node_id,instance_id) DO UPDATE SET target_version=$3,applied_version=$4,state=$5,observed_at=now()", id, p.Instance, version, p.Applied, p.State)
		return e
	})
	if e != nil {
		return e
	}
	jsonOut(w, 200, map[string]any{"version": version, "settings": settings})
	return nil
}
