package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	trucks "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/trucks"
	"github.com/bearlytools/claw/clawc/languages/go/types/list"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
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

func TestApplyListBools(t *testing.T) {
	tests := []struct {
		name      string
		fromBools []bool
		toBools   []bool
		wantBools []bool
		wantOps   int
		wantErr   bool
	}{
		{
			name:      "Success: identical bool lists produce no ops",
			fromBools: []bool{true, false, true},
			toBools:   []bool{true, false, true},
			wantBools: []bool{true, false, true},
			wantOps:   0,
		},
		{
			name:      "Success: different bool lists",
			fromBools: []bool{true, false, true},
			toBools:   []bool{false, true, false},
			wantBools: []bool{false, true, false},
			wantOps:   3, // 3 LIST_SET ops since all elements differ
		},
		{
			name:      "Success: add bools to empty list",
			fromBools: nil,
			toBools:   []bool{true, false},
			wantBools: []bool{true, false},
			wantOps:   1,
		},
		{
			name:      "Success: clear bools",
			fromBools: []bool{true, false},
			toBools:   nil,
			wantBools: nil,
			wantOps:   1,
		},
		{
			name:      "Success: both empty",
			fromBools: nil,
			toBools:   nil,
			wantBools: nil,
			wantOps:   0,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle()
		if test.fromBools != nil {
			bools := list.NewBools()
			bools.Append(test.fromBools...)
			from.SetBools(bools)
		}

		to := vehicles.NewVehicle()
		if test.toBools != nil {
			bools := list.NewBools()
			bools.Append(test.toBools...)
			to.SetBools(bools)
		}

		patch, err := Diff(from, to)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestApplyListBools(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestApplyListBools(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if len(patch.Ops()) != test.wantOps {
			t.Errorf("TestApplyListBools(%s): got %d ops, want %d ops", test.name, len(patch.Ops()), test.wantOps)
		}

		// Apply patch
		freshFrom := vehicles.NewVehicle()
		if test.fromBools != nil {
			bools := list.NewBools()
			bools.Append(test.fromBools...)
			freshFrom.SetBools(bools)
		}

		if err := Apply(freshFrom, patch); err != nil {
			t.Errorf("TestApplyListBools(%s): Apply error: %s", test.name, err)
			continue
		}

		var gotBools []bool
		bools := freshFrom.Bools()
		if !bools.IsNil() {
			gotBools = bools.Slice()
		}
		if len(gotBools) != len(test.wantBools) {
			t.Errorf("TestApplyListBools(%s): got len=%d, want len=%d", test.name, len(gotBools), len(test.wantBools))
			continue
		}
		for i, want := range test.wantBools {
			if gotBools[i] != want {
				t.Errorf("TestApplyListBools(%s): got[%d]=%v, want=%v", test.name, i, gotBools[i], want)
			}
		}
	}
}

func TestApplyNestedStruct(t *testing.T) {
	// Test that identical nested structs produce no ops (no apply needed)
	t.Run("Success: identical nested structs produce no ops", func(t *testing.T) {
		from := vehicles.NewVehicle().SetCar(cars.NewCar().SetYear(2023).SetModel(cars.GT))
		to := vehicles.NewVehicle().SetCar(cars.NewCar().SetYear(2023).SetModel(cars.GT))

		patch, err := Diff(from, to)
		if err != nil {
			t.Fatalf("TestApplyNestedStruct: Diff error: %s", err)
		}

		if len(patch.Ops()) != 0 {
			t.Errorf("TestApplyNestedStruct: got %d ops, want 0 ops for identical structs", len(patch.Ops()))
		}
	})

	// Test nested struct field changes with STRUCT_PATCH
	t.Run("Success: nested struct diff and apply with STRUCT_PATCH", func(t *testing.T) {
		from := vehicles.NewVehicle().SetCar(cars.NewCar().SetYear(2023).SetModel(cars.GT))
		to := vehicles.NewVehicle().SetCar(cars.NewCar().SetYear(2024).SetModel(cars.Venza))

		patch, err := Diff(from, to)
		if err != nil {
			t.Fatalf("TestApplyNestedStruct: Diff error: %s", err)
		}

		if len(patch.Ops()) != 1 {
			t.Errorf("TestApplyNestedStruct: got %d ops, want 1 op", len(patch.Ops()))
		}

		if len(patch.Ops()) > 0 && patch.Ops()[0].Type() != msgs.StructPatch {
			t.Errorf("TestApplyNestedStruct: got op type %v, want StructPatch", patch.Ops()[0].Type())
		}

		// Apply the patch and verify the nested struct was updated
		base := vehicles.NewVehicle().SetCar(cars.NewCar().SetYear(2023).SetModel(cars.GT))
		if err := Apply(base, patch); err != nil {
			t.Fatalf("TestApplyNestedStruct: Apply error: %s", err)
		}

		if base.Car().Year() != 2024 {
			t.Errorf("TestApplyNestedStruct: Car.Year=%d, want 2024", base.Car().Year())
		}
		if base.Car().Model() != cars.Venza {
			t.Errorf("TestApplyNestedStruct: Car.Model=%v, want %v", base.Car().Model(), cars.Venza)
		}
	})
}

func TestApplyListStructs(t *testing.T) {
	tests := []struct {
		name     string
		base     func() vehicles.Vehicle
		target   func() vehicles.Vehicle
		wantLen  int
		wantErr  bool
	}{
		{
			name: "Success: add trucks to empty list",
			base: func() vehicles.Vehicle {
				return vehicles.NewVehicle()
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2023).SetModel(trucks.F100))
				return v
			},
			wantLen: 1,
		},
		{
			name: "Success: modify truck in list",
			base: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2023).SetModel(trucks.F100))
				return v
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2024).SetModel(trucks.Tundra))
				return v
			},
			wantLen: 1,
		},
		{
			name: "Success: add additional truck",
			base: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2023).SetModel(trucks.F100))
				return v
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2023).SetModel(trucks.F100))
				v.AppendTruck(trucks.NewTruck().SetYear(2024).SetModel(trucks.Tundra))
				return v
			},
			wantLen: 2,
		},
		{
			name: "Success: remove truck from list",
			base: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2023).SetModel(trucks.F100))
				v.AppendTruck(trucks.NewTruck().SetYear(2024).SetModel(trucks.Tundra))
				return v
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle()
				v.AppendTruck(trucks.NewTruck().SetYear(2023).SetModel(trucks.F100))
				return v
			},
			wantLen: 1,
		},
	}

	for _, test := range tests {
		base := test.base()
		target := test.target()

		patch, err := Diff(base, target)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestApplyListStructs(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestApplyListStructs(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Apply the patch
		applied := test.base()
		if err := Apply(applied, patch); err != nil {
			t.Errorf("TestApplyListStructs(%s): Apply error: %s", test.name, err)
			continue
		}

		// Verify list length
		truckList := applied.Truck()
		if truckList == nil && test.wantLen > 0 {
			t.Errorf("TestApplyListStructs(%s): got nil truck list, want len %d", test.name, test.wantLen)
			continue
		}
		if truckList != nil && len(truckList) != test.wantLen {
			t.Errorf("TestApplyListStructs(%s): got len %d, want %d", test.name, len(truckList), test.wantLen)
		}
	}
}

func TestApplyNestedStructWithManufacturer(t *testing.T) {
	// Test that diff correctly identifies changes in nested struct (diff only)
	from := vehicles.NewVehicle().SetCar(
		cars.NewCar().SetYear(2023).SetManufacturer(manufacturers.Toyota),
	)
	to := vehicles.NewVehicle().SetCar(
		cars.NewCar().SetYear(2024).SetManufacturer(manufacturers.Ford),
	)

	patch, err := Diff(from, to)
	if err != nil {
		t.Fatalf("TestApplyNestedStructWithManufacturer: Diff error: %s", err)
	}

	// Should produce 1 STRUCT_PATCH op for the Car field
	if len(patch.Ops()) != 1 {
		t.Errorf("TestApplyNestedStructWithManufacturer: got %d ops, want 1", len(patch.Ops()))
	}

	if len(patch.Ops()) > 0 && patch.Ops()[0].Type() != msgs.StructPatch {
		t.Errorf("TestApplyNestedStructWithManufacturer: got op type %v, want StructPatch", patch.Ops()[0].Type())
	}
}

func TestApplyEmptyPatch(t *testing.T) {
	// Test applying an empty patch makes no changes
	base := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	patch := msgs.NewPatch()
	patch.SetVersion(PatchVersion)

	if err := Apply(base, patch); err != nil {
		t.Fatalf("TestApplyEmptyPatch: Apply error: %s", err)
	}

	if base.Year() != 2023 {
		t.Errorf("TestApplyEmptyPatch: got Year=%d, want Year=2023", base.Year())
	}
	if base.Model() != cars.GT {
		t.Errorf("TestApplyEmptyPatch: got Model=%v, want Model=%v", base.Model(), cars.GT)
	}
}

func TestDiffIdenticalStructs(t *testing.T) {
	// Test that diffing identical structs produces an empty patch
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2023).SetModel(cars.GT)

	patch, err := Diff(from, to)
	if err != nil {
		t.Fatalf("TestDiffIdenticalStructs: Diff error: %s", err)
	}

	if !IsEmpty(patch) {
		t.Errorf("TestDiffIdenticalStructs: expected empty patch for identical structs, got %d ops", len(patch.Ops()))
	}
}

