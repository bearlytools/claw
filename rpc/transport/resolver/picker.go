package resolver

import (
	"errors"
	"sync/atomic"
)

// ErrNoAddresses is returned when Pick is called with an empty address list.
var ErrNoAddresses = errors.New("no addresses available")

// Picker selects an address from a list of resolved addresses.
type Picker interface {
	// Pick selects an address from the given list.
	// Returns ErrNoAddresses if the list is empty.
	Pick(addrs []Address) (Address, error)
}

// RoundRobinPicker distributes requests across addresses in round-robin order.
type RoundRobinPicker struct {
	counter atomic.Uint64
}

// Pick selects the next address in round-robin order.
func (p *RoundRobinPicker) Pick(addrs []Address) (Address, error) {
	if len(addrs) == 0 {
		return Address{}, ErrNoAddresses
	}
	idx := p.counter.Add(1) % uint64(len(addrs))
	return addrs[idx], nil
}

// FirstPicker always returns the first address in the list.
// This is useful when addresses are already priority-sorted.
type FirstPicker struct{}

// Pick returns the first address.
func (p *FirstPicker) Pick(addrs []Address) (Address, error) {
	if len(addrs) == 0 {
		return Address{}, ErrNoAddresses
	}
	return addrs[0], nil
}

// PriorityPicker selects the address with the lowest priority value.
// When multiple addresses have the same priority, it uses round-robin among them.
type PriorityPicker struct {
	counter atomic.Uint64
}

// Pick selects an address with the lowest priority.
func (p *PriorityPicker) Pick(addrs []Address) (Address, error) {
	if len(addrs) == 0 {
		return Address{}, ErrNoAddresses
	}

	// Find lowest priority value
	minPriority := addrs[0].Priority
	for _, addr := range addrs[1:] {
		if addr.Priority < minPriority {
			minPriority = addr.Priority
		}
	}

	// Collect addresses with minimum priority
	candidates := make([]Address, 0, len(addrs))
	for _, addr := range addrs {
		if addr.Priority == minPriority {
			candidates = append(candidates, addr)
		}
	}

	// Round-robin among candidates
	idx := p.counter.Add(1) % uint64(len(candidates))
	return candidates[idx], nil
}
