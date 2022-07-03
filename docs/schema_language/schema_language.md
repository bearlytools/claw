# Schema Language

Claw is not completely self describing, in that you can decode any message without having
an IDL file, but you will not have field names associated with the fields.

Claw is strongly typed in the same vein as the Go language is.

It's basic syntax is a mixed bag of concepts I've taken from:
* The Go language specification
* Capt'n Proto language specification
* Protocol Buffer language specification

Here is an example schema file:

```claw
// Below is the name of this package defined using the keyword "package"
package cars // This must be the first line

// This details the version of the language to use. This defaults to 0
// and is not required to put in the file unless specifying another version.
version 0

// These are other .claw packages you want to import, following the same
// import rules that Go uses. In Claw, imports have to be defined in blocks
// like below, even for single entries.
import (
	"github.com/johnsiilver/cars/vins"
)

// We support enumerators that are either based on uint8 or uint16. You must
// always have a 0 Enum. Unlike other IDLs, Enums are global and cannot be put
// inside a Struct. 
enum Maker uint8 {
	Unknown @0 [jsonName(unknown)] // This show using an option to rename this on JSON export
	Toyota @1
	Ford @2
	Tesla @10
}

// Struct is our message type.
struct Car {
	Name string @0
	Maker Maker @1
	Year uint16 @2
	Vin vins.Number @4 // Using an type from an external package
	PreviousVersions []Car @3 // Self referential list type
	Image bytes @5
}
```
A few notes:

* Imports must be used
* Imports cannot have circular dependencies, aka package A cannot import B if B imports A
* All entries must exist on a single line (we don't support wrapped lines)
* Everthing is case sensitive
* The package name must start with a lowercase letter, it can contain _, but we prefer it doesn't
* All fields must start with a capital letter and cannot use - or _, we prefer camel case
* Same for Enum values
* Types are lower case
* Types are order dependent. If Struct A refers to Struct B, B must be ahead of it in the file
* Only 1 .claw file per directory, must have the name of the directory, which must be the same as the package name

## Comments

Comments use the C style of `//`.  The comments can stand alone on a line or be used at the end of a line.

Comments are ignored by the compiler and we pretend they don't exist when talking about requirements for the rest of the document.

## Package declaration

All Claw files must contain a package declaration as their first line in the file. This package name is used with the location in the filesystem or version control to defines the import name. A package that imports another package refers to it by only the package name. The package name must be the same as the file name, which must be the same as the directory name.

Package names must start with a lower case character and may contain letters, numbers or _. It is preferred to avoid _ in the name if possible.

An example package declaration:

```claw
package vins
```

## Version declaration

Currently there is only one version of the Schema language, version 0. It is not required to declare the version number, as it will default to 0. 

However, this is silenty declared after the package name if not explicitly declared:

```claw
Version 0
```

The version must be declared after the package declaration.

## Options declaration

Options are declared using the options block statement, which consists of `options (` on one line, option statements on the following lines, and a closing `)` on its own line.

Each option uses the option syntax, which is `<name>(<args>, <separated>, <by>, <commas>)` and is declared on its own line.

Options are optional and do not have to be declared.  They must come after package and version, but before imports.

## Imports

Imports are declared using the import block statement, which consists of `import (` on one line, imports statements on the following lines, and a closing `)` on its own line.

Imports must be defined before any Struct or Enum and after package, version and options.

Each import line declares either the location of the other package from a file directory root or the full path via version control (or proxy). 

A sample import would be:

```claw
import (
    "github.com/johnsiilver/cars/vins"
    "github.com/johnsiilver/cars/models"
)
```

Since Claw uses only the base name (vins and models in this example) to reference what types from those files, name collisions can occur. To avoid this, we borrow Go's rename syntax to allow us to reference types without colliding. Here is an example:

```claw

```claw
import (
    "github.com/johnsiilver/bicycles/models"
    carModels "github.com/johnsiilver/cars/models"
)
```

Now we can reference `.../cars/models` using the word `carModels`, avoiding the namespace collision. Renames must start with a lower case letter and may contain a mix of letters and numbers. It is preferred to use camelCase.

## Built-in Types
The following basic field types are defined:

* Boolean: `bool`
* Signed integers: `int8`, `int16`, `int32`, `int64`
* Unsigned integers: `uint8`, `uint16`, `uint32`, `uint64`
* Floating points: `float32`, `float64`
* Blobs: `string`, `bytes`
* Lists: `[]Type`

Notes:

* String is always UTF-8 encoded
* Bytes is an array of byte. String is the same as Bytes on the wire, with the only difference being how the generated package deals with it

## Structs 

A Struct is a set of named typed fields that are numbered consecutively starting from 0.

```claw
Struct Car {
	Name string @0
	Maker Maker @1
}
```

Each field can also have options, which are declared after `@n` and are contained within a `[]` block, with commas between entries.

An example would be:

```claw
Struct Car {
    Name string @0 [jsonRename("name")]
}
```

## Enums

An Enum provides a set of symbolic values that translate to a number. Claw allows the numbers to be uint8 or uint16 in size. Enums must start at 0, but may represent any positive value that can be covered.

```claw
Enum CarMake {
    Unknown @0
    Toyota @1
    Tesla @20
}
```

The same as Struct fields, an enum entry can have a list of options.

## Changing the definitions

You can change the definitions in a Claw file in the following ways without breaking the wire format:

* New Structs or Enums
* New fields
* Order of fields (but an existing field CANNOT be renumbered)
* Fields can be renamed, which will not change anything on the wire, but will cause existing code that depended on the name to break

Any change not listed above should be considered breaking, especially:

* You cannot change a field number
* You cannot change a field type

This only applies to the Claw native format. Exporting to any other format can cause breaking changes (such as JSON).

### Note on this file

I liked the form of Capn proto's schema definition file, so this mirrors that format.