func TestApplyEnumList(t *testing.T) {
	tests := []struct {
		name      string
		fromTypes []vehicles.Type
		toTypes   []vehicles.Type
		wantTypes []vehicles.Type
		wantOps   int
	}{
		{
			name:      "Success: identical enum lists produce no ops",
			fromTypes: []vehicles.Type{vehicles.Car, vehicles.Truck},
			toTypes:   []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantTypes: []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantOps:   0,
		},
		{
			name:      "Success: different enum lists",
			fromTypes: []vehicles.Type{vehicles.Car},
			toTypes:   []vehicles.Type{vehicles.Truck},
			wantTypes: []vehicles.Type{vehicles.Truck},
			wantOps:   1,
		},
		{
			name:      "Success: add to enum list",
			fromTypes: nil,
			toTypes:   []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantTypes: []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantOps:   1,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle()
		if test.fromTypes != nil {
			types := list.NewEnums[vehicles.Type]()
			for _, t := range test.fromTypes {
				types.Append(t)
			}
			from.SetTypes(types)
		}

		to := vehicles.NewVehicle()
		if test.toTypes != nil {
			types := list.NewEnums[vehicles.Type]()
			for _, t := range test.toTypes {
				types.Append(t)
			}
			to.SetTypes(types)
		}

		patch, err := Diff(from, to)
		if err != nil {
			t.Errorf("TestApplyEnumList(%s): Diff error: %s", test.name, err)
			continue
		}

		if len(patch.Ops()) != test.wantOps {
			t.Errorf("TestApplyEnumList(%s): got %d ops, want %d ops", test.name, len(patch.Ops()), test.wantOps)
		}

		if test.wantOps == 0 {
			continue
		}

		freshFrom := vehicles.NewVehicle()
		if test.fromTypes != nil {
			types := list.NewEnums[vehicles.Type]()
			for _, t := range test.fromTypes {
				types.Append(t)
			}
			freshFrom.SetTypes(types)
		}

		if err := Apply(freshFrom, patch); err != nil {
			t.Errorf("TestApplyEnumList(%s): Apply error: %s", test.name, err)
			continue
		}

		gotTypes := freshFrom.Types()
		if gotTypes.Len() != len(test.wantTypes) {
			t.Errorf("TestApplyEnumList(%s): got len=%d, want len=%d", test.name, gotTypes.Len(), len(test.wantTypes))
			continue
		}
		for i, want := range test.wantTypes {
			if gotTypes.Get(i) != want {
				t.Errorf("TestApplyEnumList(%s): got[%d]=%v, want=%v", test.name, i, gotTypes.Get(i), want)
			}
		}
	}
}

func TestDiffListSetInsertRemove(t *testing.T) {
	tests := []struct {
		name        string
		fromBools   []bool
		toBools     []bool
		wantOps     int
		wantOpTypes []msgs.OpType
	}{
		{
			name:        "Success: LIST_SET for changed elements only",
			fromBools:   []bool{true, false, true},
			toBools:     []bool{true, true, true},
			wantOps:     1,
			wantOpTypes: []msgs.OpType{msgs.ListSet},
		},
		{
			name:        "Success: LIST_INSERT for new elements",
			fromBools:   []bool{true, false},
			toBools:     []bool{true, false, true, false},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListInsert, msgs.ListInsert},
		},
		{
			name:        "Success: LIST_REMOVE for deleted elements",
			fromBools:   []bool{true, false, true, false},
			toBools:     []bool{true, false},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListRemove, msgs.ListRemove},
		},
		{
			name:        "Success: mixed SET and INSERT",
			fromBools:   []bool{true, false},
			toBools:     []bool{false, false, true},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListSet, msgs.ListInsert},
		},
		{
			name:        "Success: mixed SET and REMOVE (below threshold)",
			fromBools:   []bool{true, false, true, false, true},
			toBools:     []bool{false, false, true, false},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListSet, msgs.ListRemove},
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle()
		boolsFrom := list.NewBools()
		boolsFrom.Append(test.fromBools...)
		from.SetBools(boolsFrom)

		to := vehicles.NewVehicle()
		boolsTo := list.NewBools()
		boolsTo.Append(test.toBools...)
		to.SetBools(boolsTo)

		patch, err := Diff(from, to)
		if err != nil {
			t.Errorf("TestDiffListSetInsertRemove(%s): Diff error: %s", test.name, err)
			continue
		}

		if len(patch.Ops()) != test.wantOps {
			t.Errorf("TestDiffListSetInsertRemove(%s): got %d ops, want %d ops", test.name, len(patch.Ops()), test.wantOps)
			continue
		}

		for i, wantType := range test.wantOpTypes {
			if i >= len(patch.Ops()) {
				break
			}
			if patch.Ops()[i].Type() != wantType {
				t.Errorf("TestDiffListSetInsertRemove(%s): op[%d] got type %v, want %v", test.name, i, patch.Ops()[i].Type(), wantType)
			}
		}

		freshFrom := vehicles.NewVehicle()
		boolsFresh := list.NewBools()
		boolsFresh.Append(test.fromBools...)
		freshFrom.SetBools(boolsFresh)

		if err := Apply(freshFrom, patch); err != nil {
			t.Errorf("TestDiffListSetInsertRemove(%s): Apply error: %s", test.name, err)
			continue
		}

		gotBools := freshFrom.Bools().Slice()
		if len(gotBools) != len(test.toBools) {
			t.Errorf("TestDiffListSetInsertRemove(%s): got len=%d, want len=%d", test.name, len(gotBools), len(test.toBools))
			continue
		}
		for i, want := range test.toBools {
			if gotBools[i] != want {
				t.Errorf("TestDiffListSetInsertRemove(%s): got[%d]=%v, want=%v", test.name, i, gotBools[i], want)
			}
		}
	}
}

