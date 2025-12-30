package testing

import (
	"context"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/kylelemons/godebug/pretty"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
)

// tokenCompare is a simplified token representation for comparison.
type tokenCompare struct {
	Kind       clawiter.TokenKind
	Name       string
	Type       field.Type
	Value      any
	IsEnum     bool
	EnumGroup  string
	EnumName   string
	StructName string
	IsNil      bool
	Len        int
}

func toTokenCompare(tok clawiter.Token) tokenCompare {
	tc := tokenCompare{
		Kind:       tok.Kind,
		Name:       tok.Name,
		Type:       tok.Type,
		IsEnum:     tok.IsEnum,
		EnumGroup:  tok.EnumGroup,
		EnumName:   tok.EnumName,
		StructName: tok.StructName,
		IsNil:      tok.IsNil,
		Len:        tok.Len,
	}
	switch tok.Type {
	case field.FTBool:
		tc.Value = tok.Bool()
	case field.FTInt8:
		tc.Value = tok.Int8()
	case field.FTInt16:
		tc.Value = tok.Int16()
	case field.FTInt32:
		tc.Value = tok.Int32()
	case field.FTInt64:
		tc.Value = tok.Int64()
	case field.FTUint8:
		tc.Value = tok.Uint8()
	case field.FTUint16:
		tc.Value = tok.Uint16()
	case field.FTUint32:
		tc.Value = tok.Uint32()
	case field.FTUint64:
		tc.Value = tok.Uint64()
	case field.FTFloat32:
		tc.Value = tok.Float32()
	case field.FTFloat64:
		tc.Value = tok.Float64()
	case field.FTString:
		tc.Value = tok.String()
	case field.FTBytes:
		tc.Value = tok.Bytes
	}
	return tc
}

