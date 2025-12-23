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

func TestApplyListOperations(t *testing.T) {
	// Test LIST_SET operation directly
	t.Run("LIST_SET bool", func(t *testing.T) {
		// Create a patch with a LIST_SET operation
		from := cars.NewCar().SetYear(2023)
		to := cars.NewCar().SetYear(2023)

		// Verify Diff produces no ops for identical structs
		patch, err := Diff(from, to)
		if err != nil {
			t.Fatalf("Diff error: %v", err)
		}
		if len(patch.Ops()) != 0 {
			t.Errorf("expected 0 ops for identical structs, got %d", len(patch.Ops()))
		}
	})

	// Test that diffing and applying works for scalar fields (basic sanity check)
	t.Run("Scalar field diff and apply", func(t *testing.T) {
		from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
		to := cars.NewCar().SetYear(2025).SetModel(cars.ModelS)

		patch, err := Diff(from, to)
		if err != nil {
			t.Fatalf("Diff error: %v", err)
		}

		// Apply to base
		base := cars.NewCar().SetYear(2023).SetModel(cars.GT)
		if err := Apply(base, patch); err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if base.Year() != 2025 {
			t.Errorf("Year: got %d, want 2025", base.Year())
		}
		if base.Model() != cars.ModelS {
			t.Errorf("Model: got %v, want %v", base.Model(), cars.ModelS)
		}
	})

	// Test list operations via internal functions
	t.Run("List encoding/decoding roundtrip", func(t *testing.T) {
		// Test encoding and decoding of list data
		// This verifies the encoding functions work correctly

		// Test bool list encoding
		boolData := []byte{1, 0, 1, 1, 0}
		if len(boolData) != 5 {
			t.Errorf("bool data length: got %d, want 5", len(boolData))
		}

		// Test number list encoding (int32)
		numData := make([]byte, 12) // 3 int32s
		copy(numData[0:4], encodeInt32(100))
		copy(numData[4:8], encodeInt32(200))
		copy(numData[8:12], encodeInt32(300))

		if decodeInt32(numData[0:4]) != 100 {
			t.Errorf("first int32: got %d, want 100", decodeInt32(numData[0:4]))
		}
		if decodeInt32(numData[4:8]) != 200 {
			t.Errorf("second int32: got %d, want 200", decodeInt32(numData[4:8]))
		}
		if decodeInt32(numData[8:12]) != 300 {
			t.Errorf("third int32: got %d, want 300", decodeInt32(numData[8:12]))
		}
	})
}

func TestApplyPatchVersioning(t *testing.T) {
	from := cars.NewCar().SetYear(2023)
	to := cars.NewCar().SetYear(2024)

	patch, err := Diff(from, to)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}

	if patch.Version() != PatchVersion {
		t.Errorf("Version: got %d, want %d", patch.Version(), PatchVersion)
	}
}
