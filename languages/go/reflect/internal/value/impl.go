package value

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"unicode"
	"unsafe"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/internal/pragma"
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/languages/go/reflect/runtime"
	"github.com/bearlytools/claw/languages/go/structs"
	"github.com/bearlytools/claw/languages/go/structs/header"
)

// PackageDescrImpl is the implementation of PackageDescr.
type PackageDescrImpl struct {
	doNotImplement

	Name             string
	Path             string
	ImportDescrs     []interfaces.PackageDescr
	EnumGroupsDescrs interfaces.EnumGroups
	StructsDescrs    interfaces.StructDescrs

	initOnce sync.Once
}

// PackageName returns the name of the package.
func (p *PackageDescrImpl) PackageName() string {
	return p.Name
}

// FullPath returns the full path of the package.
func (p *PackageDescrImpl) FullPath() string {
	return p.Path
}

// Imports is a list of imported claw files.
func (p *PackageDescrImpl) Imports() []interfaces.PackageDescr {
	return p.ImportDescrs
}

// Enums is a list of the Enum declarations.
func (p *PackageDescrImpl) Enums() interfaces.EnumGroups {
	return p.EnumGroupsDescrs
}

// Messages is a list of the top-level message declarations.
func (p *PackageDescrImpl) Structs() interfaces.StructDescrs {
	return p.StructsDescrs
}

/*
// EnumGroupImpl implements EnumGroup.
type EnumGroupImpl struct {
	doNotImplement

	// GroupName is the name of the EnumGroup.
	GroupName string
	// GroupLen is how many enumerated values are in this group.
	GroupLen int
	// EnumSize is the bit size, 8 or 16, that the values are.
	EnumSize uint8
	// Descrs hold the valuye descriptors.
	Descrs []interfaces.Enum
}

// Name is the name of the enum group.
func (e EnumGroupImpl) Name() string {
	return e.GroupName
}

// Len reports the number of enum values.
func (e EnumGroupImpl) Len() int {
	return e.GroupLen
}

// Get returns the ith EnumValue. It panics if out of bounds.
func (e EnumGroupImpl) Get(i uint16) interfaces.Enum {
	return e.Descrs[i]
}

// ByName returns the EnumValue for an enum named s.
// It returns nil if not found.
func (e EnumGroupImpl) ByName(s string) interfaces.Enum {
	// Enums are usually small and reflection is the slow path. For now,
	// I'm going to simply use a for loop for what I think will be the majority of
	// cases. Go's map implementation is pretty gretat, but I think this will be
	// similar in speed for the majority of cases and not cost us another map allocation.
	for _, descr := range e.Descrs {
		if descr.Name() == s {
			return descr
		}
	}
	return nil
}

func (e EnumGroupImpl) ByValue(i uint16) interfaces.Enum {
	for _, descr := range e.Descrs {
		if descr.Number() == uint16(i) {
			return descr
		}
	}
	return nil
}

// Size returns the size in bits of the enumerator.
func (e EnumGroupImpl) Size() uint8 {
	return e.EnumSize
}

// EnumGroupsImpl implements reflect.EnumGroups.
type EnumGroupsImpl struct {
	doNotImplement

	List   []interfaces.EnumGroup
	Lookup map[string]interfaces.EnumGroup
}

// Len reports the number of enum types.
func (e EnumGroupsImpl) Len() int {
	return len(e.List)
}

// Get returns the ith EnumDescriptor. It panics if out of bounds.
func (e EnumGroupsImpl) Get(i int) interfaces.EnumGroup {
	return e.List[i]
}

// ByName returns the EnumDescriptor for an enum named s.
// It returns nil if not found.
func (e EnumGroupsImpl) ByName(s string) interfaces.EnumGroup {
	return e.Lookup[s]
}

// EnumImpl implements EnumValueDescr.
type EnumImpl struct {
	doNotImplement

	EnumName   string
	EnumNumber uint16
	EnumSize   uint8

	// EnumGroup holds the enum gropu that this enum belongs to.
	EnumGroup EnumGroupImpl
}

// Name returns the name of the Enum value.
func (e EnumImpl) Name() string {
	return e.EnumName
}

// Number returns the enum number value.
func (e EnumImpl) Number() uint16 {
	return e.EnumNumber
}

// Size returns the size of the value, either 8 or 16 bits.
func (e EnumImpl) Size() uint8 {
	return e.EnumSize
}
*/

