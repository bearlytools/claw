// DO NOT EDIT
// This package is autogenerated and should not be modified except by the clawc compiler.

// Package vehicles 
package vehicles

import (
    "github.com/bearlytools/claw/languages/go/mapping"
    "github.com/bearlytools/claw/languages/go/reflect"
    "github.com/bearlytools/claw/languages/go/reflect/runtime"
    "github.com/bearlytools/claw/languages/go/structs"
    "github.com/bearlytools/claw/languages/go/types/list"
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
// This is a set of all constants representing enumerated values for enum Type.
const (
    Unknown Type = 0
    Car Type = 1
    Truck Type = 2
)

// TypeByName converts a string representing the enumerator into a Type.
var TypeByName = map[string]Type{
    "Car": 1,
    "Truck": 2,
    "Unknown": 0,
}

// TypeByValue converts a uint8 representing a Type into its string name.
var TypeByValue = map[uint8]string{
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

func (x Vehicle) SetType(value Type) Vehicle {
    structs.MustSetNumber(x.s, 0, uint8(value))
    return x
} 

func (x Vehicle) Car() cars.Car {
    s := structs.MustGetStruct(x.s, 1)
    return cars.XXXNewFrom(s)
}

func (x Vehicle) SetCar(value cars.Car) Vehicle {
    structs.MustSetStruct(x.s, 1, value.XXXGetStruct())
    return x
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
  
func (x Vehicle) Types() list.Enums[Type] {
    n := structs.MustGetListNumber[Type](x.s, 3)
    return list.XXXEnumsFromNumbers(n) 
}

func (x Vehicle) SetTypes(value list.Enums[Type]) Vehicle {
    n := value.XXXNumbers()
    structs.MustSetListNumber(x.s, 3, n)
    return x
} 

func (x Vehicle) Bools() list.Bools {
    return list.XXXFromBools(structs.MustGetListBool(x.s, 4))
}

func (x Vehicle) SetBools(value list.Bools) Vehicle {
    structs.MustSetListBool(x.s, 4, value.XXXBools())
    return x
}  

// ClawStruct returns a reflection type representing the Struct.
func (x Vehicle) ClawStruct() reflect.Struct{
    descr := XXXStructDescrVehicle
    return reflect.XXXNewStruct(x.s, descr)
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
            FieldNum: 0,
            IsEnum: true,
            EnumGroup: "Type",
        },
        {
            Name: "Car",
            Type: field.FTStruct,
            Package: "cars",
            FullPath: "github.com/bearlytools/test_claw_imports/cars/claw",
            FieldNum: 1,
            IsEnum: false,
            StructName: "cars.Car",
            
            Mapping: cars.XXXMappingCar,
        },
        {
            Name: "Truck",
            Type: field.FTListStructs,
            Package: "trucks",
            FullPath: "github.com/bearlytools/test_claw_imports/trucks",
            FieldNum: 2,
            IsEnum: false,
        },
        {
            Name: "Types",
            Type: field.FTListUint8,
            Package: "vehicles",
            FullPath: "github.com/bearlytools/claw/testing/imports/vehicles/claw",
            FieldNum: 3,
            IsEnum: true,
            EnumGroup: "Type",
        },
        {
            Name: "Bools",
            Type: field.FTListBools,
            Package: "vehicles",
            FullPath: "github.com/bearlytools/claw/testing/imports/vehicles/claw",
            FieldNum: 4,
            IsEnum: false,
        },
    },
}



var XXXEnumGroupType = reflect.XXXEnumGroupImpl{
    GroupName: "Type",
    GroupLen: 3,
    EnumSize: 8,
    Descrs: []reflect.EnumValueDescr{
        reflect.XXXEnumValueDescrImpl{
            EnumName: "Unknown",
            EnumNumber: 0,
            EnumSize: 8,
        },
        reflect.XXXEnumValueDescrImpl{
            EnumName: "Car",
            EnumNumber: 1,
            EnumSize: 8,
        },
        reflect.XXXEnumValueDescrImpl{
            EnumName: "Truck",
            EnumNumber: 2,
            EnumSize: 8,
        },
    },
}

// Deprecated: Not deprecated, but shouldn't be used directly or show up in documentation.
var XXXEnumGroups reflect.EnumGroups = reflect.XXXEnumGroupsImpl{
    List:   []reflect.EnumGroup{
        XXXEnumGroupType,
    },
    Lookup: map[string]reflect.EnumGroup{
        "Type": XXXEnumGroupType,
    },
} 
var XXXStructDescrVehicle = &reflect.XXXStructDescrImpl{
    Name:      "Vehicle",
    Pkg:       XXXMappingVehicle.Pkg,
    Path:      XXXMappingVehicle.Path,
    Mapping:   XXXMappingVehicle,
    FieldList: []reflect.FieldDescr {
        
        reflect.XXXFieldDescrImpl{
            FD:  XXXMappingVehicle.Fields[0],
            EG: XXXEnumGroupType, 
        }, 
        
        reflect.XXXFieldDescrImpl{
            FD: XXXMappingVehicle.Fields[1],
            SD: cars.XXXStructDescrCar,
        },
         
        
        reflect.XXXFieldDescrImpl{
            FD: XXXMappingVehicle.Fields[2],
            SD: trucks.XXXStructDescrTruck,
        },
         
        
        reflect.XXXFieldDescrImpl{
            FD:  XXXMappingVehicle.Fields[3],
            EG: XXXEnumGroupType, 
        }, 
        
        reflect.XXXFieldDescrImpl{
            FD:  XXXMappingVehicle.Fields[4],  
        },  
    },
}

var XXXStructDescrs = map[string]*reflect.XXXStructDescrImpl{
    "Vehicle":  XXXStructDescrVehicle,
}

// Deprecated: No deprecated, but shouldn't be used directly or show up in documentation.
var XXXPackageDescr reflect.PackageDescr = &reflect.XXXPackageDescrImpl{
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
            XXXStructDescrVehicle,
        },
    },  
}

// PackageDescr returns a PackageDescr for this package.
func PackageDescr() reflect.PackageDescr {
    return XXXPackageDescr
}

// Registers our package description with the runtime.
func init() {
    runtime.RegisterPackage(XXXPackageDescr)
}