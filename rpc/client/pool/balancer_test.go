package pool

import (
	"testing"

	"github.com/bearlytools/claw/rpc/transport/resolver"
)

func TestRoundRobinBalancer(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{
			name:    "Error: empty subconns",
			count:   0,
			wantErr: true,
		},
		{
			name:  "Success: single subconn",
			count: 1,
		},
		{
			name:  "Success: multiple subconns",
			count: 3,
		},
	}

	for _, test := range tests {
		subConns := make([]*SubConn, test.count)
		for i := 0; i < test.count; i++ {
			subConns[i] = &SubConn{
				addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080"},
			}
		}

		b := &RoundRobinBalancer{}
		sc, err := b.Pick(subConns)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestRoundRobinBalancer](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestRoundRobinBalancer](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if sc == nil {
			t.Errorf("[TestRoundRobinBalancer](%s): got nil SubConn", test.name)
		}
	}
}

func TestRoundRobinBalancerDistribution(t *testing.T) {
	subConns := make([]*SubConn, 3)
	for i := 0; i < 3; i++ {
		subConns[i] = &SubConn{
			addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080"},
		}
	}

	b := &RoundRobinBalancer{}
	counts := make(map[string]int)

	numPicks := 300
	for i := 0; i < numPicks; i++ {
		sc, err := b.Pick(subConns)
		if err != nil {
			t.Fatalf("[TestRoundRobinBalancerDistribution]: unexpected error: %v", err)
		}
		counts[sc.addr.Addr]++
	}

	expected := numPicks / len(subConns)
	for addr, count := range counts {
		if count != expected {
			t.Errorf("[TestRoundRobinBalancerDistribution]: address %q picked %d times, want %d", addr, count, expected)
		}
	}
}

func TestPickFirstBalancer(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{
			name:    "Error: empty subconns",
			count:   0,
			wantErr: true,
		},
		{
			name:  "Success: single subconn",
			count: 1,
		},
		{
			name:  "Success: multiple subconns",
			count: 3,
		},
	}

	for _, test := range tests {
		subConns := make([]*SubConn, test.count)
		for i := 0; i < test.count; i++ {
			subConns[i] = &SubConn{
				addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080"},
			}
		}

		b := &PickFirstBalancer{}
		sc, err := b.Pick(subConns)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestPickFirstBalancer](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestPickFirstBalancer](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if len(subConns) > 0 && sc != subConns[0] {
			t.Errorf("[TestPickFirstBalancer](%s): did not pick first SubConn", test.name)
		}
	}
}

func TestPickFirstBalancerAlwaysFirst(t *testing.T) {
	subConns := make([]*SubConn, 3)
	for i := 0; i < 3; i++ {
		subConns[i] = &SubConn{
			addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080"},
		}
	}

	b := &PickFirstBalancer{}

	for i := 0; i < 100; i++ {
		sc, err := b.Pick(subConns)
		if err != nil {
			t.Fatalf("[TestPickFirstBalancerAlwaysFirst]: unexpected error: %v", err)
		}
		if sc != subConns[0] {
			t.Errorf("[TestPickFirstBalancerAlwaysFirst]: pick %d did not return first SubConn", i)
		}
	}
}

func TestWeightedBalancer(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{
			name:    "Error: empty subconns",
			count:   0,
			wantErr: true,
		},
		{
			name:  "Success: single subconn",
			count: 1,
		},
		{
			name:  "Success: multiple subconns",
			count: 3,
		},
	}

	for _, test := range tests {
		subConns := make([]*SubConn, test.count)
		for i := 0; i < test.count; i++ {
			subConns[i] = &SubConn{
				addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080", Weight: uint32(i + 1)},
			}
		}

		b := &WeightedBalancer{}
		sc, err := b.Pick(subConns)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestWeightedBalancer](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestWeightedBalancer](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if sc == nil {
			t.Errorf("[TestWeightedBalancer](%s): got nil SubConn", test.name)
		}
	}
}

