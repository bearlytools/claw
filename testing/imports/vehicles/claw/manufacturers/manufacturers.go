// DO NOT EDIT
// This package is autogenerated and should not be modified except by the clawc compiler.

// Package manufacturers 
package manufacturers

import (
    "github.com/bearlytools/claw/languages/go/reflect"
    "github.com/bearlytools/claw/languages/go/reflect/runtime"
    
)

// SyntaxVersion is the major version of the Claw language that is being rendered.
const SyntaxVersion = 0


type Manufacturer uint8

// String implements fmt.Stringer.
func (x Manufacturer) String() string {
    return ManufacturerByValue[uint8(x)]
}

// XXXEnumGroup will return the EnumGroup descriptor for this group of enumerators.
// This should only be used by the reflect package and is has no compatibility promises 
// like all XXX fields.
func (x Manufacturer) XXXEnumGroup() reflect.EnumGroup {
    return XXXEnumGroups.Get(0)
}

// XXXEnumGroup will return the EnumValueDescr descriptor for an enumerated value.
// This should only be used by the reflect package and is has no compatibility promises 
// like all XXX fields.
func (x Manufacturer) XXXEnumValueDescr() reflect.EnumValueDescr {
    return XXXEnumGroups.Get(0).ByValue(int(x))
}


const (
    Unknown Manufacturer = 0
    Toyota Manufacturer = 1
    Ford Manufacturer = 2
    Tesla Manufacturer = 3
)

var ManufacturerByName = map[string]Manufacturer{
    "Ford": 2,
    "Tesla": 3,
    "Toyota": 1,
    "Unknown": 0,
}

var ManufacturerByValue = map[uint8 ]string{
    0: "Unknown",
    1: "Toyota",
    2: "Ford",
    3: "Tesla",
} 
 

// Everything below this line is internal details.
var _package = "manufacturers"

var XXXEnumGroups reflect.EnumGroups = reflect.XXXEnumGroupsImpl{
    List:   []reflect.EnumGroup{
        reflect.XXXEnumGroupImpl{
            GroupName: "Manufacturer",
            GroupLen: 4,
            EnumSize: 8,
            Descrs: []reflect.EnumValueDescr{
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Unknown",
                    EnumNumber: 0,
                },
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Toyota",
                    EnumNumber: 1,
                },
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Ford",
                    EnumNumber: 2,
                },
                reflect.XXXEnumValueDescrImpl{
                    EnumName: "Tesla",
                    EnumNumber: 3,
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

var XXXPackageDescr reflect.PackageDescr = reflect.XXXPackageDescrImpl{
    Name: "manufacturers",
    Path: "github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers", 
    EnumGroupsDescrs: XXXEnumGroups, 
    StructsDescrs: []reflect.StructDescr{  
    },  
}

func init() {
    runtime.RegisterPackage(XXXPackageDescr)
}