package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

func TestDiffScalarFields(t *testing.T) {
	ctx := t.Context()
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
				return cars.NewCar(ctx).SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			wantOps: 0,
		},
		{
			name: "Success: number field changed",
			from: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2024)
			},
			wantOps: 1,
		},
	}

	for _, test := range tests {
		from := test.from()
		to := test.to()

		p, err := Diff(ctx, from, to)
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

		gotOps := p.OpsLen(ctx)
		if gotOps != test.wantOps {
			t.Errorf("TestDiffScalarFields(%s): got %d ops, want %d ops", test.name, gotOps, test.wantOps)
		}
	}
}

func TestDiffIsEmpty(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name      string
		from      func() cars.Car
		to        func() cars.Car
		wantEmpty bool
	}{
		{
			name: "Success: identical structs",
			from: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			wantEmpty: true,
		},
		{
			name: "Success: different structs",
			from: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2024)
			},
			wantEmpty: false,
		},
	}

	for _, test := range tests {
		from := test.from()
		to := test.to()

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Errorf("TestDiffIsEmpty(%s): unexpected error: %s", test.name, err)
			continue
		}

		gotEmpty := IsEmpty(ctx, p)
		if gotEmpty != test.wantEmpty {
			t.Errorf("TestDiffIsEmpty(%s): got IsEmpty=%v, want IsEmpty=%v", test.name, gotEmpty, test.wantEmpty)
		}
	}
}

func TestDiffPatchSerialization(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name     string
		from     func() cars.Car
		to       func() cars.Car
		wantOps  int
		validate func(t *testing.T, base cars.Car)
	}{
		{
			name: "Success: single field change roundtrip",
			from: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			to: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2024)
			},
			wantOps: 1,
			validate: func(t *testing.T, base cars.Car) {
				if base.Year() != 2024 {
					t.Errorf("Year=%d, want 2024", base.Year())
				}
			},
		},
		{
			name: "Success: multiple field changes roundtrip",
			from: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
			},
			to: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2024).SetModel(cars.Venza)
			},
			wantOps: 2,
			validate: func(t *testing.T, base cars.Car) {
				if base.Year() != 2024 {
					t.Errorf("Year=%d, want 2024", base.Year())
				}
				if base.Model() != cars.Venza {
					t.Errorf("Model=%v, want %v", base.Model(), cars.Venza)
				}
			},
		},
	}

	for _, test := range tests {
		from := test.from()
		to := test.to()

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Errorf("TestDiffPatchSerialization(%s): Diff error: %s", test.name, err)
			continue
		}

		if p.OpsLen(ctx) != test.wantOps {
			t.Errorf("TestDiffPatchSerialization(%s): original patch has %d ops, want %d", test.name, p.OpsLen(ctx), test.wantOps)
			continue
		}

		// Serialize the patch
		data, err := p.Marshal()
		if err != nil {
			t.Errorf("TestDiffPatchSerialization(%s): Marshal error: %s", test.name, err)
			continue
		}

		// Deserialize the patch
		deserializedPatch := msgs.NewPatch(ctx)
		if err := deserializedPatch.Unmarshal(data); err != nil {
			t.Errorf("TestDiffPatchSerialization(%s): Unmarshal error: %s", test.name, err)
			continue
		}

		// Verify deserialized patch has same number of ops
		gotOps := deserializedPatch.OpsLen(ctx)
		if gotOps != test.wantOps {
			t.Errorf("TestDiffPatchSerialization(%s): deserialized patch has %d ops, want %d", test.name, gotOps, test.wantOps)
			continue
		}

		// Verify version is preserved
		if deserializedPatch.Version() != p.Version() {
			t.Errorf("TestDiffPatchSerialization(%s): deserialized version %d, want %d", test.name, deserializedPatch.Version(), p.Version())
		}

		// Apply the roundtripped patch and verify it produces correct result
		base := test.from()
		if err := Apply(ctx, base, deserializedPatch); err != nil {
			t.Errorf("TestDiffPatchSerialization(%s): Apply error: %s", test.name, err)
			continue
		}

		test.validate(t, base)
	}
}