func TestWeightedBalancerDistribution(t *testing.T) {
	// Create 3 subconns with weights 1, 2, 3 (total weight = 6)
	subConns := []*SubConn{
		{addr: resolver.Address{Addr: "host0:8080", Weight: 1}},
		{addr: resolver.Address{Addr: "host1:8080", Weight: 2}},
		{addr: resolver.Address{Addr: "host2:8080", Weight: 3}},
	}

	b := &WeightedBalancer{}
	counts := make(map[string]int)

	numPicks := 600 // Multiple of total weight (6)
	for i := 0; i < numPicks; i++ {
		sc, err := b.Pick(subConns)
		if err != nil {
			t.Fatalf("[TestWeightedBalancerDistribution]: unexpected error: %v", err)
		}
		counts[sc.addr.Addr]++
	}

	// Expected picks: host0 = 100, host1 = 200, host2 = 300
	expected := map[string]int{
		"host0:8080": numPicks / 6 * 1,
		"host1:8080": numPicks / 6 * 2,
		"host2:8080": numPicks / 6 * 3,
	}

	for addr, want := range expected {
		got := counts[addr]
		if got != want {
			t.Errorf("[TestWeightedBalancerDistribution]: address %q picked %d times, want %d", addr, got, want)
		}
	}
}

func TestWeightedBalancerZeroWeight(t *testing.T) {
	// Zero weight should be treated as 1
	subConns := []*SubConn{
		{addr: resolver.Address{Addr: "host0:8080", Weight: 0}},
		{addr: resolver.Address{Addr: "host1:8080", Weight: 0}},
	}

	b := &WeightedBalancer{}
	counts := make(map[string]int)

	numPicks := 100
	for i := 0; i < numPicks; i++ {
		sc, err := b.Pick(subConns)
		if err != nil {
			t.Fatalf("[TestWeightedBalancerZeroWeight]: unexpected error: %v", err)
		}
		counts[sc.addr.Addr]++
	}

	// Both should be picked roughly equally
	expected := numPicks / 2
	for addr, count := range counts {
		if count != expected {
			t.Errorf("[TestWeightedBalancerZeroWeight]: address %q picked %d times, want %d", addr, count, expected)
		}
	}
}

func TestRandomBalancer(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{
			name:    "Error: empty subconns",
			count:   0,
			wantErr: true,
		},
		{
			name:  "Success: single subconn",
			count: 1,
		},
		{
			name:  "Success: multiple subconns",
			count: 3,
		},
	}

	for _, test := range tests {
		subConns := make([]*SubConn, test.count)
		for i := 0; i < test.count; i++ {
			subConns[i] = &SubConn{
				addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080"},
			}
		}

		b := &RandomBalancer{}
		sc, err := b.Pick(subConns)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestRandomBalancer](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestRandomBalancer](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if sc == nil {
			t.Errorf("[TestRandomBalancer](%s): got nil SubConn", test.name)
		}
	}
}

func TestRandomBalancerDistribution(t *testing.T) {
	subConns := make([]*SubConn, 3)
	for i := 0; i < 3; i++ {
		subConns[i] = &SubConn{
			addr: resolver.Address{Addr: "host" + string(rune('0'+i)) + ":8080"},
		}
	}

	b := &RandomBalancer{}
	counts := make(map[string]int)

	numPicks := 3000
	for i := 0; i < numPicks; i++ {
		sc, err := b.Pick(subConns)
		if err != nil {
			t.Fatalf("[TestRandomBalancerDistribution]: unexpected error: %v", err)
		}
		counts[sc.addr.Addr]++
	}

	// Random distribution should be roughly equal, allow 20% variance
	expected := numPicks / len(subConns)
	tolerance := expected / 5

	for addr, count := range counts {
		if count < expected-tolerance || count > expected+tolerance {
			t.Errorf("[TestRandomBalancerDistribution]: address %q picked %d times, expected %d±%d", addr, count, expected, tolerance)
		}
	}
}
