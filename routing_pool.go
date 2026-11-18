package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"math"
	"sort"
	"time"
)

type routingNode struct {
	id, url, tier                    string
	resource                         ResourceReport
	fresh                            bool
	readBudget, writeBudget, reserve int64
	readDelay, writeDelay            float64
	available, reserved              int64
	quotaAvailable, volumeAvailable  int64
}

func bandwidth(configured int64, observed float64) float64 {
	if configured > 0 {
		return float64(configured)
	}
	if observed > 0 {
		return math.Max(65536, math.Min(observed, 1e12))
	}
	return float64(chunkSize) // Bootstrap estimate only, never advertised as a measured link speed.
}
func (s *Server) routingPool(ctx context.Context, t pgx.Tx) ([]*routingNode, error) {
	rows, e := t.Query(ctx, `SELECT n.id,n.url,n.tier,n.read_bandwidth_bps,n.write_bandwidth_bps,n.reserve_bytes,
 greatest(0,extract(epoch FROM n.read_until-now()))::float8,greatest(0,extract(epoch FROM n.write_until-now()))::float8,
 coalesce(r.received_at>now()-interval '15 seconds' AND r.sampled_at>now()-interval '20 seconds',false),r.payload
 FROM nodes n LEFT JOIN node_resources r ON r.node_id=n.id WHERE n.last_seen>now()-interval '30 seconds' ORDER BY n.id`)
	if e != nil {
		return nil, e
	}
	configured := map[string]bool{}
	for _, n := range s.cfg.Nodes {
		configured[n.ID] = true
	}
	all := []*routingNode{}
	volumes := map[string]int64{}
	for rows.Next() {
		n := &routingNode{}
		var raw []byte
		if e = rows.Scan(&n.id, &n.url, &n.tier, &n.readBudget, &n.writeBudget, &n.reserve, &n.readDelay, &n.writeDelay, &n.fresh, &raw); e != nil {
			rows.Close()
			return nil, e
		}
		if !configured[n.id] {
			continue
		}
		if len(raw) > 0 {
			if e = json.Unmarshal(raw, &n.resource); e != nil {
				rows.Close()
				return nil, e
			}
		}
		if n.fresh {
			v := n.resource.Volume
			available := n.resource.Storage.VolumeAvailable
			if prior, ok := volumes[v]; !ok || available < prior {
				volumes[v] = available
			}
		}
		all = append(all, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	rows, e = t.Query(ctx, `SELECT p.node_id,nr.payload->>'volume',coalesce(sum(p.size),0)::bigint FROM placement_reservations p
 JOIN node_resources nr ON nr.node_id=p.node_id
 LEFT JOIN replicas rp ON rp.hash=p.hash AND rp.node_id=p.node_id AND rp.generation=p.generation
 WHERE (p.expires_at>now() OR EXISTS(SELECT 1 FROM tasks t WHERE t.hash=p.hash AND t.node_id=p.node_id AND t.generation=p.generation AND t.kind='copy' AND t.state IN ('queued','running')))
 AND (rp.hash IS NULL OR nr.sampled_at<rp.confirmed_at)
 GROUP BY p.node_id,nr.payload->>'volume'`)
	if e != nil {
		return nil, e
	}
	reserved := map[string]int64{}
	nodeReserved := map[string]int64{}
	for rows.Next() {
		var id, v string
		var size int64
		if e = rows.Scan(&id, &v, &size); e != nil {
			rows.Close()
			return nil, e
		}
		reserved[v] += size
		nodeReserved[id] = size
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for _, n := range all {
		n.reserved = nodeReserved[n.id]
		n.quotaAvailable = n.resource.Storage.Available - n.reserved
		n.volumeAvailable = volumes[n.resource.Volume] - reserved[n.resource.Volume]
		n.available = min(n.quotaAvailable, n.volumeAvailable)
	}
	return all, nil
}
func (n *routingNode) freeRatio() float64 {
	if n.resource.Storage.Total <= 0 {
		return 0
	}
	return math.Max(0, float64(n.available)/float64(n.resource.Storage.Total))
}
func (n *routingNode) floor() int64 { return max(n.reserve, n.resource.Storage.Total/20) }
func (n *routingNode) allocation(size int64) int64 {
	unit := max(1, n.resource.Storage.AllocationUnit)
	return (size + unit - 1) / unit * unit
}
func (n *routingNode) writable(size int64) bool {
	return n.fresh && !n.resource.Storage.ReadOnly && n.available >= n.floor()+n.allocation(size)
}
func (n *routingNode) score(size int64, read bool, preferred string) float64 {
	speed := bandwidth(n.writeBudget, n.resource.WriteSpeed)
	delay := n.writeDelay
	traffic := n.resource.ReceiveRate
	active := n.resource.ActiveWrites
	if read {
		speed = bandwidth(n.readBudget, n.resource.ReadSpeed)
		delay = n.readDelay
		traffic = n.resource.SendRate
		active = n.resource.ActiveReads
	}
	cost := (delay + math.Max(.01, float64(size)/speed)) * (1 + math.Min(traffic/speed, 10)) * (1 + float64(active))
	if !n.fresh {
		cost *= 100
	}
	if !read {
		cost *= 1 + 4*math.Pow(1-n.freeRatio(), 2)
		if preferred != "" && n.tier != preferred {
			cost *= 4
		}
	}
	return cost
}
func rankNodes(nodes []*routingNode, size int64, read bool, preferred string) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i].score(size, read, preferred), nodes[j].score(size, read, preferred)
		if a == b {
			return nodes[i].id < nodes[j].id
		}
		return a < b
	})
}
func reserveBandwidth(ctx context.Context, t pgx.Tx, n *routingNode, size int64, read bool) error {
	speed := bandwidth(n.writeBudget, n.resource.WriteSpeed)
	column := "write_until"
	if read {
		speed = bandwidth(n.readBudget, n.resource.ReadSpeed)
		column = "read_until"
	}
	seconds := math.Max(.01, float64(size)/speed)
	_, e := t.Exec(ctx, "UPDATE nodes SET "+column+"=greatest("+column+",now())+make_interval(secs=>$2) WHERE id=$1", n.id, seconds)
	if read {
		n.readDelay += seconds
	} else {
		n.writeDelay += seconds
	}
	return e
}
func reservePlacement(ctx context.Context, t pgx.Tx, pool []*routingNode, n *routingNode, hash string, generation, size int64) error {
	allocation := n.allocation(size)
	var already bool
	if e := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM placement_reservations p WHERE hash=$1 AND node_id=$2 AND generation=$3 AND (expires_at>now() OR EXISTS(SELECT 1 FROM tasks t WHERE t.hash=p.hash AND t.node_id=p.node_id AND t.generation=p.generation AND t.kind='copy' AND t.state IN ('queued','running'))))", hash, n.id, generation).Scan(&already); e != nil {
		return e
	}
	_, e := t.Exec(ctx, `INSERT INTO placement_reservations(hash,node_id,generation,size,expires_at) VALUES($1,$2,$3,$4,now()+interval '120 seconds')
 ON CONFLICT(hash,node_id,generation) DO NOTHING`, hash, n.id, generation, allocation)
	if e != nil {
		return e
	}
	if !already {
		for _, other := range pool {
			if other.resource.Volume == n.resource.Volume {
				other.volumeAvailable -= allocation
				if other.id == n.id {
					other.quotaAvailable -= allocation
					other.reserved += allocation
				}
				other.available = min(other.quotaAvailable, other.volumeAvailable)
			}
		}
	}
	_, e = t.Exec(ctx, "UPDATE placement_reservations SET expires_at=greatest(expires_at,now()+interval '120 seconds') WHERE hash=$1 AND node_id=$2 AND generation=$3", hash, n.id, generation)
	if e != nil {
		return e
	}
	if already {
		return nil
	}
	return reserveBandwidth(ctx, t, n, size, false)
}

