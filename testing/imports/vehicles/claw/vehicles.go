// DO NOT EDIT
// This package is autogenerated and should not be modified except by the clawc compiler.

// Package vehicles 
package vehicles

import (
    "github.com/bearlytools/claw/languages/go/mapping"
    "github.com/bearlytools/claw/languages/go/reflect"
    "github.com/bearlytools/claw/languages/go/reflect/runtime"
    "github.com/bearlytools/claw/languages/go/structs"
    "github.com/bearlytools/claw/languages/go/field"
    
    "github.com/bearlytools/test_claw_imports/cars/claw"
    "github.com/bearlytools/test_claw_imports/trucks"
    "github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
)

// SyntaxVersion is the major version of the Claw language that is being rendered.
const SyntaxVersion = 0

var _package = "vehicles"
var _packagePath = "github.com/bearlytools/claw/testing/imports/vehicles/claw"


type Type uint8

// String implements fmt.Stringer.
func (x Type) String() string {
    return TypeByValue[uint8(x)]
}

// XXXEnumGroup will return the EnumGroup descriptor for this group of enumerators.
// This should only be used by the reflect package and is has no compatibility promises 
// like all XXX fields.
func (x Type) XXXEnumGroup() reflect.EnumGroup {
    return XXXEnumGroups.Get(0)
}

// XXXEnumGroup will return the EnumValueDescr descriptor for an enumerated value.
// This should only be used by the reflect package and is has no compatibility promises 
// like all XXX fields.
func (x Type) XXXEnumValueDescr() reflect.EnumValueDescr {
    return XXXEnumGroups.Get(0).ByValue(uint16(x))
}


const (
    Unknown Type = 0
    Car Type = 1
    Truck Type = 2
)

var TypeByName = map[string]Type{
    "Car": 1,
    "Truck": 2,
    "Unknown": 0,
}

var TypeByValue = map[uint8 ]string{
    0: "Unknown",
    1: "Car",
    2: "Truck",
} 



type Vehicle struct {
   s *structs.Struct
}

// NewVehicle creates a new instance of Vehicle.
func NewVehicle() Vehicle {
    s := structs.New(0, XXXMappingVehicle)
    s.XXXSetNoZeroTypeCompression()
    return Vehicle{
        s: s,
    }
}

// XXXNewFrom creates a new Vehicle from our internal Struct representation.
// As with all things marked XXX*, this should not be used and has not compatibility
// guarantees.
//
// Deprecated: This is not actually deprecated, but it should not be used directly nor
// show up in any documentation.
func XXXNewFrom(s *structs.Struct) Vehicle {
    return Vehicle{s: s}
} 

func (x Vehicle) Type() Type {
    return Type(structs.MustGetNumber[uint8](x.s, 0))
}

func (x Vehicle) SetType(value Type) {
    structs.MustSetNumber(x.s, 0, uint8(value))
} 

func (x Vehicle) Car() cars.Car {
    s := structs.MustGetStruct(x.s, 1)
    return cars.XXXNewFrom(s)
}

func (x Vehicle) SetCar(value cars.Car) {
    structs.MustSetStruct(x.s, 1, value.XXXGetStruct())
} 

func (x Vehicle) Truck() []trucks.Truck {
    l := structs.MustGetListStruct(x.s, 2)
    vals := make([]trucks.Truck, l.Len())

    for i := range vals {
        vals[i] = trucks.XXXNewFrom(l.Get(i))
    }
    return vals
}

func (x Vehicle) AppendTruck(values ...trucks.Truck) {
    vals := make([]*structs.Struct, len(values))
    for i, val := range values {
        vals[i] = val.XXXGetStruct()
    }
    structs.MustAppendListStruct(x.s, 2, vals...)
}
  

// ClawStruct returns a reflection type representing the Struct.
func (x Vehicle) ClawStruct() reflect.Struct{
   return reflect.XXXNewStruct(x.s)
}

