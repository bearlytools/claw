package idl

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/johnsiilver/halfpike"
)

/*
package {{package name}}
version {{Integer}}
options {{ [ {{option string}}, {{option string}} ] }}

import (
	"github.com/some/location"
)

Enum {{String}} {{uint8|uint16}} {
	{{Name}} @{{Number}}
}

Struct {{String}} {
	{{Name}} {{Type}} @{{Integer}}
}

*/

// FileOption is an option for the file.
type FileOption int

type File struct {
	Package    string
	Version    int
	Options    []FileOption
	Identifers map[string]any
	Imports    map[string]Import
}

func New() *File {
	return &File{
		Identifers: map[string]any{},
		//hp: halfpike.NewParser(input, val Validator) (*Parser, error)
	}
}

func (f *File) Validate() error {
	return nil
}

// Start is the start point for reading the IDL.
func (f *File) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return f.ParsePackage
}

func (f *File) SkipLinesWithComments(p *halfpike.Parser) {
	l := p.Next()

	if strings.HasPrefix("//", l.Items[0].Val) {
		if p.EOF(l) {
			return
		}
		f.SkipLinesWithComments(p)
	} else {
		p.Backup()
	}
}

var (
	underscore = '_'
)

// ParseVersion finds the version
func (f *File) ParsePackage(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	f.SkipLinesWithComments(p)

	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'package {{package name}}'", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("package", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	if err := validPackage(line.Items[1].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}
	f.Package = line.Items[1].Val

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf(err.Error())
	}

	return f.ParseVersion
}

// ParseVersion finds the version
func (f *File) ParseVersion(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	f.SkipLinesWithComments(p)

	line := p.Next()

	if len(line.Items) < 3 {
		return p.Errorf("[Line %d] error: got %q, want: 'version {{Integer}}'", line.LineNum, line.Raw)
	}

	if err := caseSensitiveCheck("version", line.Items[0].Val); err != nil {
		return p.Errorf("[Line %d] error: %w", line.LineNum, err)
	}

	var err error
	f.Version, err = line.Items[1].ToInt()
	if err != nil {
		return p.Errorf("[Line %d] error: got: %q, want: 'version {{Integer}}'", line.LineNum, line.Raw)
	}

	if err := commentOrEOL(line, 2); err != nil {
		return p.Errorf(err.Error())
	}

	return f.FindNext
}

func (f *File) FindNext(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	f.SkipLinesWithComments(p)

	line := p.Next()

	switch line.Items[0].Val {
	case "options":
		if f.Options != nil {
			return p.Errorf("[Line %d] error: duplicate 'options' line found", line.LineNum)
		}
		if len(f.Identifers) != 0 {
			return p.Errorf("[Line %d] 'options' must come before any Structs or Enums", line.LineNum)
		}
		p.Backup()
		return f.ParseOptions(ctx, p)
	case "import":
		if f.Imports != nil {
			return p.Errorf("[Line %d] error: duplicate 'import' line found", line.LineNum)
		}
		if len(f.Identifers) != 0 {
			return p.Errorf("[Line %d] 'import' must come before any Structs or Enums", line.LineNum)
		}
		p.Backup()
		panic("not supported")
	case "Enum":
		p.Backup()
		e := NewEnum()
		if err := e.parse(p); err != nil {
			return p.Errorf(err.Error())
		}
		if _, ok := f.Identifers[e.Name]; ok {
			return p.Errorf("Error: found two top level identifiers named %q", e.Name)
		}
		f.Identifers[e.Name] = e
		return f.FindNext
	case "Struct":
		p.Backup()
		return f.ParseStruct(ctx, p)
	default:
		if p.EOF(line) {
			return nil
		}
		return p.Errorf("[Line %d] do not understand this line", line.LineNum)
	}
	return nil
}

func (f *File) ParseOptions(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return nil
}