// StructDescrsImpl implements interfaces.StructDescrs. This stores a list of Struct
// inside a package.
type StructDescrsImpl struct {
	doNotImplement
	Descrs []interfaces.StructDescr
}

// Len reports the number of messages.
func (s StructDescrsImpl) Len() int {
	return len(s.Descrs)
}

// Get returns the ith StructDescr. It panics if out of bounds.
func (s StructDescrsImpl) Get(i int) interfaces.StructDescr {
	return s.Descrs[i]
}

// ByName returns the StructDescr for a Struct named s.
// It returns nil if not found.
func (s StructDescrsImpl) ByName(name string) interfaces.StructDescr {
	for _, v := range s.Descrs {
		if v.StructName() == name {
			return v
		}
	}
	return nil
}

// StructDescrImpl implements StructDescr.
type StructDescrImpl struct {
	doNotImplement

	Name      string
	Pkg       string
	Path      string
	FieldList []interfaces.FieldDescr

	Mapping *mapping.Map
}

/*
// TODO(jdoak): Remove this.
// NewStructDescrImpl creates a new StructDescrImpl
func NewStructDescrImpl(m *mapping.Map) *StructDescrImpl {
	defer log.Println("\tDONE")
	descr := &StructDescrImpl{
		Name:    m.Name,
		Pkg:     m.Pkg,
		Path:    m.Path,
		Mapping: m,
	}

	descr.FieldList = make([]interfaces.FieldDescr, len(m.Fields))

	// Load all non-enum fields and locally defined enum definitions.
	for i, fd := range m.Fields {
		log.Printf("\tNew: field(%s)", fd.Name)
		// Only load fields that reference local types, as external fields
		// have not loaded yet.
		if fd.Package != m.Pkg {
			log.Printf("\tNewStructDescrImpl: pkg(%s), struct(%s), skipping field(%s)", m.Pkg, m.Name, fd.Name)
			continue
		}

		pkgDescr := runtime.PackageDescr(m.Path)
		if pkgDescr == nil {
			panic(fmt.Sprintf("pkg(%s), struct(%s), can't find package descriptor(%s)", m.Pkg, m.Name, m.Path))
		}
		if fd.IsEnum {
			log.Println("\tLooking for local EnumGroup: ", fd.EnumGroup)
			sp := strings.Split(fd.EnumGroup, ".")

			// We only care about locally defined enums.
			if len(sp) != 1 {
				panic(fmt.Sprintf("\tbug: can't have FullPath == '' and enum name %s", fd.EnumGroup))
			}

			egName := fd.EnumGroup
			eg := pkgDescr.Enums().ByName(egName)
			if eg == nil {
				panic(fmt.Sprintf("\tbug: EnumGroup %s could not be found in runtime[%s]", egName, pkgDescr.FullPath()))
			}
			fd := FieldDescrImpl{FD: fd, EG: eg}
			descr.FieldList[i] = fd
		} else {
			log.Println("\tLooking for local type: ", fd.Name)
			v := pkgDescr.Structs().ByName(fd.Name)
			if v == nil {
				panic(fmt.Sprintf("\tpkg(%s), struct(%s), field(%s): can't find in internal package %s", pkgDescr.PackageName(), m.Name, fd.Name, pkgDescr.PackageName()))
			}
			descr.FieldList[i] = FieldDescrImpl{
				FD: fd,
				SD: v,
			}
		}
	}
	return descr
}

// TODO(jdoak): Remove this.
// Init initializes the StructDescrImpl's externally defined reflection types.
// Unfortunately, we need data from other packages and this isn't available until after
// compile time. We could static this in the repo as static code, but it makes the
// templates more unwieldy.
func (s *StructDescrImpl) Init() error {
	//panic("THIS APPREARS to be running for internal and external references")
	log.Println("Init() ran for struct: ", s.Name)
	for i, fd := range s.Mapping.Fields {
		// Ignore any fields that have already been defined.
		if s.FieldList[i] != nil {
			log.Println("skipping field: ", s.FieldList[i].Name())
			continue
		}
		pkgDescr := runtime.PackageDescr(fd.FullPath)

		if fd.IsEnum {
			log.Println("external enumerator FullPath: ", fd.FullPath)
			sp := strings.Split(fd.EnumGroup, ".")
			if len(sp) == 1 {
				continue
			}
			log.Println("Looking for external EnumGroup: ", fd.EnumGroup)

			//pkgName := sp[0]
			egName := sp[1]
			eg := pkgDescr.Enums().ByName(egName)
			if eg == nil {
				return fmt.Errorf("bug(emumerator): pkg %s, struct %s, field %s: could not locate reflect reference for enum", s.Mapping.Path, s.Mapping.Name, fd.Name)
			}

			s.FieldList[i] = FieldDescrImpl{FD: fd, EG: eg}
		} else { // It is a Struct
			log.Println("field FullPath: ", fd.FullPath)
			v := pkgDescr.Structs().ByName(fd.Name)
			if v == nil {
				return fmt.Errorf("bug(Struct): pkg(%s), struct(%s), field(%s): can't find in external package %s", pkgDescr.PackageName(), s.Name, fd.Name, pkgDescr.PackageName())
			}
			s.FieldList[i] = FieldDescrImpl{
				FD: fd,
				SD: v,
			}
		}
	}
	return nil
}
*/

