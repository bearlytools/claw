package value

import (
	"fmt"
	"unsafe"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/languages/go/internal/pragma"
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/bearlytools/claw/languages/go/structs"
	"github.com/bearlytools/claw/languages/go/structs/header"
)

// _ PackageDescr is simply used to make sure we are implementing our interface.
var _ PackageDescr = PackageDescrImpl{}

// PackageDescrImpl is the implementation of PackageDescr.
type PackageDescrImpl struct {
	doNotImplement

	Name             string
	Path             string
	ImportDescrs     []PackageDescr
	EnumGroupsDescrs EnumGroups
	StructsDescrs    []StructDescr
}

// PackageName returns the name of the package.
func (p PackageDescrImpl) PackageName() string {
	return p.Name
}

// FullPath returns the full path of the package.
func (p PackageDescrImpl) FullPath() string {
	return p.Path
}

// Imports is a list of imported claw files.
func (p PackageDescrImpl) Imports() []PackageDescr {
	return p.ImportDescrs
}

// Enums is a list of the Enum declarations.
func (p PackageDescrImpl) Enums() EnumGroups {
	return p.EnumGroupsDescrs
}

// Messages is a list of the top-level message declarations.
func (p PackageDescrImpl) Structs() []StructDescr {
	return p.StructsDescrs
}

// _ EnumGroup is simply used to make sure we are implementing our interface.
var _ EnumGroup = EnumGroupImpl{}

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
	Descrs []EnumValueDescr
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
func (e EnumGroupImpl) Get(i int) EnumValueDescr {
	return e.Descrs[i]
}

