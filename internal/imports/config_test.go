package imports

import "github.com/bearlytools/claw/internal/idl"

var wantCars = &idl.File{
	Package:  "cars",
	FullPath: "github.com/bearlytools/test_claw_imports/cars/claw",
	Version:  0,
	Options:  map[string]idl.Option{},
	Identifers: map[string]interface{}{
		"Car": idl.Struct{
			Name: "Car",
			Fields: []idl.StructField{
				{
					Name:            "Manufacturer",
					Index:           0,
					Type:            0,
					IsEnum:          false,
					IdentName:       "manufacturers.Manufacturer",
					SelfReferential: false,
				},
				{
					Name:            "Model",
					Index:           1,
					Type:            6,
					IsEnum:          true,
					IdentName:       "Model",
					SelfReferential: false,
				},
				{
					Name:            "Year",
					Index:           2,
					Type:            7,
					IsEnum:          false,
					IdentName:       "",
					SelfReferential: false,
				},
			},
		},
		"Model": idl.Enum{
			Name: "Model",
			Size: 8,
		},
	},
	External: map[string]*idl.File{
		"manufacturers.Manufacturer": wantManufacturer,
	},
	Imports: idl.Import{
		Imports: map[string]idl.ImportEntry{
			"manufacturers": {
				Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers",
				Name: "manufacturers",
			},
		},
	},
	RepoVersion: "",
	SHA256:      "5dd9820b64c83de01acab412fb4b1f002b724cc61b2c5d1c5d392931384a5dd8",
}

var wantManufacturer = &idl.File{
	Package:  "manufacturers",
	FullPath: "github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers",
	Version:  0,
	Options:  map[string]idl.Option{},
	Identifers: map[string]interface{}{
		"Manufacturer": idl.Enum{
			Name: "Manufacturer",
			Size: 8,
		},
	},
	External: map[string]*idl.File{},
	Imports: idl.Import{
		Imports: map[string]idl.ImportEntry(nil),
	},
	RepoVersion: "",
	SHA256:      "",
}

var wantTrucks = &idl.File{
	Package:  "trucks",
	FullPath: "github.com/bearlytools/test_claw_imports/trucks",
	Version:  0,
	Options:  map[string]idl.Option{},
	Identifers: map[string]interface{}{
		"Model": idl.Enum{
			Name: "Model",
			Size: 8,
		},
		"Truck": idl.Struct{
			Name: "Truck",
			Fields: []idl.StructField{
				{
					Name:            "Manufacturer",
					Index:           0,
					Type:            0,
					IsEnum:          false,
					IdentName:       "manufacturers.Manufacturer",
					SelfReferential: false,
				},
				{
					Name:            "Model",
					Index:           1,
					Type:            6,
					IsEnum:          true,
					IdentName:       "Model",
					SelfReferential: false,
				},
				{
					Name:            "Year",
					Index:           2,
					Type:            7,
					IsEnum:          false,
					IdentName:       "",
					SelfReferential: false,
				},
			},
		},
	},
	External: map[string]*idl.File{
		"manufacturers.Manufacturer": wantManufacturer,
	},
	Imports: idl.Import{
		Imports: map[string]idl.ImportEntry{
			"manufacturers": {
				Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers",
				Name: "manufacturers",
			},
		},
	},
	RepoVersion: "",
	SHA256:      "46463a0f40b2881faedbf4cc3fda56ac8f2df161aa7046ec19110587b7140720",
}

var wantRoot = &idl.File{
	Package:  "vehicles",
	FullPath: "github.com/bearlyworks/claw/testing/imports/vehicles/claw",
	Version:  0,
	Options:  map[string]idl.Option{},
	Identifers: map[string]interface{}{
		"Type": idl.Enum{
			Name: "Type",
			Size: 8,
		},
		"Vehicle": idl.Struct{
			Name: "Vehicle",
			Fields: []idl.StructField{
				{
					Name:            "Type",
					Index:           0,
					Type:            6,
					IsEnum:          true,
					IdentName:       "Type",
					SelfReferential: false,
				},
				{
					Name:            "Car",
					Index:           1,
					Type:            0,
					IsEnum:          false,
					IdentName:       "cars.Car",
					SelfReferential: false,
				},
				{
					Name:            "Truck",
					Index:           2,
					Type:            0,
					IsEnum:          false,
					IdentName:       "trucks.Truck",
					SelfReferential: false,
				},
			},
		},
	},
	External: map[string]*idl.File{
		"cars.Car":     wantCars,
		"trucks.Truck": wantTrucks,
	},
	Imports: idl.Import{
		Imports: map[string]idl.ImportEntry{
			"cars": {
				Path: "github.com/bearlytools/test_claw_imports/cars/claw",
				Name: "cars",
			},
			"manufacturers": {
				Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers",
				Name: "manufacturers",
			},
			"trucks": {
				Path: "github.com/bearlytools/test_claw_imports/trucks",
				Name: "trucks",
			},
		},
	},
	RepoVersion: "",
	SHA256:      "",
}

var wantConfig = &Config{
	Root: wantRoot,
	Imports: map[string]*idl.File{
		"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers": wantManufacturer,
		"github.com/bearlytools/test_claw_imports/cars/claw":                      wantCars,
		"github.com/bearlytools/test_claw_imports/trucks":                         wantTrucks,
		"github.com/bearlyworks/claw/testing/imports/vehicles/claw":               wantRoot,
	},
	Module: &Module{
		Path:     "github.com/bearlyworks/claw/testing/imports/vehicles/claw",
		Required: nil,
		Replace:  nil,
		ACLs:     nil,
	},
	LocalReplace: LocalReplace{
		Replace: nil,
	},
	GlobalReplace: map[string]Replace{},
}