// New creates a new interfaces.Struct based on this StructDescrImpl.
func (s StructDescrImpl) New() interfaces.Struct {
	v := structs.New(0, s.Mapping)
	return StructImpl{
		s:     v,
		descr: s,
	}
}

// Struct Name will be the name of the struct.
func (s StructDescrImpl) StructName() string {
	return s.Name
}

// Package will be return name package name this struct was defined in.
func (s StructDescrImpl) Package() string {
	return s.Pkg
}

// FullPath will return the full path of the package as used in Go import statements.
func (s StructDescrImpl) FullPath() string {
	return s.Path
}

// Fields will return a list of field descriptions.
func (s StructDescrImpl) Fields() []interfaces.FieldDescr {
	return s.FieldList
}

// FieldDescrByName returns the FieldDescr by the name of the field. If the field
// is not found, this will be nil.
func (s StructDescrImpl) FieldDescrByName(name string) interfaces.FieldDescr {
	if name == "" || unicode.IsLower(rune(name[0])) {
		panic("cannot call FieldDescrByName if name is the empty string or starts with a lower case letter")
	}
	for _, fd := range s.FieldList {
		log.Printf("%s == %s", name, fd.Name())
		if fd.Name() == name {
			log.Println("return")
			return fd
		}
	}
	return nil
}

// FieldDescByIndex returns the FieldDescr by index. If the index is out of bounds this
// will panic.
func (s StructDescrImpl) FieldDescrByIndex(index int) interfaces.FieldDescr {
	return s.FieldList[index]
}

// FieldDescrImpl describes a field inside a Struct type.
type FieldDescrImpl struct {
	FD *mapping.FieldDescr
	SD interfaces.StructDescr
	EG interfaces.EnumGroup
}

// Name returns the name of the field.
func (f FieldDescrImpl) Name() string {
	return f.FD.Name
}

// Type returns the type of the field.
func (f FieldDescrImpl) Type() field.Type {
	return f.FD.Type
}

// FieldNum returns the field number in the Struct.
func (f FieldDescrImpl) FieldNum() uint16 {
	return f.FD.FieldNum
}

// IsEnum indicates if the field is an enumerator.
func (f FieldDescrImpl) IsEnum() bool {
	return f.FD.IsEnum
}

