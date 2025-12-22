package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
)

func TestDiffScalarFields(t *testing.T) {
	tests := []struct {
		name    string
		from    func() cars.Car
		to      func() cars.Car
		wantOps int
		wantErr bool
	}{
		{
			name: "Success: no changes",
			from: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			wantOps: 0,
		},
		{
			name: "Success: number field changed",
			from: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar().SetYear(2024)
			},
			wantOps: 1,
		},
	}

	for _, test := range tests {
		from := test.from()
		to := test.to()

		patch, err := Diff(from, to)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestDiffScalarFields(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestDiffScalarFields(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		gotOps := len(patch.Ops())
		if gotOps != test.wantOps {
			t.Errorf("TestDiffScalarFields(%s): got %d ops, want %d ops", test.name, gotOps, test.wantOps)
		}
	}
}

func TestDiffIsEmpty(t *testing.T) {
	tests := []struct {
		name      string
		from      func() cars.Car
		to        func() cars.Car
		wantEmpty bool
	}{
		{
			name: "Success: identical structs",
			from: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			wantEmpty: true,
		},
		{
			name: "Success: different structs",
			from: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar().SetYear(2024)
			},
			wantEmpty: false,
		},
	}

	for _, test := range tests {
		from := test.from()
		to := test.to()

		patch, err := Diff(from, to)
		if err != nil {
			t.Errorf("TestDiffIsEmpty(%s): unexpected error: %s", test.name, err)
			continue
		}

		gotEmpty := IsEmpty(patch)
		if gotEmpty != test.wantEmpty {
			t.Errorf("TestDiffIsEmpty(%s): got IsEmpty=%v, want IsEmpty=%v", test.name, gotEmpty, test.wantEmpty)
		}
	}
}

func TestDiffPatchSerialization(t *testing.T) {
	from := cars.NewCar().SetYear(2023)
	to := cars.NewCar().SetYear(2024)

	patch, err := Diff(from, to)
	if err != nil {
		t.Fatalf("TestDiffPatchSerialization: Diff error: %s", err)
	}

	// Serialize the patch
	data, err := patch.Marshal()
	if err != nil {
		t.Fatalf("TestDiffPatchSerialization: Marshal error: %s", err)
	}

	if len(data) == 0 {
		t.Fatal("TestDiffPatchSerialization: marshaled data is empty")
	}

	// For now just verify serialization works
	t.Logf("Patch serialized to %d bytes with %d operations", len(data), len(patch.Ops()))
}
