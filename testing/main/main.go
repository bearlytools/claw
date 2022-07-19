package main

import (
	"fmt"

	hello "github.com/bearlytools/claw/testing"
)

func main() {
	venza2021 := hello.NewCar()

	venza2021.SetMaker(hello.Toyota)
	venza2021.SetName("Venza")
	venza2021.SetYear(2021)
	venza2021.SetSerial(12345)

	// Older models
	venza2020 := hello.NewCar()
	venza2020.SetMaker(hello.Toyota)
	venza2020.SetName("Venza")
	venza2020.SetYear(2020)
	venza2020.SetSerial(1234)

	venza2019 := hello.NewCar()
	venza2019.SetMaker(hello.Toyota)
	venza2019.SetName("Venza")
	venza2019.SetYear(2019)
	venza2019.SetSerial(123)

	venza2021.AppendPreviousVersions(venza2020, venza2019)

	fmt.Println("Name: ", venza2021.Name())
	fmt.Println("Maker: ", venza2021.Maker())
	fmt.Println("Year: ", venza2021.Year())
	fmt.Println("Serial: ", venza2021.Serial())
	fmt.Println("Previous models: ")
	for i, car := range venza2021.PreviousVersions() {
		if i > 0 {
			fmt.Println()
		}
		fmt.Println("\tName: ", car.Name())
		fmt.Println("\tMaker: ", car.Maker())
		fmt.Println("\tYear: ", car.Year())
		fmt.Println("\tSerial: ", car.Serial())
	}

}