// EnumGroup returns the EnumGroup for the field. If the field is not an enum, this
// will panic. Use IsEnum() if you want to check before calling this.
func (f FieldDescrImpl) EnumGroup() interfaces.EnumGroup {
	if f.EG == nil {
		panic("called EnumGroup on field that was not an Enum")
	}
	return f.EG
}

type ItemType struct {
	Name    string
	Path    string
	Mapping mapping.Map
}

// ItemType returns the Struct type in a []Struct. If this is not a []Struct, then
// this will panic.
func (f FieldDescrImpl) ItemType() string {
	if f.FD.Type != field.FTListStructs {
		panic(fmt.Sprintf("cannot call ItemType() on non list of Struct(%s)", f.FD.Type))
	}
	return f.FD.Mapping.Name
}

type ListBools struct {
	b *structs.Bools
	doNotImplement
}

func NewListBools(b *structs.Bools) ListBools {
	return ListBools{b: b}
}

func (l ListBools) Type() field.Type {
	return field.FTListBools
}

func (l ListBools) Len() int {
	return l.b.Len()
}

func (l ListBools) Get(i int) interfaces.Value {
	return ValueOfBool(l.b.Get(i))
}

func (l ListBools) Set(i int, v interfaces.Value) {
	l.b.Set(i, v.Bool())
}

func (l ListBools) Append(v interfaces.Value) {
	l.b.Append(v.Bool())
}

func (l ListBools) New() interfaces.Struct {
	panic("Listbools does not support New()")
}

type ListNumbers[N interfaces.Number] struct {
	n  *structs.Numbers[N]
	ty field.Type
	doNotImplement
}

func NewListNumbers[N interfaces.Number](n *structs.Numbers[N]) ListNumbers[N] {
	var ty field.Type
	var t N

	switch any(t).(type) {
	case int8:
		ty = field.FTInt8
	case int16:
		ty = field.FTInt16
	case int32:
		ty = field.FTInt32
	case int64:
		ty = field.FTInt64
	case uint8:
		ty = field.FTUint8
	case uint16:
		ty = field.FTUint16
	case uint32:
		ty = field.FTUint32
	case uint64:
		ty = field.FTUint64
	case float32:
		ty = field.FTFloat32
	case float64:
		ty = field.FTFloat64
	default:
		panic("bug: unsupported field type")
	}
	return ListNumbers[N]{n: n, ty: ty}
}

func (l ListNumbers[N]) Type() field.Type {
	return l.ty
}

func (l ListNumbers[N]) Len() int {
	return l.n.Len()
}

func (l ListNumbers[N]) Get(i int) interfaces.Value {
	return ValueOfNumber(l.n.Get(i))
}

func (l ListNumbers[N]) Set(i int, v interfaces.Value) {
	l.n.Set(i, v.Any().(N))
}

func (l ListNumbers[N]) Append(v interfaces.Value) {
	l.n.Append(v.Any().(N))
}

func (l ListNumbers[N]) New() interfaces.Struct {
	panic("ListNumbers does not support New()")
}

type ListBytes struct {
	b *structs.Bytes
	doNotImplement
}

func NewListBytes(b *structs.Bytes) ListBytes {
	return ListBytes{b: b}
}

func (l ListBytes) Type() field.Type {
	return field.FTListBytes
}

func (l ListBytes) Len() int {
	return l.b.Len()
}

func (l ListBytes) Get(i int) interfaces.Value {
	return ValueOfBytes(l.b.Get(i))
}

func (l ListBytes) Set(i int, v interfaces.Value) {
	l.b.Set(i, v.Bytes())
}

func (l ListBytes) Append(v interfaces.Value) {
	l.b.Append(v.Bytes())
}

func (l ListBytes) New() interfaces.Struct {
	panic("ListBytes does not support New()")
}

type ListStrings struct {
	b *structs.Bytes
	doNotImplement
}

func NewListStrings(b *structs.Bytes) ListStrings {
	return ListStrings{b: b}
}

func (l ListStrings) Type() field.Type {
	return field.FTListStrings
}

func (l ListStrings) Len() int {
	return l.b.Len()
}