// ByName returns the EnumValue for an enum named s.
// It returns nil if not found.
func (e EnumGroupImpl) ByName(s string) EnumValueDescr {
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

func (e EnumGroupImpl) ByValue(i int) EnumValueDescr {
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

// _ EnumGroups is simply used to make sure we are implementing our interface.
var _ EnumGroups = EnumGroupsImpl{}

// EnumGroupsImpl implements reflect.EnumGroups.
type EnumGroupsImpl struct {
	doNotImplement

	List   []EnumGroup
	Lookup map[string]EnumGroup
}

// Len reports the number of enum types.
func (e EnumGroupsImpl) Len() int {
	return len(e.List)
}

// Get returns the ith EnumDescriptor. It panics if out of bounds.
func (e EnumGroupsImpl) Get(i int) EnumGroup {
	return e.List[i]
}

// ByName returns the EnumDescriptor for an enum named s.
// It returns nil if not found.
func (e EnumGroupsImpl) ByName(s string) EnumGroup {
	return e.Lookup[s]
}

// _ EnumValueDescrImpl is simply used to make sure we are implementing our interface.
var _ EnumValueDescr = EnumValueDescrImpl{}

// EnumValueDescrImpl implements EnumValueDescr.
type EnumValueDescrImpl struct {
	doNotImplement

	EnumName   string
	EnumNumber uint16
}

// Name returns the name of the Enum value.
func (e EnumValueDescrImpl) Name() string {
	return e.EnumName
}

// Number returns the enum number value.
func (e EnumValueDescrImpl) Number() uint16 {
	return e.EnumNumber
}

// _ StructDescr is simply used to make sure we are implementing our interface.
var _ StructDescr = StructDescrImpl{}

// StructDescrImpl implements StructDescr.
type StructDescrImpl struct {
	doNotImplement

	Name      string
	Pkg       string
	Path      string
	FieldList []FieldDescr
}

func NewStructDescrImpl(m *mapping.Map) StructDescrImpl {
	descr := StructDescrImpl{
		Name: m.Name,
		Pkg:  m.Pkg,
		Path: m.Path,
	}
	for _, fd := range m.Fields {
		descr.FieldList = append(descr.FieldList, FieldDescrImpl{FD: fd})
	}
	return descr
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
func (s StructDescrImpl) Fields() []FieldDescr {
	return s.FieldList
}

// _ FieldDescr is simply used to make sure we are implementing our interface.
var _ FieldDescr = FieldDescrImpl{}

// FieldDescrImpl describes a field inside a Struct type.
type FieldDescrImpl struct {
	FD *mapping.FieldDescr
	SD StructDescr
	EG EnumGroup
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
func (f FieldDescrImpl) EnumGroup() EnumGroup {
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
		panic("cannot call ItemType() on non list of Struct")
	}
	return f.FD.Mapping.Name
}

// _ List is simply used to make sure we are implementing our interface.
var _ List = ListBools{}

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

func (l ListBools) Get(i int) Value {
	return ValueOfBool(l.b.Get(i))
}

func (l ListBools) Set(i int, v Value) {
	l.b.Set(i, v.Bool())
}

func (l ListBools) Append(v Value) {
	l.b.Append(v.Bool())
}

func (l ListBools) New() Struct {
	panic("Listbools does not support New()")
}

// _ List is simply used to make sure we are implementing our interface.
var _ List = ListNumbers[uint8]{}

type ListNumbers[N Number] struct {
	n  *structs.Numbers[N]
	ty field.Type
	doNotImplement
}

func NewListNumbers[N Number](n *structs.Numbers[N]) ListNumbers[N] {
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

func (l ListNumbers[N]) Get(i int) Value {
	return ValueOfNumber(l.n.Get(i))
}

func (l ListNumbers[N]) Set(i int, v Value) {
	l.n.Set(i, v.Any().(N))
}

func (l ListNumbers[N]) Append(v Value) {
	l.n.Append(v.Any().(N))
}

func (l ListNumbers[N]) New() Struct {
	panic("ListNumbers does not support New()")
}

// _ List is simply used to make sure we are implementing our interface.
var _ List = ListBytes{}

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

func (l ListBytes) Get(i int) Value {
	return ValueOfBytes(l.b.Get(i))
}

func (l ListBytes) Set(i int, v Value) {
	l.b.Set(i, v.Bytes())
}

func (l ListBytes) Append(v Value) {
	l.b.Append(v.Bytes())
}

func (l ListBytes) New() Struct {
	panic("ListBytes does not support New()")
}

// _ List is simply used to make sure we are implementing our interface.
var _ List = ListStrings{}

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

func (l ListStrings) Get(i int) Value {
	return ValueOfString(conversions.ByteSlice2String(l.b.Get(i)))
}

func (l ListStrings) Set(i int, v Value) {
	l.b.Set(i, conversions.UnsafeGetBytes(v.String()))
}

func (l ListStrings) Append(v Value) {
	l.b.Append(conversions.UnsafeGetBytes(v.String()))
}

func (l ListStrings) New() Struct {
	panic("ListStrings does not support New()")
}

// _ List is simply used to make sure we are implementing our interface.
var _ List = ListStructs{}

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

func (l ListStructs) Get(i int) Value {
	v := l.l.Get(i)

	s := StructImpl{
		s:     v,
		descr: l.sd,
	}
	return ValueOfStruct(s)
}

func (l ListStructs) Set(i int, v Value) {
	if err := l.l.Set(i, v.aStruct.realType()); err != nil {
		panic(err)
	}
}

func (l ListStructs) Append(v Value) {
	if err := l.l.Append(v.Struct().realType()); err != nil {
		panic(err)
	}
}

func (l ListStructs) New() Struct {
	return NewStruct(l.l.New(), l.sd)
}

// _ Struct is simply used to make sure we are implementing our interface.
var _ Struct = StructImpl{}

type StructImpl struct {
	s     *structs.Struct
	descr StructDescr
}

func NewStruct(s *structs.Struct, descr StructDescr) StructImpl {
	return StructImpl{s: s, descr: descr}
}

func (s StructImpl) Descriptor() StructDescr {
	return s.descr
}

func (s StructImpl) New() Struct {
	n := s.s.NewFrom()
	return NewStruct(n, s.descr)
}

func (s StructImpl) ClawInternal(pragma.DoNotImplement) {}

func (s StructImpl) Range(f func(FieldDescr, Value) bool) {
	for _, fdescr := range s.descr.Fields() {
		ok := f(fdescr, s.Get(fdescr))
		if !ok {
			return
		}
	}
}

func (s StructImpl) Get(descr FieldDescr) Value {
	return GetValue(s.s, descr.FieldNum())
}

func (s StructImpl) Has(descr FieldDescr) bool {
	return s.s.IsSet(descr.FieldNum())
}

func (s StructImpl) Clear(desc FieldDescr) {
	structs.DeleteField(s.s, desc.FieldNum())
}

func (s StructImpl) Set(descr FieldDescr, v Value) {
	structs.SetField(s.s, descr.FieldNum(), v.Any())
}

func (s StructImpl) NewField(descr FieldDescr) Value {
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
		return Value{h: h}
	case field.FTUint16:
		h := header.New()
		h.SetFieldType(field.FTUint16)
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

func (s StructImpl) realType() *structs.Struct {
	return s.s
}

// GetValue allows us to get a Value from the internal Struct representation.
func GetValue(s *structs.Struct, fieldNum uint16) Value {
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
		return ValueOfNumber(n)
	case field.FTUint16:
		n := structs.MustGetNumber[uint16](s, fieldNum)
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
