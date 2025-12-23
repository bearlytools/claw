package idl

import (
	"github.com/gostdlib/base/context"
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

func TestStructOptions(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantErr    bool
		wantOption string
	}{
		{
			name: "Success: struct without options",
			content: `
package test
Struct Car {
	Name string @0
}
`,
			wantErr: false,
		},
		{
			name: "Success: struct with NoPatch option",
			content: `
package test
Struct Patch [NoPatch()] {
	Ops bytes @0
}
`,
			wantErr:    false,
			wantOption: "NoPatch",
		},
		{
			name: "Error: invalid struct option",
			content: `
package test
Struct Car [InvalidOption()] {
	Name string @0
}
`,
			wantErr: true,
		},
		{
			name: "Error: NoPatch with arguments",
			content: `
package test
Struct Car [NoPatch("arg")] {
	Name string @0
}
`,
			wantErr: true,
		},
	}

	for _, test := range tests {
		f := New()
		err := halfpike.Parse(context.Background(), test.content, f)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestStructOptions(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestStructOptions(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if test.wantOption != "" {
			structs := f.Structs()
			if len(structs) == 0 {
				t.Errorf("TestStructOptions(%s): expected at least one struct", test.name)
				continue
			}
			if !structs[0].HasOption(test.wantOption) {
				t.Errorf("TestStructOptions(%s): struct does not have expected option %q", test.name, test.wantOption)
			}
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