func (l ListStrings) Get(i int) interfaces.Value {
	return ValueOfString(conversions.ByteSlice2String(l.b.Get(i)))
}

func (l ListStrings) Set(i int, v interfaces.Value) {
	l.b.Set(i, conversions.UnsafeGetBytes(v.String()))
}

func (l ListStrings) Append(v interfaces.Value) {
	l.b.Append(conversions.UnsafeGetBytes(v.String()))
}

func (l ListStrings) New() interfaces.Struct {
	panic("ListStrings does not support New()")
}

type ListStructs struct {
	l  *structs.Structs
	sd StructDescrImpl
	doNotImplement
}

func NewListStructs(l *structs.Structs) ListStructs {
	m := l.Map()

	sd := StructDescrImpl{
		Name: m.Name,
		Pkg:  m.Pkg,
		Path: m.Path,
	}
	for _, fd := range m.Fields {
		sd.FieldList = append(sd.FieldList, FieldDescrImpl{FD: fd})
	}
	return ListStructs{l: l, sd: sd}
}

func (l ListStructs) Type() field.Type {
	return field.FTListStructs
}

func (l ListStructs) Len() int {
	return l.l.Len()
}

func (l ListStructs) Get(i int) interfaces.Value {
	v := l.l.Get(i)

	s := StructImpl{
		s:     v,
		descr: l.sd,
	}
	return ValueOfStruct(s)
}

func (l ListStructs) Set(i int, v interfaces.Value) {
	realV := v.(Value)
	if err := l.l.Set(i, RealStruct(realV.aStruct)); err != nil {
		panic(err)
	}
}

func (l ListStructs) Append(v interfaces.Value) {
	impl := v.Struct().(StructImpl)
	if err := l.l.Append(RealStruct(impl)); err != nil {
		panic(err)
	}
}

func (l ListStructs) New() interfaces.Struct {
	return NewStruct(l.l.New(), l.sd)
}

type StructImpl struct {
	s     *structs.Struct
	descr interfaces.StructDescr
}

func NewStruct(s *structs.Struct, descr interfaces.StructDescr) StructImpl {
	return StructImpl{s: s, descr: descr}
}

func (s StructImpl) Descriptor() interfaces.StructDescr {
	return s.descr
}

func (s StructImpl) New() interfaces.Struct {
	n := s.s.NewFrom()
	return NewStruct(n, s.descr)
}

func (s StructImpl) ClawInternal(pragma.DoNotImplement) {}

func (s StructImpl) Range(f func(interfaces.FieldDescr, interfaces.Value) bool) {
	for _, fdescr := range s.descr.Fields() {
		ok := f(fdescr, s.Get(fdescr))
		if !ok {
			return
		}
	}
}

func (s StructImpl) Get(descr interfaces.FieldDescr) interfaces.Value {
	return GetValue(s.s, descr.FieldNum())
}

func (s StructImpl) Has(descr interfaces.FieldDescr) bool {
	return s.s.IsSet(descr.FieldNum())
}

func (s StructImpl) Clear(desc interfaces.FieldDescr) {
	structs.DeleteField(s.s, desc.FieldNum())
}

func (s StructImpl) Set(descr interfaces.FieldDescr, v interfaces.Value) {
	structs.SetField(s.s, descr.FieldNum(), v.Any())
}

