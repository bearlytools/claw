package json

import (
	"bytes"
	"log"
	"testing"

	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
	cars "github.com/bearlytools/test_claw_imports/cars/claw"
)

func TestArrayWrite(t *testing.T) {
	buff := bytes.Buffer{}
	arr, err := NewArray(Options{UseEnumNumbers: true}, &buff)
	if err != nil {
		panic(err)
	}

	vehicle := vehicles.NewVehicle()
	car := cars.NewCar()
	car.SetModel(cars.ModelS)
	car.SetManufacturer(manufacturers.Tesla)
	vehicle.SetCar(car)

	if err := arr.Write(vehicle); err != nil {
		panic(err)
	}

	vehicle = vehicles.NewVehicle()
	car = cars.NewCar()
	car.SetModel(cars.Venza)
	car.SetManufacturer(manufacturers.Toyota)
	vehicle.SetCar(car)

	if err := arr.Write(vehicle); err != nil {
		panic(err)
	}

	if err := arr.Close(); err != nil {
		panic(err)
	}

	log.Println(buff.String())
}