func (f *File) ParseEnum(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {

	return nil
}

func (f *File) ParseStruct(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
	return nil
}

func caseSensitiveCheck(want string, item string) error {
	if item != want {
		if strings.EqualFold(item, want) {
			return fmt.Errorf("%q keyword found, but it is required to be %q", item, want)
		}
		return fmt.Errorf("got: %q, want: %q", item, want)
	}
	return nil
}

func commentOrEOL(line halfpike.Line, from int) error {
	if strings.HasPrefix(line.Items[from].Val, "//") {
		return nil
	}

	if len(line.Items[from:]) > 1 {
		return fmt.Errorf("got item %q after %q, which was unexpected", halfpike.ItemJoin(line, from, len(line.Items)), halfpike.ItemJoin(line, 0, from))
	}

	return nil
}

func validPackage(pkgName string) error {
	runes := []rune(pkgName)
	if unicode.IsUpper(runes[0]) {
		return fmt.Errorf("package name cannot start with an uppercase letter")
	}
	if !unicode.IsLetter(runes[0]) {
		return fmt.Errorf("package name must start with a letter")
	}
	for _, r := range runes[1:] {
		if unicode.IsLetter(r) {
			continue
		}
		if unicode.IsNumber(r) {
			continue
		}
		if r == underscore {
			continue
		}
		return fmt.Errorf("package name contains character %v which is invalid for a package name", r)
	}
	return nil
}

func validateIdent(ident string) error {
	runes := []rune(ident)
	if unicode.IsLower(runes[0]) {
		return fmt.Errorf("identifier cannot start with an lowercase letter")
	}

	if !unicode.IsLetter(runes[0]) {
		return fmt.Errorf("identifier must start with a letter")
	}

	for _, r := range runes[1:] {
		if unicode.IsLetter(r) {
			continue
		}
		if unicode.IsNumber(r) {
			continue
		}
		return fmt.Errorf("identifier contains character %v which is invalid for an identifer", r)
	}
	return nil
}

type Struct struct {
}

type Enum struct {
	Name   string
	Size   int
	Names  map[string]uint16
	Values map[uint16]string
}

func NewEnum() Enum {
	return Enum{
		Names:  map[string]uint16{},
		Values: map[uint16]string{},
	}
}

type EnumVal struct {
}

func (e *Enum) parse(p *halfpike.Parser) error {
	l := p.Next()
	if len(l.Items) < 5 {
		return fmt.Errorf("[Line %d]: error: Enum line has incorrect format", l.LineNum)
	}

	if err := validateIdent(l.Items[1].Val); err != nil {
		return fmt.Errorf("[Line %d]: error: Enum identifier: %w", l.LineNum, err)
	}

	e.Name = l.Items[1].Val

	switch l.Items[2].Val {
	case "uint8":
		e.Size = 8
	case "uint16":
		e.Size = 16
	default:
		return fmt.Errorf("[Line %d]: error: expected keyword 'uint8' or 'uint16, got %q", l.LineNum, l.Items[2].Val)
	}

	if l.Items[3].Val != "{" {
		return fmt.Errorf("[Line %d]: error: expected '{' at the end of the line, got %q", l.LineNum, l.Items[3].Val)
	}

	if err := commentOrEOL(l, 4); err != nil {
		return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
	}

	for {
		l = p.Next()
		if p.EOF(l) {
			return fmt.Errorf("[Line %d]: Malformed Enum, EOF reached before closing '}'", l.LineNum)
		}
		if l.Items[0].Val == "}" {
			break
		}
		if len(l.Items) < 3 {
			return fmt.Errorf("[Line %d]: Malformed Enum entry", l.LineNum)
		}
		if err := validateIdent(l.Items[0].Val); err != nil {
			return fmt.Errorf("[Line %d]: error: Enum identifier: %w", l.LineNum, err)
		}
		if _, ok := e.Names[l.Items[0].Val]; ok {
			return fmt.Errorf("[Line %d]: error: Enum %q already contains enumerator %q", l.LineNum, e.Name, l.Items[0].Val)
		}
		if !strings.HasPrefix(l.Items[1].Val, "@") {
			return fmt.Errorf("[Line %d]: error: expected @{{Number}} after identifier, got %q", l.LineNum, l.Items[1].Val)
		}
		numString := strings.Split(l.Items[1].Val, "@")[1]
		n, err := strconv.Atoi(numString)
		if err != nil {
			return fmt.Errorf("[Line %d]: error: expected @{{Number}} after identifier, got %q", l.LineNum, l.Items[1].Val)
		}
		if n < 0 {
			return fmt.Errorf("[Line %d]: error: cannot have an enumerated value < 0", l.LineNum)
		}
		if _, ok := e.Values[uint16(n)]; ok {
			return fmt.Errorf("[Line %d]: error: Enum %q already contains enumerator(%s) with value %d", l.LineNum, e.Name, e.Values[uint16(n)], n)
		}
		e.Names[l.Items[0].Val] = uint16(n)
		e.Values[uint16(n)] = l.Items[0].Val

		if err := commentOrEOL(l, 2); err != nil {
			return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
		}
	}
	// We are on the line with }.
	if len(e.Names) == 0 {
		return fmt.Errorf("Enum %q has no entries, which is not valid", e.Name)
	}
	if err := commentOrEOL(l, 1); err != nil {
		return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
	}
	return nil
}

type Import struct{}