func (s StructImpl) NewField(descr interfaces.FieldDescr) interfaces.Value {
	switch descr.Type() {
	case field.FTBool:
		h := header.New()
		h.SetFieldType(field.FTBool)
		return Value{h: h}
	case field.FTInt8:
		h := header.New()
		h.SetFieldType(field.FTInt8)
		return Value{h: h}
	case field.FTInt16:
		h := header.New()
		h.SetFieldType(field.FTInt16)
		return Value{h: h}
	case field.FTInt32:
		h := header.New()
		h.SetFieldType(field.FTInt32)
		return Value{h: h}
	case field.FTInt64:
		h := header.New()
		h.SetFieldType(field.FTInt64)
		p := []byte{0, 0, 0, 0}
		return Value{h: h, ptr: unsafe.Pointer(&p)}
	case field.FTUint8:
		h := header.New()
		h.SetFieldType(field.FTUint8)
		if descr.IsEnum() {
			return Value{h: h, isEnum: true, enumGroup: descr.EnumGroup()}
		}
		return Value{h: h}
	case field.FTUint16:
		h := header.New()
		h.SetFieldType(field.FTUint16)
		if descr.IsEnum() {
			return Value{h: h, isEnum: true, enumGroup: descr.EnumGroup()}
		}
		return Value{h: h}
	case field.FTUint32:
		h := header.New()
		h.SetFieldType(field.FTUint32)
		return Value{h: h}
	case field.FTUint64:
		h := header.New()
		h.SetFieldType(field.FTUint64)
		p := []byte{0, 0, 0, 0}
		return Value{h: h, ptr: unsafe.Pointer(&p)}
	case field.FTFloat32:
		h := header.New()
		h.SetFieldType(field.FTFloat32)
		return Value{h: h}
	case field.FTFloat64:
		h := header.New()
		h.SetFieldType(field.FTFloat64)
		p := []byte{0, 0, 0, 0}
		return Value{h: h, ptr: unsafe.Pointer(&p)}
	case field.FTBytes:
		h := header.New()
		h.SetFieldType(field.FTBytes)
		return Value{h: h}
	case field.FTString:
		h := header.New()
		h.SetFieldType(field.FTString)
		return Value{h: h}
	case field.FTStruct:
		h := header.New()
		h.SetFieldType(field.FTStruct)
		return Value{h: h}
	case field.FTListBools:
		h := header.New()
		h.SetFieldType(field.FTListBools)
		return Value{h: h}
	case field.FTListInt8:
		h := header.New()
		h.SetFieldType(field.FTListInt8)
		return Value{h: h}
	case field.FTListInt16:
		h := header.New()
		h.SetFieldType(field.FTListInt16)
		return Value{h: h}
	case field.FTListInt32:
		h := header.New()
		h.SetFieldType(field.FTListInt32)
		return Value{h: h}
	case field.FTListInt64:
		h := header.New()
		h.SetFieldType(field.FTListInt64)
		return Value{h: h}
	case field.FTListUint8:
		h := header.New()
		h.SetFieldType(field.FTListUint8)
		return Value{h: h}
	case field.FTListUint16:
		h := header.New()
		h.SetFieldType(field.FTListUint16)
		return Value{h: h}
	case field.FTListUint32:
		h := header.New()
		h.SetFieldType(field.FTListUint32)
		return Value{h: h}
	case field.FTListUint64:
		h := header.New()
		h.SetFieldType(field.FTListUint64)
		return Value{h: h}
	case field.FTListFloat32:
		h := header.New()
		h.SetFieldType(field.FTListFloat32)
		return Value{h: h}
	case field.FTListFloat64:
		h := header.New()
		h.SetFieldType(field.FTListFloat64)
		return Value{h: h}
	case field.FTListBytes:
		h := header.New()
		h.SetFieldType(field.FTListBytes)
		return Value{h: h}
	case field.FTListStrings:
		h := header.New()
		h.SetFieldType(field.FTListStrings)
		return Value{h: h}
	case field.FTListStructs:
		h := header.New()
		h.SetFieldType(field.FTListStructs)
		return Value{h: h}
	default:
		panic(fmt.Sprintf("bug: unsupported type %s", descr.Type()))
	}
}

// Struct extracts the *structs.Struct that holds all the data (not a reflect.Struct).
func (s StructImpl) Struct() *structs.Struct {
	return s.s
}

// RealStruct extracts the *structs.Struct that holds all the data (not a reflect.Struct).
func RealStruct(s interfaces.Struct) *structs.Struct {
	i := s.(StructImpl)
	return i.s
}