func TestWalkVehicle(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		setup func() vehicles.Vehicle
		want  []tokenCompare
	}{
		{
			name: "Success: basic vehicle with car",
			setup: func() vehicles.Vehicle {
				car := cars.NewCar(ctx).
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Car).
					SetCar(car)
			},
			want: []tokenCompare{
				{Kind: clawiter.TokenStructStart, Name: "Vehicle"},
				{Kind: clawiter.TokenField, Name: "Type", Type: field.FTUint8, Value: uint8(1), IsEnum: true, EnumGroup: "Type", EnumName: "Car"},
				{Kind: clawiter.TokenField, Name: "Car", Type: field.FTStruct, StructName: "cars.Car"},
				{Kind: clawiter.TokenStructStart, Name: "Car"},
				{Kind: clawiter.TokenField, Name: "Manufacturer", Type: field.FTUint8, Value: uint8(1), IsEnum: true, EnumGroup: "Manufacturer", EnumName: "Toyota"},
				{Kind: clawiter.TokenField, Name: "Model", Type: field.FTUint8, Value: uint8(2), IsEnum: true, EnumGroup: "Model", EnumName: "Venza"},
				{Kind: clawiter.TokenField, Name: "Year", Type: field.FTUint16, Value: uint16(2010)},
				{Kind: clawiter.TokenStructEnd, Name: "Car"},
				{Kind: clawiter.TokenField, Name: "Truck", Type: field.FTListStructs, StructName: "trucks.Truck", IsNil: true},
				{Kind: clawiter.TokenField, Name: "Types", Type: field.FTListUint8, IsNil: true, IsEnum: true, EnumGroup: "Type"},
				{Kind: clawiter.TokenField, Name: "Bools", Type: field.FTListBools, IsNil: true},
				{Kind: clawiter.TokenStructEnd, Name: "Vehicle"},
			},
		},
		{
			name: "Success: vehicle with enum list",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Truck).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
			want: []tokenCompare{
				{Kind: clawiter.TokenStructStart, Name: "Vehicle"},
				{Kind: clawiter.TokenField, Name: "Type", Type: field.FTUint8, Value: uint8(2), IsEnum: true, EnumGroup: "Type", EnumName: "Truck"},
				{Kind: clawiter.TokenField, Name: "Car", Type: field.FTStruct, StructName: "cars.Car", IsNil: true},
				{Kind: clawiter.TokenField, Name: "Truck", Type: field.FTListStructs, StructName: "trucks.Truck", IsNil: true},
				{Kind: clawiter.TokenField, Name: "Types", Type: field.FTListUint8, IsEnum: true, EnumGroup: "Type"},
				{Kind: clawiter.TokenListStart, Name: "Types", Type: field.FTListUint8, Len: 2},
				{Kind: clawiter.TokenField, Type: field.FTUint8, Value: uint8(1), IsEnum: true, EnumGroup: "Type", EnumName: "Car"},
				{Kind: clawiter.TokenField, Type: field.FTUint8, Value: uint8(2), IsEnum: true, EnumGroup: "Type", EnumName: "Truck"},
				{Kind: clawiter.TokenListEnd, Name: "Types"},
				{Kind: clawiter.TokenField, Name: "Bools", Type: field.FTListBools, IsNil: true},
				{Kind: clawiter.TokenStructEnd, Name: "Vehicle"},
			},
		},
		{
			name: "Success: vehicle with bool list",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetBools(true, false, true)
			},
			want: []tokenCompare{
				{Kind: clawiter.TokenStructStart, Name: "Vehicle"},
				{Kind: clawiter.TokenField, Name: "Type", Type: field.FTUint8, Value: uint8(0), IsEnum: true, EnumGroup: "Type", EnumName: "Unknown"},
				{Kind: clawiter.TokenField, Name: "Car", Type: field.FTStruct, StructName: "cars.Car", IsNil: true},
				{Kind: clawiter.TokenField, Name: "Truck", Type: field.FTListStructs, StructName: "trucks.Truck", IsNil: true},
				{Kind: clawiter.TokenField, Name: "Types", Type: field.FTListUint8, IsNil: true, IsEnum: true, EnumGroup: "Type"},
				{Kind: clawiter.TokenField, Name: "Bools", Type: field.FTListBools},
				{Kind: clawiter.TokenListStart, Name: "Bools", Type: field.FTListBools, Len: 3},
				{Kind: clawiter.TokenField, Type: field.FTBool, Value: true},
				{Kind: clawiter.TokenField, Type: field.FTBool, Value: false},
				{Kind: clawiter.TokenField, Type: field.FTBool, Value: true},
				{Kind: clawiter.TokenListEnd, Name: "Bools"},
				{Kind: clawiter.TokenStructEnd, Name: "Vehicle"},
			},
		},
	}

	for _, test := range tests {
		v := test.setup()
		var got []tokenCompare
		for tok := range v.Walk() {
			got = append(got, toTokenCompare(tok))
		}
		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("TestWalkVehicle(%s): -want/+got:\n%s", test.name, diff)
		}
	}
}

func TestWalkCar(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		setup func() cars.Car
		want  []tokenCompare
	}{
		{
			name: "Success: basic car",
			setup: func() cars.Car {
				return cars.NewCar(ctx).
					SetManufacturer(manufacturers.Tesla).
					SetModel(cars.ModelS).
					SetYear(2023)
			},
			want: []tokenCompare{
				{Kind: clawiter.TokenStructStart, Name: "Car"},
				{Kind: clawiter.TokenField, Name: "Manufacturer", Type: field.FTUint8, Value: uint8(3), IsEnum: true, EnumGroup: "Manufacturer", EnumName: "Tesla"},
				{Kind: clawiter.TokenField, Name: "Model", Type: field.FTUint8, Value: uint8(3), IsEnum: true, EnumGroup: "Model", EnumName: "ModelS"},
				{Kind: clawiter.TokenField, Name: "Year", Type: field.FTUint16, Value: uint16(2023)},
				{Kind: clawiter.TokenStructEnd, Name: "Car"},
			},
		},
		{
			name: "Success: empty car",
			setup: func() cars.Car {
				return cars.NewCar(ctx)
			},
			want: []tokenCompare{
				{Kind: clawiter.TokenStructStart, Name: "Car"},
				{Kind: clawiter.TokenField, Name: "Manufacturer", Type: field.FTUint8, Value: uint8(0), IsEnum: true, EnumGroup: "Manufacturer", EnumName: "Unknown"},
				{Kind: clawiter.TokenField, Name: "Model", Type: field.FTUint8, Value: uint8(0), IsEnum: true, EnumGroup: "Model", EnumName: "ModelUnknown"},
				{Kind: clawiter.TokenField, Name: "Year", Type: field.FTUint16, Value: uint16(0)},
				{Kind: clawiter.TokenStructEnd, Name: "Car"},
			},
		},
	}

	for _, test := range tests {
		c := test.setup()
		var got []tokenCompare
		for tok := range c.Walk() {
			got = append(got, toTokenCompare(tok))
		}
		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("TestWalkCar(%s): -want/+got:\n%s", test.name, diff)
		}
	}
}

