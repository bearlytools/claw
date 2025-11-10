package idl

import (
	"context"
	"log"
	"testing"

	"github.com/johnsiilver/halfpike"
	"github.com/kylelemons/godebug/pretty"
)

func TestFile(t *testing.T) {
	content := `
// A comment
// About something
package hello // Yeah I can comment here

// Okay, love the version
version 0 // And here too

// Comment.
options [ NoZeroValueCompression() ]// Comment

import (
	"github.com/johnsiilver/something"
	renamed "github.com/r/something" // Yeah, yeah
)

Enum Maker uint8 {
	Unknown @0 // [jsonName(unknown)]
	Toyota @1
	Ford @2
	Tesla @3 // Comment
}

Struct Car {
	Name string @0
	Maker Maker @1 //Comment
	Year uint16 @2
	Serial uint64 @3
	PreviousVersions []Car @5
	Image bytes @4
}
`
	wantOpts := map[string]Option{
		"NoZeroValueCompression": {"NoZeroValueCompression", nil},
	}

	f := New()

	if err := halfpike.Parse(context.Background(), content, f); err != nil {
		panic(err)
	}

	if f.Package != "hello" {
		panic("package")
	}
	if f.Version != 0 {
		panic("version")
	}
	if diff := pretty.Compare(wantOpts, f.Options); diff != "" {
		t.Fatalf("TestFile(options) -want/+got:\n%s", diff)
	}

	for _, impName := range []string{"something", "renamed"} {
		if _, ok := f.Imports.Imports[impName]; !ok {
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

// lineLexer is provided to simply lex out a single line for testing.
type lineLexer struct {
	line halfpike.Line
}

func (l *lineLexer) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	l.line = p.Next()
	return nil
}
func (p *lineLexer) Validate() error {
	return nil
}