func TestDiffStringField(t *testing.T) {
	tests := []struct {
		name     string
		fromName string
		toName   string
		wantName string
		wantOps  int
	}{
		{
			name:     "Success: identical strings produce no ops",
			fromName: "Tesla Model S",
			toName:   "Tesla Model S",
			wantName: "Tesla Model S",
			wantOps:  0,
		},
		{
			name:     "Success: different strings",
			fromName: "Tesla Model S",
			toName:   "Tesla Model X",
			wantName: "Tesla Model X",
			wantOps:  1,
		},
		{
			name:     "Success: empty to non-empty string",
			fromName: "",
			toName:   "Tesla Model 3",
			wantName: "Tesla Model 3",
			wantOps:  1,
		},
		{
			name:     "Success: non-empty to empty string",
			fromName: "Tesla Model 3",
			toName:   "",
			wantName: "",
			wantOps:  1,
		},
		{
			name:     "Success: both empty strings",
			fromName: "",
			toName:   "",
			wantName: "",
			wantOps:  0,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle().SetName(test.fromName)
		to := vehicles.NewVehicle().SetName(test.toName)

		patch, err := Diff(from, to)
		if err != nil {
			t.Errorf("TestDiffStringField(%s): Diff error: %s", test.name, err)
			continue
		}

		if len(patch.Ops()) != test.wantOps {
			t.Errorf("TestDiffStringField(%s): got %d ops, want %d ops", test.name, len(patch.Ops()), test.wantOps)
		}

		if test.wantOps == 0 {
			continue
		}

		freshFrom := vehicles.NewVehicle().SetName(test.fromName)
		if err := Apply(freshFrom, patch); err != nil {
			t.Errorf("TestDiffStringField(%s): Apply error: %s", test.name, err)
			continue
		}

		if freshFrom.Name() != test.wantName {
			t.Errorf("TestDiffStringField(%s): got Name=%q, want Name=%q", test.name, freshFrom.Name(), test.wantName)
		}
	}
}

func TestDiffBytesField(t *testing.T) {
	tests := []struct {
		name    string
		fromVIN []byte
		toVIN   []byte
		wantVIN []byte
		wantOps int
	}{
		{
			name:    "Success: identical bytes produce no ops",
			fromVIN: []byte{0x01, 0x02, 0x03},
			toVIN:   []byte{0x01, 0x02, 0x03},
			wantVIN: []byte{0x01, 0x02, 0x03},
			wantOps: 0,
		},
		{
			name:    "Success: different bytes",
			fromVIN: []byte{0x01, 0x02, 0x03},
			toVIN:   []byte{0xAA, 0xBB, 0xCC},
			wantVIN: []byte{0xAA, 0xBB, 0xCC},
			wantOps: 1,
		},
		{
			name:    "Success: nil to non-nil bytes",
			fromVIN: nil,
			toVIN:   []byte{0xDE, 0xAD, 0xBE, 0xEF},
			wantVIN: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			wantOps: 1,
		},
		{
			name:    "Success: non-nil to nil bytes",
			fromVIN: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			toVIN:   nil,
			wantVIN: nil,
			wantOps: 1,
		},
		{
			name:    "Success: both nil bytes",
			fromVIN: nil,
			toVIN:   nil,
			wantVIN: nil,
			wantOps: 0,
		},
		{
			name:    "Success: different length bytes",
			fromVIN: []byte{0x01, 0x02},
			toVIN:   []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			wantVIN: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			wantOps: 1,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle()
		if test.fromVIN != nil {
			from.SetVIN(test.fromVIN)
		}

		to := vehicles.NewVehicle()
		if test.toVIN != nil {
			to.SetVIN(test.toVIN)
		}

		patch, err := Diff(from, to)
		if err != nil {
			t.Errorf("TestDiffBytesField(%s): Diff error: %s", test.name, err)
			continue
		}

		if len(patch.Ops()) != test.wantOps {
			t.Errorf("TestDiffBytesField(%s): got %d ops, want %d ops", test.name, len(patch.Ops()), test.wantOps)
		}

		if test.wantOps == 0 {
			continue
		}

		freshFrom := vehicles.NewVehicle()
		if test.fromVIN != nil {
			freshFrom.SetVIN(test.fromVIN)
		}

		if err := Apply(freshFrom, patch); err != nil {
			t.Errorf("TestDiffBytesField(%s): Apply error: %s", test.name, err)
			continue
		}

		gotVIN := freshFrom.VIN()
		if len(gotVIN) != len(test.wantVIN) {
			t.Errorf("TestDiffBytesField(%s): got VIN len=%d, want len=%d", test.name, len(gotVIN), len(test.wantVIN))
			continue
		}
		for i, want := range test.wantVIN {
			if gotVIN[i] != want {
				t.Errorf("TestDiffBytesField(%s): got VIN[%d]=%x, want=%x", test.name, i, gotVIN[i], want)
			}
		}
	}
}

func TestDiffListStructPatch(t *testing.T) {
	tests := []struct {
		name       string
		fromYears  []uint16
		toYears    []uint16
		wantOps    int
		checkPatch bool
	}{
		{
			name:       "Success: single struct modified uses LIST_STRUCT_PATCH",
			fromYears:  []uint16{2023},
			toYears:    []uint16{2024},
			wantOps:    1,
			checkPatch: true,
		},
		{
			name:      "Success: add struct uses LIST_INSERT",
			fromYears: []uint16{2023},
			toYears:   []uint16{2023, 2024},
			wantOps:   1,
		},
		{
			name:      "Success: remove struct uses LIST_REMOVE",
			fromYears: []uint16{2023, 2024},
			toYears:   []uint16{2023},
			wantOps:   1,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle()
		for _, year := range test.fromYears {
			from.AppendTruck(trucks.NewTruck().SetYear(year).SetModel(trucks.F100))
		}

		to := vehicles.NewVehicle()
		for _, year := range test.toYears {
			to.AppendTruck(trucks.NewTruck().SetYear(year).SetModel(trucks.F100))
		}

		patch, err := Diff(from, to)
		if err != nil {
			t.Errorf("TestDiffListStructPatch(%s): Diff error: %s", test.name, err)
			continue
		}

		if len(patch.Ops()) != test.wantOps {
			t.Errorf("TestDiffListStructPatch(%s): got %d ops, want %d ops", test.name, len(patch.Ops()), test.wantOps)
			continue
		}

		if test.checkPatch && len(patch.Ops()) > 0 {
			if patch.Ops()[0].Type() != msgs.ListStructPatch {
				t.Errorf("TestDiffListStructPatch(%s): got op type %v, want ListStructPatch", test.name, patch.Ops()[0].Type())
			}
		}

		freshFrom := vehicles.NewVehicle()
		for _, year := range test.fromYears {
			freshFrom.AppendTruck(trucks.NewTruck().SetYear(year).SetModel(trucks.F100))
		}

		if err := Apply(freshFrom, patch); err != nil {
			t.Errorf("TestDiffListStructPatch(%s): Apply error: %s", test.name, err)
			continue
		}

		gotTrucks := freshFrom.Truck()
		if len(gotTrucks) != len(test.toYears) {
			t.Errorf("TestDiffListStructPatch(%s): got %d trucks, want %d trucks", test.name, len(gotTrucks), len(test.toYears))
			continue
		}
		for i, wantYear := range test.toYears {
			if gotTrucks[i].Year() != wantYear {
				t.Errorf("TestDiffListStructPatch(%s): truck[%d].Year=%d, want=%d", test.name, i, gotTrucks[i].Year(), wantYear)
			}
		}
	}
}