func TestTokenAccessors(t *testing.T) {
	tests := []struct {
		name     string
		setToken func(*clawiter.Token)
		getValue func(clawiter.Token) any
		want     any
	}{
		{
			name:     "Success: bool true",
			setToken: func(tok *clawiter.Token) { tok.SetBool(true) },
			getValue: func(tok clawiter.Token) any { return tok.Bool() },
			want:     true,
		},
		{
			name:     "Success: bool false",
			setToken: func(tok *clawiter.Token) { tok.SetBool(false) },
			getValue: func(tok clawiter.Token) any { return tok.Bool() },
			want:     false,
		},
		{
			name:     "Success: int8",
			setToken: func(tok *clawiter.Token) { tok.SetInt8(-42) },
			getValue: func(tok clawiter.Token) any { return tok.Int8() },
			want:     int8(-42),
		},
		{
			name:     "Success: int16",
			setToken: func(tok *clawiter.Token) { tok.SetInt16(-1000) },
			getValue: func(tok clawiter.Token) any { return tok.Int16() },
			want:     int16(-1000),
		},
		{
			name:     "Success: int32",
			setToken: func(tok *clawiter.Token) { tok.SetInt32(-100000) },
			getValue: func(tok clawiter.Token) any { return tok.Int32() },
			want:     int32(-100000),
		},
		{
			name:     "Success: int64",
			setToken: func(tok *clawiter.Token) { tok.SetInt64(-9223372036854775808) },
			getValue: func(tok clawiter.Token) any { return tok.Int64() },
			want:     int64(-9223372036854775808),
		},
		{
			name:     "Success: uint8",
			setToken: func(tok *clawiter.Token) { tok.SetUint8(255) },
			getValue: func(tok clawiter.Token) any { return tok.Uint8() },
			want:     uint8(255),
		},
		{
			name:     "Success: uint16",
			setToken: func(tok *clawiter.Token) { tok.SetUint16(65535) },
			getValue: func(tok clawiter.Token) any { return tok.Uint16() },
			want:     uint16(65535),
		},
		{
			name:     "Success: uint32",
			setToken: func(tok *clawiter.Token) { tok.SetUint32(4294967295) },
			getValue: func(tok clawiter.Token) any { return tok.Uint32() },
			want:     uint32(4294967295),
		},
		{
			name:     "Success: uint64",
			setToken: func(tok *clawiter.Token) { tok.SetUint64(18446744073709551615) },
			getValue: func(tok clawiter.Token) any { return tok.Uint64() },
			want:     uint64(18446744073709551615),
		},
		{
			name:     "Success: float32",
			setToken: func(tok *clawiter.Token) { tok.SetFloat32(3.14) },
			getValue: func(tok clawiter.Token) any { return tok.Float32() },
			want:     float32(3.14),
		},
		{
			name:     "Success: float64",
			setToken: func(tok *clawiter.Token) { tok.SetFloat64(3.14159265358979) },
			getValue: func(tok clawiter.Token) any { return tok.Float64() },
			want:     3.14159265358979,
		},
		{
			name: "Success: string",
			setToken: func(tok *clawiter.Token) {
				tok.Bytes = []byte("hello world")
			},
			getValue: func(tok clawiter.Token) any { return tok.String() },
			want:     "hello world",
		},
		{
			name:     "Success: empty string",
			setToken: func(tok *clawiter.Token) {},
			getValue: func(tok clawiter.Token) any { return tok.String() },
			want:     "",
		},
		// Map key accessors
		{
			name:     "Success: key bool true",
			setToken: func(tok *clawiter.Token) { tok.SetKeyBool(true) },
			getValue: func(tok clawiter.Token) any { return tok.KeyBool() },
			want:     true,
		},
		{
			name:     "Success: key bool false",
			setToken: func(tok *clawiter.Token) { tok.SetKeyBool(false) },
			getValue: func(tok clawiter.Token) any { return tok.KeyBool() },
			want:     false,
		},
		{
			name:     "Success: key int8",
			setToken: func(tok *clawiter.Token) { tok.SetKeyInt8(-42) },
			getValue: func(tok clawiter.Token) any { return tok.KeyInt8() },
			want:     int8(-42),
		},
		{
			name:     "Success: key int16",
			setToken: func(tok *clawiter.Token) { tok.SetKeyInt16(-1000) },
			getValue: func(tok clawiter.Token) any { return tok.KeyInt16() },
			want:     int16(-1000),
		},
		{
			name:     "Success: key int32",
			setToken: func(tok *clawiter.Token) { tok.SetKeyInt32(-100000) },
			getValue: func(tok clawiter.Token) any { return tok.KeyInt32() },
			want:     int32(-100000),
		},
		{
			name:     "Success: key int64",
			setToken: func(tok *clawiter.Token) { tok.SetKeyInt64(-9223372036854775808) },
			getValue: func(tok clawiter.Token) any { return tok.KeyInt64() },
			want:     int64(-9223372036854775808),
		},
		{
			name:     "Success: key uint8",
			setToken: func(tok *clawiter.Token) { tok.SetKeyUint8(255) },
			getValue: func(tok clawiter.Token) any { return tok.KeyUint8() },
			want:     uint8(255),
		},
		{
			name:     "Success: key uint16",
			setToken: func(tok *clawiter.Token) { tok.SetKeyUint16(65535) },
			getValue: func(tok clawiter.Token) any { return tok.KeyUint16() },
			want:     uint16(65535),
		},
		{
			name:     "Success: key uint32",
			setToken: func(tok *clawiter.Token) { tok.SetKeyUint32(4294967295) },
			getValue: func(tok clawiter.Token) any { return tok.KeyUint32() },
			want:     uint32(4294967295),
		},
		{
			name:     "Success: key uint64",
			setToken: func(tok *clawiter.Token) { tok.SetKeyUint64(18446744073709551615) },
			getValue: func(tok clawiter.Token) any { return tok.KeyUint64() },
			want:     uint64(18446744073709551615),
		},
		{
			name:     "Success: key float32",
			setToken: func(tok *clawiter.Token) { tok.SetKeyFloat32(3.14) },
			getValue: func(tok clawiter.Token) any { return tok.KeyFloat32() },
			want:     float32(3.14),
		},
		{
			name:     "Success: key float64",
			setToken: func(tok *clawiter.Token) { tok.SetKeyFloat64(3.14159265358979) },
			getValue: func(tok clawiter.Token) any { return tok.KeyFloat64() },
			want:     3.14159265358979,
		},
		{
			name: "Success: key string",
			setToken: func(tok *clawiter.Token) {
				tok.KeyBytes = []byte("hello key")
			},
			getValue: func(tok clawiter.Token) any { return tok.KeyString() },
			want:     "hello key",
		},
		{
			name:     "Success: empty key string",
			setToken: func(tok *clawiter.Token) {},
			getValue: func(tok clawiter.Token) any { return tok.KeyString() },
			want:     "",
		},
	}

	for _, test := range tests {
		var tok clawiter.Token
		test.setToken(&tok)
		got := test.getValue(tok)
		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("TestTokenAccessors(%s): -want/+got:\n%s", test.name, diff)
		}
	}
}
