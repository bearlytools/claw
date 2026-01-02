package pool

import (
	"sync/atomic"
)

// BalancerPicker selects a SubConn from available connections for an RPC.
// Implementations must be safe for concurrent use.
type BalancerPicker interface {
	// Pick selects a SubConn from the provided ready connections.
	// Only SubConns that are ready (connected and healthy) are passed.
	// Returns ErrNoReadySubConns if the slice is empty.
	Pick(subConns []*SubConn) (*SubConn, error)
}

// RoundRobinBalancer distributes RPCs evenly across ready connections.
type RoundRobinBalancer struct {
	counter atomic.Uint64
}

// Pick selects the next SubConn in round-robin order.
func (b *RoundRobinBalancer) Pick(subConns []*SubConn) (*SubConn, error) {
	if len(subConns) == 0 {
		return nil, ErrNoReadySubConns
	}

	idx := b.counter.Add(1) - 1
	return subConns[idx%uint64(len(subConns))], nil
}

// PickFirstBalancer always picks the first ready connection.
// This provides failover behavior - traffic goes to the first backend
// and only moves to others if the first fails.
type PickFirstBalancer struct{}

// Pick selects the first SubConn in the list.
func (b *PickFirstBalancer) Pick(subConns []*SubConn) (*SubConn, error) {
	if len(subConns) == 0 {
		return nil, ErrNoReadySubConns
	}
	return subConns[0], nil
}

// WeightedBalancer distributes RPCs according to address weights.
// Addresses with higher weights receive proportionally more traffic.
type WeightedBalancer struct {
	counter atomic.Uint64
}

// Pick selects a SubConn based on weights using weighted round-robin.
// SubConns with weight 0 are treated as weight 1.
func (b *WeightedBalancer) Pick(subConns []*SubConn) (*SubConn, error) {
	if len(subConns) == 0 {
		return nil, ErrNoReadySubConns
	}

	// Calculate total weight
	var totalWeight uint64
	for _, sc := range subConns {
		w := uint64(sc.addr.Weight)
		if w == 0 {
			w = 1
		}
		totalWeight += w
	}

	// Get position in weighted cycle
	idx := b.counter.Add(1) - 1
	pos := idx % totalWeight

	// Find the SubConn at this position
	var cumulative uint64
	for _, sc := range subConns {
		w := uint64(sc.addr.Weight)
		if w == 0 {
			w = 1
		}
		cumulative += w
		if pos < cumulative {
			return sc, nil
		}
	}

	// Fallback (should not reach here)
	return subConns[0], nil
}

// RandomBalancer picks a random SubConn.
// This provides simple load distribution without maintaining state.
type RandomBalancer struct {
	counter atomic.Uint64
}

// Pick selects a SubConn using a simple pseudo-random selection.
// Uses an LCG-style approach for fast random selection.
func (b *RandomBalancer) Pick(subConns []*SubConn) (*SubConn, error) {
	if len(subConns) == 0 {
		return nil, ErrNoReadySubConns
	}

	if len(subConns) == 1 {
		return subConns[0], nil
	}

	// Use counter as seed with LCG-style randomization
	seed := b.counter.Add(1)
	// LCG parameters (from Numerical Recipes)
	idx := (seed*6364136223846793005 + 1442695040888963407) % uint64(len(subConns))
	return subConns[idx], nil
}
