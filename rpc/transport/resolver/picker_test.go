package resolver

import (
	"testing"
)

func TestRoundRobinPicker(t *testing.T) {
	tests := []struct {
		name    string
		addrs   []Address
		picks   int
		wantErr bool
	}{
		{
			name:    "Error: empty addresses",
			addrs:   []Address{},
			picks:   1,
			wantErr: true,
		},
		{
			name:    "Error: nil addresses",
			addrs:   nil,
			picks:   1,
			wantErr: true,
		},
		{
			name: "Success: single address",
			addrs: []Address{
				{Addr: "localhost:8080"},
			},
			picks: 3,
		},
		{
			name: "Success: multiple addresses",
			addrs: []Address{
				{Addr: "host1:8080"},
				{Addr: "host2:8080"},
				{Addr: "host3:8080"},
			},
			picks: 6,
		},
	}

	for _, test := range tests {
		picker := &RoundRobinPicker{}

		for i := 0; i < test.picks; i++ {
			got, err := picker.Pick(test.addrs)
			switch {
			case err == nil && test.wantErr:
				t.Errorf("[TestRoundRobinPicker](%s): got err == nil, want err != nil", test.name)
				break
			case err != nil && !test.wantErr:
				t.Errorf("[TestRoundRobinPicker](%s): got err == %s, want err == nil", test.name, err)
				break
			case err != nil:
				break
			}

			if !test.wantErr {
				// Verify address is from the list
				found := false
				for _, addr := range test.addrs {
					if got.Addr == addr.Addr {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("[TestRoundRobinPicker](%s): picked address %q not in list", test.name, got.Addr)
				}
			}
		}
	}
}

func TestRoundRobinPickerDistribution(t *testing.T) {
	addrs := []Address{
		{Addr: "host1:8080"},
		{Addr: "host2:8080"},
		{Addr: "host3:8080"},
	}

	picker := &RoundRobinPicker{}
	counts := make(map[string]int)

	// Pick many times to verify distribution
	numPicks := 300
	for i := 0; i < numPicks; i++ {
		got, err := picker.Pick(addrs)
		if err != nil {
			t.Fatalf("[TestRoundRobinPickerDistribution]: unexpected error: %v", err)
		}
		counts[got.Addr]++
	}

	// Each address should be picked exactly numPicks/len(addrs) times
	expected := numPicks / len(addrs)
	for addr, count := range counts {
		if count != expected {
			t.Errorf("[TestRoundRobinPickerDistribution]: address %q picked %d times, want %d", addr, count, expected)
		}
	}
}

func TestFirstPicker(t *testing.T) {
	tests := []struct {
		name    string
		addrs   []Address
		want    Address
		wantErr bool
	}{
		{
			name:    "Error: empty addresses",
			addrs:   []Address{},
			wantErr: true,
		},
		{
			name:    "Error: nil addresses",
			addrs:   nil,
			wantErr: true,
		},
		{
			name: "Success: single address",
			addrs: []Address{
				{Addr: "localhost:8080"},
			},
			want: Address{Addr: "localhost:8080"},
		},
		{
			name: "Success: multiple addresses",
			addrs: []Address{
				{Addr: "host1:8080"},
				{Addr: "host2:8080"},
				{Addr: "host3:8080"},
			},
			want: Address{Addr: "host1:8080"},
		},
	}

	for _, test := range tests {
		picker := &FirstPicker{}
		got, err := picker.Pick(test.addrs)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestFirstPicker](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestFirstPicker](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if got.Addr != test.want.Addr {
			t.Errorf("[TestFirstPicker](%s): got %q, want %q", test.name, got.Addr, test.want.Addr)
		}
	}
}

func TestPriorityPicker(t *testing.T) {
	tests := []struct {
		name    string
		addrs   []Address
		wantErr bool
	}{
		{
			name:    "Error: empty addresses",
			addrs:   []Address{},
			wantErr: true,
		},
		{
			name: "Success: picks lowest priority",
			addrs: []Address{
				{Addr: "host1:8080", Priority: 10},
				{Addr: "host2:8080", Priority: 1},
				{Addr: "host3:8080", Priority: 5},
			},
		},
	}

	for _, test := range tests {
		picker := &PriorityPicker{}
		got, err := picker.Pick(test.addrs)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestPriorityPicker](%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestPriorityPicker](%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Verify picked address has minimum priority
		if len(test.addrs) > 0 {
			minPriority := test.addrs[0].Priority
			for _, addr := range test.addrs {
				if addr.Priority < minPriority {
					minPriority = addr.Priority
				}
			}
			if got.Priority != minPriority {
				t.Errorf("[TestPriorityPicker](%s): picked priority %d, want %d", test.name, got.Priority, minPriority)
			}
		}
	}
}

func TestPriorityPickerRoundRobinWithinPriority(t *testing.T) {
	addrs := []Address{
		{Addr: "host1:8080", Priority: 1},
		{Addr: "host2:8080", Priority: 1},
		{Addr: "host3:8080", Priority: 10}, // Should never be picked
	}

	picker := &PriorityPicker{}
	counts := make(map[string]int)

	numPicks := 100
	for i := 0; i < numPicks; i++ {
		got, err := picker.Pick(addrs)
		if err != nil {
			t.Fatalf("[TestPriorityPickerRoundRobinWithinPriority]: unexpected error: %v", err)
		}
		counts[got.Addr]++
	}

	// host3 should never be picked (higher priority number = lower priority)
	if counts["host3:8080"] != 0 {
		t.Errorf("[TestPriorityPickerRoundRobinWithinPriority]: host3 picked %d times, want 0", counts["host3:8080"])
	}

	// host1 and host2 should be picked roughly equally
	if counts["host1:8080"] != numPicks/2 {
		t.Errorf("[TestPriorityPickerRoundRobinWithinPriority]: host1 picked %d times, want %d", counts["host1:8080"], numPicks/2)
	}
	if counts["host2:8080"] != numPicks/2 {
		t.Errorf("[TestPriorityPickerRoundRobinWithinPriority]: host2 picked %d times, want %d", counts["host2:8080"], numPicks/2)
	}
}