func (s *Server) readTargets(ctx context.Context, t pgx.Tx, hash string, size, generation int64) ([]Target, error) {
	pool, e := s.routingPool(ctx, t)
	if e != nil {
		return nil, e
	}
	rows, e := t.Query(ctx, "SELECT node_id FROM replicas WHERE hash=$1 AND generation=$2", hash, generation)
	if e != nil {
		return nil, e
	}
	locations := map[string]bool{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		locations[id] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	candidates := []*routingNode{}
	for _, n := range pool {
		if locations[n.id] {
			candidates = append(candidates, n)
		}
	}
	rankNodes(candidates, size, true, "")
	if len(candidates) == 0 {
		return nil, fail(503, "no_replica", "no live replica")
	}
	if e = reserveBandwidth(ctx, t, candidates[0], size, true); e != nil {
		return nil, e
	}
	targets := []Target{}
	for _, n := range candidates {
		targets = append(targets, Target{s.hasP2P(n.id), n.id, n.url + "/v1/blocks/" + hash, s.cap(n.id, "get", hash, size, generation)})
	}
	if e = recordAccess(ctx, t, hash); e != nil {
		return nil, e
	}
	return targets, nil
}

func heatValue(value float64, at time.Time) float64 {
	return value * math.Pow(.5, math.Max(0, time.Since(at).Seconds())/300)
}
