package idl

import (
	"context"
	"testing"

	"github.com/johnsiilver/halfpike"
)

func TestFile(t *testing.T) {
	content := `
// A comment
// About something
package hello // Yeah I can comment here

// Okay, love the version
version 0 // And here too

Enum Cars uint8 {
	Unknown @0 // [jsonName(unknown)]
	Toyota @1
	Ford @2
	Tesla @3 // Fuck Elon
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
}
