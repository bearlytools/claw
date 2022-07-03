package idl

import (
	"context"
	"log"
	"testing"

	"github.com/johnsiilver/halfpike"
)

func TestFile(t *testing.T) {
	content := `
// A comment
// About something
package hello // Yeah I can comment here

// Okay, love the version
version 1 // And here too

import (
	"github.com/johnsiilver/something"
	renamed "github.com/r/something" // Yeah, yeah
)

Enum Maker uint8 {
	Unknown @0 // [jsonName(unknown)]
	Toyota @1
	Ford @2
	Tesla @3 // Fuck Elon
}

Struct Car {
	Name string @0
	Maker Maker @1
	Year uint16 @2
	Serial uint64 @3
	PreviousVersions []Car @5
	Image bytes @4
}
`

	f := New()

	p, err := halfpike.NewParser(content, f)
	if err != nil {
		panic(err)
	}
	if err := halfpike.Parse(context.Background(), p, f.Start); err != nil {
		panic(err)
	}

	if f.Package != "hello" {
		panic("package")
	}
	if f.Version != 1 {
		panic("package")
	}

	for _, impName := range []string{"something", "renamed"} {
		if _, ok := f.Imports.imports[impName]; !ok {
			log.Fatalf("can't find import %q", impName)
		}
	}

	for _, enumName := range []string{"Maker"} {
		if _, ok := f.Identifers[enumName]; !ok {
			panic("enums")
		}
	}

	for _, structName := range []string{"Car"} {
		switch f.Identifers[structName].(type) {
		case Struct:
		default:
			panic("structs")
		}
	}
}
