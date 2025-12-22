package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

func TestApplyScalarFields(t *testing.T) {
	tests := []struct {
		name     string
		base     func() cars.Car
		target   func() cars.Car
		wantYear uint16
		wantErr  bool
	}{
		{
			name: "Success: no changes",
			base: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			target: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			wantYear: 2023,
		},
		{
			name: "Success: year changed",
			base: func() cars.Car {
				return cars.NewCar().SetYear(2023)
			},
			target: func() cars.Car {
				return cars.NewCar().SetYear(2024)
			},
			wantYear: 2024,
		},
	}

	for _, test := range tests {
		base := test.base()
		target := test.target()

		// Create patch from base to target
		patch, err := Diff(base, target)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestApplyScalarFields(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestApplyScalarFields(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Apply patch to a fresh base
		freshBase := test.base()
		if err := Apply(freshBase, patch); err != nil {
			t.Errorf("TestApplyScalarFields(%s): Apply error: %s", test.name, err)
			continue
		}

		// Verify the result
		if freshBase.Year() != test.wantYear {
			t.Errorf("TestApplyScalarFields(%s): got Year=%d, want Year=%d", test.name, freshBase.Year(), test.wantYear)
		}
	}
}

func TestApplyRoundTrip(t *testing.T) {
	// Create two different versions
	from := cars.NewCar().SetYear(2023)
	to := cars.NewCar().SetYear(2024)

	// Generate patch
	patch, err := Diff(from, to)
	if err != nil {
		t.Fatalf("TestApplyRoundTrip: Diff error: %s", err)
	}

	// Serialize the patch
	data, err := patch.Marshal()
	if err != nil {
		t.Fatalf("TestApplyRoundTrip: Marshal error: %s", err)
	}

	// Deserialize
	receivedPatch := msgs.NewPatch()
	if err := receivedPatch.Unmarshal(data); err != nil {
		t.Fatalf("TestApplyRoundTrip: Unmarshal error: %s", err)
	}

	// Apply to a fresh base
	base := cars.NewCar().SetYear(2023)
	if err := Apply(base, receivedPatch); err != nil {
		t.Fatalf("TestApplyRoundTrip: Apply error: %s", err)
	}

	// Verify the result matches target
	if base.Year() != 2024 {
		t.Errorf("TestApplyRoundTrip: got Year=%d, want Year=2024", base.Year())
	}
}

func TestApplyMultipleFields(t *testing.T) {
	// Test with multiple field changes
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.Venza)

	// Generate patch
	patch, err := Diff(from, to)
	if err != nil {
		t.Fatalf("TestApplyMultipleFields: Diff error: %s", err)
	}

	// Apply to a fresh base
	base := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	if err := Apply(base, patch); err != nil {
		t.Fatalf("TestApplyMultipleFields: Apply error: %s", err)
	}

	// Verify both fields changed
	if base.Year() != 2024 {
		t.Errorf("TestApplyMultipleFields: got Year=%d, want Year=2024", base.Year())
	}
	if base.Model() != cars.Venza {
		t.Errorf("TestApplyMultipleFields: got Model=%v, want Model=%v", base.Model(), cars.Venza)
	}
}