// XXXGetStruct returns the internal Struct representation. Like all XXX* types/methods,
// this should not be used and has no compatibility guarantees.
//
// Deprecated: Not deprectated, but should not be used and should not show up in documentation.
func (x Vehicle) XXXGetStruct() *structs.Struct {
    return x.s
}

// XXXDescr returns the Struct's descriptor. This should only be used
// by the reflect package and is has no compatibility promises like all XXX fields.
//
// Deprecated: No deprecated, but shouldn't be used directly or show up in documentation.
func (x Vehicle) XXXDescr() reflect.StructDescr {
    return XXXPackageDescr.Structs().Get(0)
} 

// Everything below this line is internal details.
// Deprecated: Not deprecated, but shouldn't be used directly or show up in documentation.
var XXXMappingVehicle = &mapping.Map{
    Name: "Vehicle",
    Pkg: "vehicles",
    Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw",
    Fields: []*mapping.FieldDescr{
        {
            Name: "Type",
            Type: field.FTUint8,
            Package: "vehicles",
            FullPath: "github.com/bearlytools/claw/testing/imports/vehicles/claw",
            IsEnum: true,
            EnumGroup: "Type",
            FieldNum: 0,
        },
        {
            Name: "Car",
            Type: field.FTStruct,
            Package: "cars",
            FullPath: "github.com/bearlytools/test_claw_imports/cars/claw",
            IsEnum: false,
            FieldNum: 1,
        },
        {
            Name: "Truck",
            Type: field.FTListStructs,
            Package: "trucks",
            FullPath: "github.com/bearlytools/test_claw_imports/trucks",
            IsEnum: false,
            FieldNum: 2,
            
            Mapping: trucks.XXXMappingTruck,
        },
    },
}

// Deprecated: Not deprecated, but shouldn't be used directly or show up in documentation.
var XXXEnumGroups reflect.EnumGroups = reflect.XXXEnumGroupsImpl{
    List:   []reflect.EnumGroup{
        reflect.XXXEnumGroupImpl{
            GroupName: "Type",
            GroupLen: 3,
            EnumSize: 8,
            Descrs: []reflect.EnumValueDescr{
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Unknown",
                    EnumNumber: 0,
                },
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Car",
                    EnumNumber: 1,
                },
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Truck",
                    EnumNumber: 2,
                },
            },
        },  
    },
    Lookup: map[string]reflect.EnumGroup{},
}

func init() {
    x := XXXEnumGroups.(reflect.XXXEnumGroupsImpl)
    for _, g := range x.List {
        x.Lookup[g.Name()] = g
    }
}  

// Deprecated: No deprecated, but shouldn't be used directly or show up in documentation.
var XXXPackageDescr reflect.PackageDescr = reflect.XXXPackageDescrImpl{
    Name: "vehicles",
    Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw",
    ImportDescrs: []reflect.PackageDescr {
        cars.XXXPackageDescr,
        manufacturers.XXXPackageDescr,
        trucks.XXXPackageDescr,  
    }, 
    EnumGroupsDescrs: XXXEnumGroups, 
    StructsDescrs: reflect.XXXStructDescrsImpl{
        Descrs: []reflect.StructDescr{
            reflect.XXXStructDescrImpl{
                Name: "Vehicle",
                Pkg: "vehicles",
                Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw",
                FieldList: []reflect.FieldDescr{
                    reflect.XXXFieldDescrImpl{
                        FD: XXXMappingVehicle.ByName("Type"),
                        EG: XXXEnumGroups.ByName("Type"),
                    },
                    reflect.XXXFieldDescrImpl{
                        FD: XXXMappingVehicle.ByName("Car"),
                    },
                    reflect.XXXFieldDescrImpl{
                        FD: XXXMappingVehicle.ByName("Truck"),
                    },
                },
            }, 
        },
    },  
}

// PackageDescr returns a PackageDescr for this package.
func PackageDescr() reflect.PackageDescr {
    return XXXPackageDescr
}

func init() {
    runtime.RegisterPackage(XXXPackageDescr)
}