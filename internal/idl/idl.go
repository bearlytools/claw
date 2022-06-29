package idl

import (
	"context"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/bearlytools/claw/internal/field"
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
	Imports    Import
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

// Structs returns all Structs that were decoded.
func (f *File) Structs() chan Struct {
	ch := make(chan Struct, 1)

	if f.Identifers == nil {
		close(ch)
		return ch
	}

	go func() {
		defer close(ch)
		for _, i := range f.Identifers {
			switch v := i.(type) {
			case Struct:
				ch <- v
			}
		}
	}()
	return ch
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
		if f.Imports.imports != nil {
			return p.Errorf("[Line %d] error: duplicate 'import' line found", line.LineNum)
		}
		if len(f.Identifers) != 0 {
			return p.Errorf("[Line %d] 'import' must come before any Structs or Enums", line.LineNum)
		}
		p.Backup()
		i := NewImport()
		if err := i.parse(p); err != nil {
			return p.Errorf(err.Error())
		}
		f.Imports = i
		return f.FindNext
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
		s := NewStruct(f)
		if err := s.parse(p); err != nil {
			return p.Errorf(err.Error())
		}
		if _, ok := f.Identifers[s.Name]; ok {
			return p.Errorf("Error: found two top level identifiers named %q", s.Name)
		}
		f.Identifers[s.Name] = s
		return f.FindNext
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

// Import represents an import block.
type Import struct {
	imports map[string]impEntry
}

// impEntry represents an individual import entry.
type impEntry struct {
	Path string
	Name string
}

// NewImport creates a new Import.
func NewImport() Import {
	return Import{imports: map[string]impEntry{}}
}

func (i *Import) parse(p *halfpike.Parser) error {
	l := p.Next()
	if len(l.Items) < 3 {
		return fmt.Errorf("[Line %d] error: got %q, want: 'import (", l.LineNum, l.Raw)
	}
	item := l.Items[1]
	if item.Val != "(" {
		return fmt.Errorf("[Line %d] error: got 'version %s', want: '(", l.LineNum, item.Val)
	}

	if err := commentOrEOL(l, 2); err != nil {
		return err
	}

	if p.EOF(l) {
		return fmt.Errorf("[Line %d] error: EOF reached before close of 'import'", l.LineNum)
	}

	for {
		l := p.Next()
		if p.EOF(l) {
			return fmt.Errorf("[Line %d]: Malformed import, EOF reached before closing '}'", l.LineNum)
		}
		if l.Items[0].Val == ")" {
			if err := commentOrEOL(l, 1); err != nil {
				return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
			}
			break
		}

		lwc := withoutCommentEOL(l)
		switch len(lwc.Items) {
		case 1:
			p, err := validImportPath(l.Items[0].Val)
			if err != nil {
				return fmt.Errorf("[Line %d]: import statement path looks malformed: %q", l.LineNum, l.Items[0].Val)
			}
			imp := impEntry{Path: p, Name: pkgFromImpPath(p)}
			if _, ok := i.imports[imp.Name]; ok {
				return fmt.Errorf("[Line %d]: duplicate import with name %q", l.LineNum, imp.Name)
			}
			i.imports[imp.Name] = imp
			continue
		case 2:
			if err := validPackage(l.Items[0].Val); err != nil {
				return fmt.Errorf("[Line %d]: bad package rename %q: %w", l.LineNum, l.Items[0].Val, err)
			}
			p, err := validImportPath(l.Items[1].Val)
			if err != nil {
				return fmt.Errorf("[Line %d]: import statement path looks malformed: %q", l.LineNum, l.Items[1].Val)
			}
			imp := impEntry{Path: pkgFromImpPath(p), Name: l.Items[0].Val}
			if _, ok := i.imports[imp.Name]; ok {
				return fmt.Errorf("[Line %d]: duplicate import with name %q", l.LineNum, imp.Name)
			}
			i.imports[imp.Name] = imp
			continue
		default:
			return fmt.Errorf("[Line %d]: import statement looks malformed: %q", l.LineNum, l.Raw)
		}

		if err := commentOrEOL(l, 2); err != nil {
			return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
		}
	}
	if len(i.imports) == 0 {
		return fmt.Errorf("empty import block, which is a parse error")
	}
	return nil
}

// Enum is a set of name values that translate to a number.
type Enum struct {
	Name   string
	Size   int
	Names  map[string]EnumVal
	Values map[uint16]EnumVal
}

// New creates a new Enum.
func NewEnum() Enum {
	return Enum{
		Names:  map[string]EnumVal{},
		Values: map[uint16]EnumVal{},
	}
}

// EnumVal is a value stored in an Enum.
type EnumVal struct {
	Name  string
	Value uint16
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
			if err := commentOrEOL(l, 1); err != nil {
				return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
			}
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
			return fmt.Errorf("[Line %d]: error: Enum %q already contains enumerator(%s) with value %d", l.LineNum, e.Name, e.Values[uint16(n)].Name, n)
		}
		v := EnumVal{Name: l.Items[0].Val, Value: uint16(n)}
		e.Names[v.Name] = v
		e.Values[v.Value] = v

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

type Struct struct {
	Name   string
	Fields []StructField

	file *File
}

func NewStruct(file *File) Struct {
	return Struct{file: file}
}

// StructField represents a field in a Struct.
type StructField struct {
	// Name is the name of the field.
	Name string
	// Index is the index of the field in the Struct.
	Index uint16
	// Type is the type of the field.
	Type field.Type
	// ListType is the type stored in the list, if this is a list type.
	ListType field.Type
	// IdentName is the name of the Struct or Enum that goes in this field. If not a Struct or Enum,
	// this is empty.
	IdentName string
}

// GoListType returns the Go language type, for use in templates.
func (s *StructField) GoListType() string {
	switch s.ListType {
	case field.FTBool:
		return "bool"
	case field.FTUint8:
		if s.IdentName != "" {
			return s.IdentName // Its an Enum
		}
		return "uint8"
	case field.FTUint16:
		if s.IdentName != "" {
			return s.IdentName // Its an Enum
		}
		return "uint16"
	case field.FTUint32:
		return "uint32"
	case field.FTUint64:
		return "uint64"
	case field.FTInt8:
		return "int8"
	case field.FTInt16:
		return "int16"
	case field.FTInt32:
		return "int32"
	case field.FTInt64:
		return "int64"
	case field.FTString:
		return "string"
	case field.FTBytes:
		return "bytes"
	case field.FTStruct:
		return s.IdentName
	default:
		panic(fmt.Sprintf("Unknown list type %v", s.ListType))
	}
}

func (s *Struct) parse(p *halfpike.Parser) error {
	if err := s.name(p); err != nil {
		return err
	}

	if err := s.fields(p); err != nil {
		return err
	}

	return nil
}

func (s *Struct) name(p *halfpike.Parser) error {
	l := p.Next()
	if len(l.Items) < 3 {
		return fmt.Errorf("[Line %d]: error: Struct line has incorrect format", l.LineNum)
	}

	if err := validateIdent(l.Items[1].Val); err != nil {
		return fmt.Errorf("[Line %d]: error: Struct identifier: %w", l.LineNum, err)
	}

	if l.Items[2].Val != "{" {
		return fmt.Errorf("[Line %d]: error: need `{` after Struct identifier: %s", l.LineNum, l.Raw)
	}

	s.Name = l.Items[1].Val

	if err := commentOrEOL(l, 3); err != nil {
		return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
	}
	return nil
}

func (s *Struct) fields(p *halfpike.Parser) error {
	var l halfpike.Line
	for {
		l = p.Next()
		if p.EOF(l) {
			return fmt.Errorf("[Line %d]: Malformed Struct, EOF reached before closing '}'", l.LineNum)
		}
		if l.Items[0].Val == "}" {
			if err := commentOrEOL(l, 1); err != nil {
				return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
			}
			break
		}
		p.Backup()
		if err := s.field(p); err != nil {
			return err
		}
	}
	// We are on the line with }.
	if len(s.Fields) == 0 {
		return fmt.Errorf("Struct %q has no entries, which is not valid", s.Name)
	}
	if err := commentOrEOL(l, 1); err != nil {
		return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
	}

	// Validate the fields are sequentially ordered. The order in the file doesn't matter, as long
	// as we start at 1 and don't skip a number.
	ids := make([]bool, len(s.Fields))
	for _, f := range s.Fields {
		if f.Index < 1 {
			return fmt.Errorf("Struct %q field %q has an invalid field number %d, fields start at 1, not 0", s.Name, f.Name, f.Index)
		}
		slIndex := f.Index - 1
		if len(ids) <= int(slIndex) {
			return fmt.Errorf("Struct %q field %q has an invalid field number %d, fields must start at 1 and be sequential", s.Name, f.Name, f.Index)
		}
		if ids[slIndex] {
			return fmt.Errorf("Struct %q field %q has duplicate field number %d", s.Name, f.Name, f.Index)
		}
		ids[slIndex] = true
	}

	return nil
}

func (s *Struct) field(p *halfpike.Parser) error {
	l := p.Next()
	if p.EOF(l) {
		return fmt.Errorf("[Line %d]: Struct %q had no ending '}'", l.LineNum, s.Name)
	}
	if len(l.Items) < 3 {
		return fmt.Errorf("[Line %d]: Struct field is invalid format", l.LineNum)
	}
	if err := validateIdent(l.Items[0].Val); err != nil {
		return fmt.Errorf("[Line %d]: Struct name %q is invalid: %w", l.LineNum, l.Items[0].Val, err)
	}
	f := StructField{Name: l.Items[0].Val}

	switch l.Items[1].Val {
	case "bool":
		f.Type = field.FTBool
	case "uint8":
		f.Type = field.FTUint8
	case "uint16":
		f.Type = field.FTUint16
	case "uint32":
		f.Type = field.FTUint32
	case "uint64":
		f.Type = field.FTUint64
	case "int8":
		f.Type = field.FTInt8
	case "int16":
		f.Type = field.FTInt16
	case "int32":
		f.Type = field.FTInt32
	case "int64":
		f.Type = field.FTInt64
	case "float32":
		f.Type = field.FTFloat32
	case "float64":
		f.Type = field.FTFloat64
	case "string":
		f.Type = field.FTString
	case "bytes":
		f.Type = field.FTBytes
	case "[]bool":
		f.Type = field.FTListBool
	case "[]uint8":
		f.Type = field.FTList8
	case "[]uint16":
		f.Type = field.FTList16
		f.ListType = field.FTUint16
	case "[]uint32":
		f.Type = field.FTList32
		f.ListType = field.FTUint32
	case "[]uint64":
		f.Type = field.FTList64
		f.ListType = field.FTUint64
	case "[]int8":
		f.Type = field.FTList8
		f.ListType = field.FTInt8
	case "[]int16":
		f.Type = field.FTList16
		f.ListType = field.FTInt16
	case "[]int32":
		f.Type = field.FTList32
		f.ListType = field.FTInt32
	case "[]int64":
		f.Type = field.FTList64
		f.ListType = field.FTInt64
	case "[]float32":
		f.Type = field.FTFloat32
		f.ListType = field.FTFloat32
	case "[]float64":
		f.Type = field.FTFloat64
		f.ListType = field.FTFloat64
	case "[]string":
		f.Type = field.FTListString
		f.ListType = field.FTString
	case "[]bytes":
		f.Type = field.FTListBytes
		f.ListType = field.FTBytes
	default:
		ft := l.Items[1].Val
		isList := false
		if strings.HasPrefix(ft, "[]") {
			ft = strings.Split(ft, "[]")[1]
			isList = true
		}

		// We have a Struct field that has itself as a type or is duplicate of an existing type.
		if s.Name == ft {
			_, ok := s.file.Identifers[ft]
			if ok {
				return fmt.Errorf("[Line %d]: found duplicate top level identifier %q", l.LineNum, ft)
			}

			f.IdentName = f.Name
			if isList {
				f.Type = field.FTListStruct
			} else {
				f.Type = field.FTStruct
			}
		} else {
			// See if field type is an identifer of an Enum or Struct.
			ident, ok := s.file.Identifers[ft]
			if !ok {
				return fmt.Errorf("[Line %d]: Struct %q has field %q with unknown type %q", l.LineNum, s.Name, f.Name, ft)
			}

			switch v := ident.(type) {
			case Enum:
				f.IdentName = ft
				switch v.Size {
				case 8:
					if isList {
						f.Type = field.FTList8
					} else {
						f.Type = field.FTUint8
					}
				case 16:
					if isList {
						f.Type = field.FTList16
					} else {
						f.Type = field.FTUint16
					}
				default:
					panic(fmt.Sprintf("bug: got an Enum with size %d", v.Size))
				}
			case Struct:
				f.IdentName = ft
				if isList {
					f.Type = field.FTListStruct
				} else {
					f.Type = field.FTStruct
				}
			default:
				panic(fmt.Sprintf("bug: we have an identifier %q that is not Enum or Struct, was %T", f.Name, ident))
			}
		}
	}

	fieldNum := l.Items[2].Val
	if !strings.HasPrefix(fieldNum, "@") {
		return fmt.Errorf("[Line %d]: Struct %q has field %q without a valid field number, %q", l.LineNum, s.Name, f.Name, fieldNum)
	}
	fieldNum = strings.Split(fieldNum, "@")[1]
	i, err := strconv.Atoi(fieldNum)
	if err != nil {
		return fmt.Errorf("[Line %d]: Struct %q has field %q without a valid field number, %q", l.LineNum, s.Name, f.Name, fieldNum)
	}
	if i > math.MaxUint16 {
		return fmt.Errorf("[Line %d]: Struct %q has field %q with a field number > that a uint16 can hold, %q", l.LineNum, s.Name, f.Name, fieldNum)
	}
	f.Index = uint16(i)
	s.Fields = append(s.Fields, f)
	if err := commentOrEOL(l, 3); err != nil {
		return fmt.Errorf("[Line %d]: error: %w", l.LineNum, err)
	}
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

func isComment(item halfpike.Item) bool {
	return strings.HasPrefix(item.Val, "//")
}

func withoutCommentEOL(line halfpike.Line) halfpike.Line {
	for x := 0; x < len(line.Items); x++ {
		if isComment(line.Items[x]) {
			line.Items = line.Items[:x]
			return line
		}
		switch line.Items[x].Type {
		case halfpike.ItemEOF, halfpike.ItemEOL:
			line.Items = line.Items[:x]
			return line
		}
	}
	return line
}

func commentOrEOL(line halfpike.Line, from int) error {
	if isComment(line.Items[from]) {
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

func validImportPath(p string) (string, error) {
	if !strings.HasPrefix(p, `"`) || !strings.HasSuffix(p, `"`) {
		return "", fmt.Errorf("invalid import path %q must be contained in double quotes", p)
	}

	wq := strings.Trim(p, `"`)
	if !strings.Contains(wq, "/") {
		return "", fmt.Errorf("invalid import path %q: can't find package name", p)
	}
	if strings.HasSuffix(wq, "/") {
		return "", fmt.Errorf("invalid import path %q: can't have trailing slash", p)
	}
	return wq, nil
}

func pkgFromImpPath(p string) string {
	return path.Base(p)
}
