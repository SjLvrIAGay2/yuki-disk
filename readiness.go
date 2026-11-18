package main

import "time"

type reconciliationState struct {
	ready     bool
	checkedAt time.Time
}

// Called after reconciliation finishes under the existing storage mutex. Health
// readers must not wait behind file bodies, fsync, or control-plane requests.
func (n *Node) publishReadiness(ready bool) {
	n.reconciled.Store(&reconciliationState{ready: ready, checkedAt: time.Now()})
}

func (n *Node) isReady() bool {
	state := n.reconciled.Load()
	if state == nil || !state.ready {
		return false
	}
	age := time.Since(state.checkedAt)
	return age >= 0 && age < 30*time.Second
}
