package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"sort"
	"time"
)

type routingBlock struct {
	hash             string
	size, generation int64
	access, heatAt   time.Time
	heat             float64
	moved            *time.Time
}

func (s *Server) rebalanceBlock(ctx context.Context, t pgx.Tx, pool []*routingNode, b routingBlock, state rebalanceState) error {
	if state.busy {
		return nil
	}
	have := map[string]bool{}
	for _, id := range state.nodes {
		have[id] = true
	}
	var e error
	locations, candidates := []*routingNode{}, []*routingNode{}
	for _, n := range pool {
		if have[n.id] {
			locations = append(locations, n)
		} else if n.writable(b.size) {
			candidates = append(candidates, n)
		}
	}
	if len(locations) == 0 {
		return nil
	}
	preferred := ""
	if b.access.Add(time.Duration(s.cfg.ColdSeconds) * time.Second).Before(time.Now()) {
		preferred = "cold"
	} else if heatValue(b.heat, b.heatAt) >= 8 {
		preferred = "hot"
	}
	// Keep the desired tier, then the faster/less occupied copies when recovering surplus.
	sort.SliceStable(locations, func(i, j int) bool {
		if preferred != "" && (locations[i].tier == preferred) != (locations[j].tier == preferred) {
			return locations[i].tier == preferred
		}
		return locations[i].score(b.size, true, "") < locations[j].score(b.size, true, "")
	})
	if len(locations) > s.cfg.Replicas {
		return s.enqueue(ctx, t, "delete", b.hash, locations[len(locations)-1].id, b.generation, map[string]any{"reason": "migration_trim"})
	}
	rankNodes(candidates, b.size, false, preferred)
	var target, replaced *routingNode
	reason := "repair"
	if len(locations) < s.cfg.Replicas {
		if len(candidates) > 0 {
			target = candidates[0]
		}
	} else {
		if preferred != "" {
			for _, old := range locations {
				if old.tier != preferred {
					for _, next := range candidates {
						if next.tier == preferred && next.freeRatio() >= .20 {
							target, replaced = next, old
							reason = preferred + "_migration"
							break
						}
					}
					if target != nil {
						break
					}
				}
			}
		}
		if target == nil && (b.moved == nil || time.Since(*b.moved) > 120*time.Second) {
			for _, old := range locations {
				if !old.fresh || old.resource.Storage.Total <= 0 {
					continue
				}
				if float64(old.resource.Storage.Available)/float64(old.resource.Storage.Total) < .10 {
					for _, next := range candidates {
						if next.resource.Volume != old.resource.Volume && next.freeRatio() >= .20 {
							target, replaced = next, old
							reason = "capacity_rebalance"
							break
						}
					}
				}
				if target != nil {
					break
				}
			}
			if target == nil && preferred == "hot" {
				old := locations[len(locations)-1]
				for _, next := range candidates {
					if next.tier == "hot" && next.freeRatio() >= .20 && bandwidth(next.readBudget, next.resource.ReadSpeed) >= 2*bandwidth(old.readBudget, old.resource.ReadSpeed) && next.score(b.size, true, "")*1.5 < old.score(b.size, true, "") {
						target, replaced = next, old
						reason = "bandwidth_rebalance"
						break
					}
				}
			}
		}
	}
	if target == nil {
		return nil
	}
	sources := append([]*routingNode(nil), locations...)
	rankNodes(sources, b.size, true, "")
	source := sources[0]
	payload := map[string]any{"source_node": source.id, "source_url": source.url, "size": b.size, "reason": reason}
	if replaced != nil {
		payload["replace_node"] = replaced.id
	}
	if e = reservePlacement(ctx, t, pool, target, b.hash, b.generation, b.size); e != nil {
		return e
	}
	if e = reserveBandwidth(ctx, t, source, b.size, true); e != nil {
		return e
	}
	if e = s.enqueue(ctx, t, "copy", b.hash, target.id, b.generation, payload); e != nil {
		return e
	}
	if replaced != nil {
		if _, e = t.Exec(ctx, "UPDATE blocks SET last_move_at=now() WHERE hash=$1", b.hash); e != nil {
			return e
		}
	}
	return event(ctx, t, "placement_scheduled", b.hash, map[string]any{"reason": reason, "target": target.id, "source": source.id})
}
