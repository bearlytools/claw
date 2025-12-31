package clawjson

import (
	"bytes"
	"context"
	"strings"
	"testing"

	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
)

// createSimpleCar creates a simple car for benchmarking.
func createSimpleCar(ctx context.Context) cars.Car {
	return cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)
}

// createVehicleWithCar creates a vehicle with a nested car for benchmarking.
func createVehicleWithCar(ctx context.Context) vehicles.Vehicle {
	car := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)
	return vehicles.NewVehicle(ctx).
		SetType(vehicles.Car).
		SetCar(car)
}

// createVehicleWithLists creates a vehicle with lists for benchmarking.
func createVehicleWithLists(ctx context.Context) vehicles.Vehicle {
	return vehicles.NewVehicle(ctx).
		SetBools(true, false, true, false, true).
		SetTypes(vehicles.Car, vehicles.Truck, vehicles.Car)
}

// createFullVehicle creates a vehicle with all fields populated for benchmarking.
func createFullVehicle(ctx context.Context) vehicles.Vehicle {
	car := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Ford).
		SetModel(cars.GT).
		SetYear(2020)
	return vehicles.NewVehicle(ctx).
		SetType(vehicles.Car).
		SetCar(car).
		SetTypes(vehicles.Car, vehicles.Truck).
		SetBools(true, false, true)
}

// Marshaling Benchmarks

func BenchmarkMarshalSimpleCar(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalNestedStruct(b *testing.B) {
	ctx := context.Background()
	vehicle := createVehicleWithCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, vehicle)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalWithLists(b *testing.B) {
	ctx := context.Background()
	vehicle := createVehicleWithLists(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, vehicle)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalEnumAsNumbers(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car, WithUseEnumNumbers(true))
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalWriter(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := MarshalWriter(ctx, car, &buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalFullVehicle(b *testing.B) {
	ctx := context.Background()
	vehicle := createFullVehicle(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, vehicle)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

// Unmarshaling Benchmarks

func BenchmarkUnmarshalSimpleCar(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	jsonData := make([]byte, buf.Len())
	copy(jsonData, buf.Bytes())
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := cars.NewCar(ctx)
		err := Unmarshal(ctx, jsonData, &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalNestedStruct(b *testing.B) {
	ctx := context.Background()
	vehicle := createVehicleWithCar(ctx)
	buf, err := Marshal(ctx, vehicle)
	if err != nil {
		b.Fatal(err)
	}
	jsonData := make([]byte, buf.Len())
	copy(jsonData, buf.Bytes())
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := vehicles.NewVehicle(ctx)
		err := Unmarshal(ctx, jsonData, &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalWithLists(b *testing.B) {
	ctx := context.Background()
	vehicle := createVehicleWithLists(ctx)
	buf, err := Marshal(ctx, vehicle)
	if err != nil {
		b.Fatal(err)
	}
	jsonData := make([]byte, buf.Len())
	copy(jsonData, buf.Bytes())
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := vehicles.NewVehicle(ctx)
		err := Unmarshal(ctx, jsonData, &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalReader(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	jsonStr := buf.String()
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := cars.NewCar(ctx)
		err := UnmarshalReader(ctx, strings.NewReader(jsonStr), &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalFullVehicle(b *testing.B) {
	ctx := context.Background()
	vehicle := createFullVehicle(ctx)
	buf, err := Marshal(ctx, vehicle)
	if err != nil {
		b.Fatal(err)
	}
	jsonData := make([]byte, buf.Len())
	copy(jsonData, buf.Bytes())
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := vehicles.NewVehicle(ctx)
		err := Unmarshal(ctx, jsonData, &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Round-trip Benchmarks

func BenchmarkRoundTripSimple(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		restored := cars.NewCar(ctx)
		err = Unmarshal(ctx, buf.Bytes(), &restored)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkRoundTripComplex(b *testing.B) {
	ctx := context.Background()
	vehicle := createFullVehicle(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, vehicle)
		if err != nil {
			b.Fatal(err)
		}
		restored := vehicles.NewVehicle(ctx)
		err = Unmarshal(ctx, buf.Bytes(), &restored)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

// Streaming Array Benchmarks

func BenchmarkArraySingleWrite(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		a, err := NewArray(&buf)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Write(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArrayMultipleWrites(b *testing.B) {
	ctx := context.Background()
	car1 := cars.NewCar(ctx).SetManufacturer(manufacturers.Toyota).SetYear(2010)
	car2 := cars.NewCar(ctx).SetManufacturer(manufacturers.Tesla).SetYear(2023)
	car3 := cars.NewCar(ctx).SetManufacturer(manufacturers.Ford).SetYear(2020)
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		a, err := NewArray(&buf)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Write(ctx, car1)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Write(ctx, car2)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Write(ctx, car3)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArrayReset(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	var buf1, buf2 bytes.Buffer

	a, err := NewArray(&buf1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			buf1.Reset()
			a.Reset(&buf1)
		} else {
			buf2.Reset()
			a.Reset(&buf2)
		}
		err = a.Write(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		err = a.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Comparison benchmarks between bytes vs writer

func BenchmarkMarshalBytesVsWriter(b *testing.B) {
	ctx := context.Background()
	vehicle := createFullVehicle(ctx)

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf, err := Marshal(ctx, vehicle)
			if err != nil {
				b.Fatal(err)
			}
			buf.Release(ctx)
		}
	})

	b.Run("MarshalWriter", func(b *testing.B) {
		var buf bytes.Buffer
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			err := MarshalWriter(ctx, vehicle, &buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUnmarshalBytesVsReader(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	jsonData := make([]byte, buf.Len())
	copy(jsonData, buf.Bytes())
	jsonStr := buf.String()
	buf.Release(ctx)

	b.Run("Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			restored := cars.NewCar(ctx)
			err := Unmarshal(ctx, jsonData, &restored)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("UnmarshalReader", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			restored := cars.NewCar(ctx)
			err := UnmarshalReader(ctx, strings.NewReader(jsonStr), &restored)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