// GetValue allows us to get a Value from the internal Struct representation.
func GetValue(s *structs.Struct, fieldNum uint16) interfaces.Value {
	sf := s.Fields()[fieldNum]

	switch sf.Header.FieldType() {
	case field.FTBool:
		b := structs.MustGetBool(s, fieldNum)
		return ValueOfBool(b)
	case field.FTInt8:
		n := structs.MustGetNumber[int8](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTInt16:
		n := structs.MustGetNumber[int16](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTInt32:
		n := structs.MustGetNumber[int32](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTInt64:
		n := structs.MustGetNumber[int64](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTUint8:
		n := structs.MustGetNumber[uint8](s, fieldNum)
		descr := s.Map().Fields[fieldNum]
		if descr.IsEnum {
			// TODO(jdoak): This split dynamic that I'm having to do is error prone.
			// This should be simplified. Better yet, we really should have lookup
			// tables that use slice index numbers to packages to make things way faster.
			var egName string
			sp := strings.Split(descr.EnumGroup, ".")
			if len(sp) == 1 {
				egName = sp[0]
			} else {
				egName = sp[1]
			}

			pkgDescr := runtime.PackageDescr(descr.FullPath)
			eg := pkgDescr.Enums().ByName(egName)
			return ValueOfEnum(n, eg)
		}
		return ValueOfNumber(n)
	case field.FTUint16:
		n := structs.MustGetNumber[uint16](s, fieldNum)
		descr := s.Map().Fields[fieldNum]
		if descr.IsEnum {
			pkgDescr := runtime.PackageDescr(descr.FullPath)
			eg := pkgDescr.Enums().ByName(descr.EnumGroup)
			return ValueOfEnum(n, eg)
		}
		return ValueOfNumber(n)
	case field.FTUint32:
		n := structs.MustGetNumber[uint32](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTUint64:
		n := structs.MustGetNumber[uint64](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTFloat32:
		n := structs.MustGetNumber[float32](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTFloat64:
		n := structs.MustGetNumber[float64](s, fieldNum)
		return ValueOfNumber(n)
	case field.FTBytes:
		b := structs.MustGetBytes(s, fieldNum)
		return ValueOfBytes(*b)
	case field.FTString:
		b := structs.MustGetBytes(s, fieldNum)
		return ValueOfString(conversions.ByteSlice2String(*b))
	case field.FTStruct:
		st := structs.MustGetStruct(s, fieldNum)

		sd := StructDescrImpl{
			Name: st.Map().Name,
			Pkg:  st.Map().Pkg,
			Path: st.Map().Path,
		}
		for _, fd := range st.Map().Fields {
			sd.FieldList = append(sd.FieldList, FieldDescrImpl{FD: fd})
		}
		return ValueOfStruct(NewStruct(st, sd))
	case field.FTListBools:
		l := structs.MustGetListBool(s, fieldNum)
		return ValueOfList(ListBools{b: l})
	case field.FTListInt8:
		l := structs.MustGetListNumber[int8](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListInt16:
		l := structs.MustGetListNumber[int16](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListInt32:
		l := structs.MustGetListNumber[int32](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListInt64:
		l := structs.MustGetListNumber[int64](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListUint8:
		l := structs.MustGetListNumber[uint8](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListUint16:
		l := structs.MustGetListNumber[uint16](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListUint32:
		l := structs.MustGetListNumber[uint32](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListUint64:
		l := structs.MustGetListNumber[uint64](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListFloat32:
		l := structs.MustGetListNumber[float32](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListFloat64:
		l := structs.MustGetListNumber[float64](s, fieldNum)
		return ValueOfList(NewListNumbers(l))
	case field.FTListBytes:
		l := structs.MustGetListBytes(s, fieldNum)
		return ValueOfList(NewListBytes(l))
	case field.FTListStrings:
		l := structs.MustGetListBytes(s, fieldNum)
		return ValueOfList(NewListStrings(l))
	case field.FTListStructs:
		l := structs.MustGetListStruct(s, fieldNum)
		return ValueOfList(NewListStructs(l))
	default:
		panic(fmt.Sprintf("unsupported field type %s", sf.Header.FieldType()))
	}
}
