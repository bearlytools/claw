package value

import (
	"fmt"
	"iter"
	"strings"
	"sync"
	"unicode"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/languages/go/reflect/internal/pragma"
	"github.com/bearlytools/claw/languages/go/reflect/runtime"
	"github.com/gostdlib/base/context"
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

// New creates a new interfaces.Struct based on this StructDescrImpl.
func (s StructDescrImpl) New() interfaces.Struct {
	v := segment.New(context.Background(), s.Mapping)
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
		if fd.Name() == name {
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
	return f.SD.StructName()
}

// IsMap returns true if this field is a map type.
func (f FieldDescrImpl) IsMap() bool {
	return f.FD.Type == field.FTMap
}

// MapKeyType returns the key type for map fields.
func (f FieldDescrImpl) MapKeyType() field.Type {
	if f.FD.Type != field.FTMap {
		panic(fmt.Sprintf("cannot call MapKeyType() on non-map field(%s)", f.FD.Type))
	}
	return f.FD.KeyType
}

// MapValueType returns the value type for map fields.
func (f FieldDescrImpl) MapValueType() field.Type {
	if f.FD.Type != field.FTMap {
		panic(fmt.Sprintf("cannot call MapValueType() on non-map field(%s)", f.FD.Type))
	}
	return f.FD.ValueType
}

type ListBools struct {
	items []bool
	doNotImplement
}

func NewListBools(items []bool) ListBools {
	return ListBools{items: items}
}

func (l ListBools) Type() field.Type {
	return field.FTListBools
}

func (l ListBools) Len() int {
	return len(l.items)
}

func (l ListBools) Get(i int) interfaces.Value {
	return ValueOfBool(l.items[i])
}

func (l ListBools) Set(i int, v interfaces.Value) {
	l.items[i] = v.Bool()
}

func (l ListBools) Append(v interfaces.Value) {
	l.items = append(l.items, v.Bool())
}

func (l ListBools) New() interfaces.Struct {
	panic("ListBools does not support New()")
}

type ListNumbers[N interfaces.Number] struct {
	items []N
	ty    field.Type
	doNotImplement
}

func NewListNumbers[N interfaces.Number](items []N) ListNumbers[N] {
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
	return ListNumbers[N]{items: items, ty: ty}
}

func (l ListNumbers[N]) Type() field.Type {
	return l.ty
}

func (l ListNumbers[N]) Len() int {
	return len(l.items)
}

func (l ListNumbers[N]) Get(i int) interfaces.Value {
	return ValueOfNumber(l.items[i])
}

func (l ListNumbers[N]) Set(i int, v interfaces.Value) {
	l.items[i] = v.Any().(N)
}

func (l ListNumbers[N]) Append(v interfaces.Value) {
	l.items = append(l.items, v.Any().(N))
}

func (l ListNumbers[N]) New() interfaces.Struct {
	panic("ListNumbers does not support New()")
}

type ListBytes struct {
	items [][]byte
	doNotImplement
}

func NewListBytes(items [][]byte) ListBytes {
	return ListBytes{items: items}
}

func (l ListBytes) Type() field.Type {
	return field.FTListBytes
}

func (l ListBytes) Len() int {
	return len(l.items)
}

func (l ListBytes) Get(i int) interfaces.Value {
	return ValueOfBytes(l.items[i])
}

func (l ListBytes) Set(i int, v interfaces.Value) {
	l.items[i] = v.Bytes()
}

func (l ListBytes) Append(v interfaces.Value) {
	l.items = append(l.items, v.Bytes())
}

func (l ListBytes) New() interfaces.Struct {
	panic("ListBytes does not support New()")
}

type ListStrings struct {
	items []string
	doNotImplement
}

func NewListStrings(items []string) ListStrings {
	return ListStrings{items: items}
}

func (l ListStrings) Type() field.Type {
	return field.FTListStrings
}

func (l ListStrings) Len() int {
	return len(l.items)
}

func (l ListStrings) Get(i int) interfaces.Value {
	return ValueOfString(l.items[i])
}

func (l ListStrings) Set(i int, v interfaces.Value) {
	l.items[i] = v.String()
}

func (l ListStrings) Append(v interfaces.Value) {
	l.items = append(l.items, v.String())
}

func (l ListStrings) New() interfaces.Struct {
	panic("ListStrings does not support New()")
}

type ListStructs struct {
	items []*segment.Struct
	sd    StructDescrImpl
	doNotImplement
}

func NewListStructs(items []*segment.Struct, m *mapping.Map) ListStructs {
	sd := StructDescrImpl{
		Name:    m.Name,
		Pkg:     m.Pkg,
		Path:    m.Path,
		Mapping: m,
	}
	for _, fd := range m.Fields {
		sd.FieldList = append(sd.FieldList, FieldDescrImpl{FD: fd})
	}
	return ListStructs{items: items, sd: sd}
}

func (l ListStructs) Type() field.Type {
	return field.FTListStructs
}

func (l ListStructs) Len() int {
	return len(l.items)
}

func (l ListStructs) Get(i int) interfaces.Value {
	s := StructImpl{
		s:     l.items[i],
		descr: l.sd,
	}
	return ValueOfStruct(s)
}

func (l ListStructs) Set(i int, v interfaces.Value) {
	impl := v.Struct().(StructImpl)
	l.items[i] = impl.s
}

func (l ListStructs) Append(v interfaces.Value) {
	impl := v.Struct().(StructImpl)
	l.items = append(l.items, impl.s)
}

func (l ListStructs) New() interfaces.Struct {
	newStruct := segment.New(context.Background(), l.sd.Mapping)
	return NewStruct(newStruct, l.sd)
}

// MapImpl is a generic map implementation that wraps segment.Maps.
// It provides a type-erased interface for reflection access to maps.
type MapImpl[K segment.MapKey, V any] struct {
	m        *segment.Maps[K, V]
	keyType  field.Type
	valType  field.Type
	valDescr *StructDescrImpl // for struct-valued maps
	doNotImplement
}

// NewMapImpl creates a new MapImpl wrapping a segment.Maps.
func NewMapImpl[K segment.MapKey, V any](m *segment.Maps[K, V], keyType, valType field.Type, valDescr *StructDescrImpl) MapImpl[K, V] {
	return MapImpl[K, V]{m: m, keyType: keyType, valType: valType, valDescr: valDescr}
}

func (m MapImpl[K, V]) KeyType() field.Type   { return m.keyType }
func (m MapImpl[K, V]) ValueType() field.Type { return m.valType }
func (m MapImpl[K, V]) Len() int              { return m.m.Len() }

func (m MapImpl[K, V]) Has(key interfaces.Value) bool {
	k := m.extractKey(key)
	return m.m.Has(k)
}

func (m MapImpl[K, V]) Get(key interfaces.Value) interfaces.Value {
	k := m.extractKey(key)
	v, ok := m.m.Get(k)
	if !ok {
		return nil
	}
	return m.wrapValue(v)
}

func (m MapImpl[K, V]) Set(key interfaces.Value, val interfaces.Value) {
	k := m.extractKey(key)
	v := m.extractValue(val)
	m.m.Set(k, v)
}

func (m MapImpl[K, V]) Delete(key interfaces.Value) {
	k := m.extractKey(key)
	m.m.Delete(k)
}

func (m MapImpl[K, V]) Range(f func(key, val interfaces.Value) bool) {
	for k, v := range m.m.All() {
		if !f(m.wrapKey(k), m.wrapValue(v)) {
			return
		}
	}
}

func (m MapImpl[K, V]) extractKey(key interfaces.Value) K {
	var zero K
	switch any(zero).(type) {
	case string:
		return any(key.String()).(K)
	case bool:
		return any(key.Bool()).(K)
	case int8:
		return any(int8(key.Int())).(K)
	case int16:
		return any(int16(key.Int())).(K)
	case int32:
		return any(int32(key.Int())).(K)
	case int64:
		return any(key.Int()).(K)
	case uint8:
		return any(uint8(key.Uint())).(K)
	case uint16:
		return any(uint16(key.Uint())).(K)
	case uint32:
		return any(uint32(key.Uint())).(K)
	case uint64:
		return any(key.Uint()).(K)
	case float32:
		return any(float32(key.Float())).(K)
	case float64:
		return any(key.Float()).(K)
	default:
		panic(fmt.Sprintf("unsupported map key type: %T", zero))
	}
}

func (m MapImpl[K, V]) wrapKey(k K) interfaces.Value {
	switch v := any(k).(type) {
	case string:
		return ValueOfString(v)
	case bool:
		return ValueOfBool(v)
	case int8:
		return ValueOfNumber(v)
	case int16:
		return ValueOfNumber(v)
	case int32:
		return ValueOfNumber(v)
	case int64:
		return ValueOfNumber(v)
	case uint8:
		return ValueOfNumber(v)
	case uint16:
		return ValueOfNumber(v)
	case uint32:
		return ValueOfNumber(v)
	case uint64:
		return ValueOfNumber(v)
	case float32:
		return ValueOfNumber(v)
	case float64:
		return ValueOfNumber(v)
	default:
		panic(fmt.Sprintf("unsupported map key type: %T", k))
	}
}

func (m MapImpl[K, V]) wrapValue(v V) interfaces.Value {
	switch m.valType {
	case field.FTStruct:
		if s, ok := any(v).(*segment.Struct); ok && s != nil {
			return ValueOfStruct(NewStruct(s, m.valDescr))
		}
		return nil
	case field.FTString:
		return ValueOfString(any(v).(string))
	case field.FTBool:
		return ValueOfBool(any(v).(bool))
	case field.FTInt8:
		return ValueOfNumber(any(v).(int8))
	case field.FTInt16:
		return ValueOfNumber(any(v).(int16))
	case field.FTInt32:
		return ValueOfNumber(any(v).(int32))
	case field.FTInt64:
		return ValueOfNumber(any(v).(int64))
	case field.FTUint8:
		return ValueOfNumber(any(v).(uint8))
	case field.FTUint16:
		return ValueOfNumber(any(v).(uint16))
	case field.FTUint32:
		return ValueOfNumber(any(v).(uint32))
	case field.FTUint64:
		return ValueOfNumber(any(v).(uint64))
	case field.FTFloat32:
		return ValueOfNumber(any(v).(float32))
	case field.FTFloat64:
		return ValueOfNumber(any(v).(float64))
	default:
		panic(fmt.Sprintf("unsupported map value type: %s", m.valType))
	}
}

func (m MapImpl[K, V]) extractValue(val interfaces.Value) V {
	var zero V
	switch m.valType {
	case field.FTStruct:
		if s := val.Struct(); s != nil {
			return any(RealStruct(s)).(V)
		}
		return zero
	case field.FTString:
		return any(val.String()).(V)
	case field.FTBool:
		return any(val.Bool()).(V)
	case field.FTInt8:
		return any(int8(val.Int())).(V)
	case field.FTInt16:
		return any(int16(val.Int())).(V)
	case field.FTInt32:
		return any(int32(val.Int())).(V)
	case field.FTInt64:
		return any(val.Int()).(V)
	case field.FTUint8:
		return any(uint8(val.Uint())).(V)
	case field.FTUint16:
		return any(uint16(val.Uint())).(V)
	case field.FTUint32:
		return any(uint32(val.Uint())).(V)
	case field.FTUint64:
		return any(val.Uint()).(V)
	case field.FTFloat32:
		return any(float32(val.Float())).(V)
	case field.FTFloat64:
		return any(val.Float()).(V)
	default:
		panic(fmt.Sprintf("unsupported map value type: %s", m.valType))
	}
}

type StructImpl struct {
	s     *segment.Struct
	descr interfaces.StructDescr
}

func NewStruct(s *segment.Struct, descr interfaces.StructDescr) StructImpl {
	return StructImpl{s: s, descr: descr}
}

func (s StructImpl) Descriptor() interfaces.StructDescr {
	return s.descr
}

func (s StructImpl) New() interfaces.Struct {
	n := segment.New(context.Background(), s.s.Mapping())
	return NewStruct(n, s.descr)
}

func (s StructImpl) ClawInternal(pragma.DoNotImplement) {}

// Fields returns an iterator over all field descriptors and values in the struct.
func (s StructImpl) Fields() iter.Seq2[interfaces.FieldDescr, interfaces.Value] {
	return func(yield func(interfaces.FieldDescr, interfaces.Value) bool) {
		for _, fdescr := range s.descr.Fields() {
			if !yield(fdescr, s.Get(fdescr)) {
				return
			}
		}
	}
}

// Range iterates over fields using a callback function (legacy method).
// Consider using Fields() for better composability with Go 1.24 iterators.
func (s StructImpl) Range(f func(interfaces.FieldDescr, interfaces.Value) bool) {
	for fdescr, value := range s.Fields() {
		if !f(fdescr, value) {
			return
		}
	}
}

func (s StructImpl) Get(descr interfaces.FieldDescr) interfaces.Value {
	return GetValue(s.s, descr.FieldNum())
}

func (s StructImpl) Has(descr interfaces.FieldDescr) bool {
	return s.s.HasField(descr.FieldNum())
}

func (s StructImpl) Clear(desc interfaces.FieldDescr) {
	ClearField(s.s, desc.FieldNum(), desc.Type())
}

func (s StructImpl) Set(descr interfaces.FieldDescr, v interfaces.Value) {
	SetField(s.s, descr.FieldNum(), descr.Type(), v.Any())
}

func (s StructImpl) NewField(descr interfaces.FieldDescr) interfaces.Value {
	// Create a zero-value Value for the given field type
	switch descr.Type() {
	case field.FTBool:
		return ValueOfBool(false)
	case field.FTInt8:
		return ValueOfNumber[int8](0)
	case field.FTInt16:
		return ValueOfNumber[int16](0)
	case field.FTInt32:
		return ValueOfNumber[int32](0)
	case field.FTInt64:
		return ValueOfNumber[int64](0)
	case field.FTUint8:
		if descr.IsEnum() {
			return ValueOfEnum[uint8](0, descr.EnumGroup())
		}
		return ValueOfNumber[uint8](0)
	case field.FTUint16:
		if descr.IsEnum() {
			return ValueOfEnum[uint16](0, descr.EnumGroup())
		}
		return ValueOfNumber[uint16](0)
	case field.FTUint32:
		return ValueOfNumber[uint32](0)
	case field.FTUint64:
		return ValueOfNumber[uint64](0)
	case field.FTFloat32:
		return ValueOfNumber[float32](0)
	case field.FTFloat64:
		return ValueOfNumber[float64](0)
	case field.FTBytes:
		return ValueOfBytes(nil)
	case field.FTString:
		return ValueOfString("")
	case field.FTStruct:
		// Create a new empty struct
		return ValueOfStruct(s.New())
	case field.FTListBools:
		return ValueOfList(NewListBools(nil))
	case field.FTListInt8:
		return ValueOfList(NewListNumbers[int8](nil))
	case field.FTListInt16:
		return ValueOfList(NewListNumbers[int16](nil))
	case field.FTListInt32:
		return ValueOfList(NewListNumbers[int32](nil))
	case field.FTListInt64:
		return ValueOfList(NewListNumbers[int64](nil))
	case field.FTListUint8:
		return ValueOfList(NewListNumbers[uint8](nil))
	case field.FTListUint16:
		return ValueOfList(NewListNumbers[uint16](nil))
	case field.FTListUint32:
		return ValueOfList(NewListNumbers[uint32](nil))
	case field.FTListUint64:
		return ValueOfList(NewListNumbers[uint64](nil))
	case field.FTListFloat32:
		return ValueOfList(NewListNumbers[float32](nil))
	case field.FTListFloat64:
		return ValueOfList(NewListNumbers[float64](nil))
	case field.FTListBytes:
		return ValueOfList(NewListBytes(nil))
	case field.FTListStrings:
		return ValueOfList(NewListStrings(nil))
	case field.FTListStructs:
		// For list of structs, we need a mapping - get it from the StructDescr
		if sd, ok := s.descr.(StructDescrImpl); ok {
			return ValueOfList(NewListStructs(nil, sd.Mapping))
		}
		panic("cannot create ListStructs without mapping")
	default:
		panic(fmt.Sprintf("bug: unsupported type %s", descr.Type()))
	}
}

// Struct extracts the *segment.Struct that holds all the data (not a reflect.Struct).
func (s StructImpl) Struct() *segment.Struct {
	return s.s
}

// RealStruct extracts the *segment.Struct that holds all the data (not a reflect.Struct).
func RealStruct(s interfaces.Struct) *segment.Struct {
	i := s.(StructImpl)
	return i.s
}

// ClearField clears a field by setting it to its zero value.
func ClearField(s *segment.Struct, fieldNum uint16, ft field.Type) {
	switch ft {
	case field.FTBool:
		segment.SetBool(s, fieldNum, false)
	case field.FTInt8:
		segment.SetInt8(s, fieldNum, 0)
	case field.FTInt16:
		segment.SetInt16(s, fieldNum, 0)
	case field.FTInt32:
		segment.SetInt32(s, fieldNum, 0)
	case field.FTInt64:
		segment.SetInt64(s, fieldNum, 0)
	case field.FTUint8:
		segment.SetUint8(s, fieldNum, 0)
	case field.FTUint16:
		segment.SetUint16(s, fieldNum, 0)
	case field.FTUint32:
		segment.SetUint32(s, fieldNum, 0)
	case field.FTUint64:
		segment.SetUint64(s, fieldNum, 0)
	case field.FTFloat32:
		segment.SetFloat32(s, fieldNum, 0)
	case field.FTFloat64:
		segment.SetFloat64(s, fieldNum, 0)
	case field.FTBytes, field.FTString:
		segment.SetBytes(s, fieldNum, nil)
	case field.FTStruct:
		segment.SetNestedStruct(s, fieldNum, nil)
	default:
		// For list types, setting nil/empty removes the field
		// The segment package handles this in SetBytes, etc.
	}
}

// SetField sets a field value using the segment setters.
func SetField(s *segment.Struct, fieldNum uint16, ft field.Type, v any) {
	switch ft {
	case field.FTBool:
		segment.SetBool(s, fieldNum, v.(bool))
	case field.FTInt8:
		segment.SetInt8(s, fieldNum, v.(int8))
	case field.FTInt16:
		segment.SetInt16(s, fieldNum, v.(int16))
	case field.FTInt32:
		segment.SetInt32(s, fieldNum, v.(int32))
	case field.FTInt64:
		segment.SetInt64(s, fieldNum, v.(int64))
	case field.FTUint8:
		// Handle enum values which are passed as interfaces.Enum
		if e, ok := v.(interfaces.Enum); ok {
			segment.SetUint8(s, fieldNum, uint8(e.Number()))
		} else {
			segment.SetUint8(s, fieldNum, v.(uint8))
		}
	case field.FTUint16:
		// Handle enum values which are passed as interfaces.Enum
		if e, ok := v.(interfaces.Enum); ok {
			segment.SetUint16(s, fieldNum, e.Number())
		} else {
			segment.SetUint16(s, fieldNum, v.(uint16))
		}
	case field.FTUint32:
		segment.SetUint32(s, fieldNum, v.(uint32))
	case field.FTUint64:
		segment.SetUint64(s, fieldNum, v.(uint64))
	case field.FTFloat32:
		segment.SetFloat32(s, fieldNum, v.(float32))
	case field.FTFloat64:
		segment.SetFloat64(s, fieldNum, v.(float64))
	case field.FTBytes:
		segment.SetBytes(s, fieldNum, v.([]byte))
	case field.FTString:
		segment.SetStringAsBytes(s, fieldNum, v.(string))
	case field.FTStruct:
		// v should be an interfaces.Struct, extract the underlying segment.Struct
		if st, ok := v.(interfaces.Struct); ok {
			segment.SetNestedStruct(s, fieldNum, RealStruct(st))
		}
	default:
		panic(fmt.Sprintf("SetField: unsupported type %s", ft))
	}
}

// GetValue allows us to get a Value from the internal Struct representation.
// If the value of the field is not set, GetValue() returns nil.
func GetValue(s *segment.Struct, fieldNum uint16) interfaces.Value {
	if !s.HasField(fieldNum) {
		return nil
	}

	// Get the field type from the mapping
	descr := s.Mapping().Fields[fieldNum]

	switch descr.Type {
	case field.FTBool:
		b := segment.GetBool(s, fieldNum)
		return ValueOfBool(b)
	case field.FTInt8:
		n := segment.GetInt8(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTInt16:
		n := segment.GetInt16(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTInt32:
		n := segment.GetInt32(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTInt64:
		n := segment.GetInt64(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTUint8:
		n := segment.GetUint8(s, fieldNum)
		if descr.IsEnum {
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
		n := segment.GetUint16(s, fieldNum)
		if descr.IsEnum {
			pkgDescr := runtime.PackageDescr(descr.FullPath)
			eg := pkgDescr.Enums().ByName(descr.EnumGroup)
			return ValueOfEnum(n, eg)
		}
		return ValueOfNumber(n)
	case field.FTUint32:
		n := segment.GetUint32(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTUint64:
		n := segment.GetUint64(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTFloat32:
		n := segment.GetFloat32(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTFloat64:
		n := segment.GetFloat64(s, fieldNum)
		return ValueOfNumber(n)
	case field.FTBytes:
		b := segment.GetBytes(s, fieldNum)
		return ValueOfBytes(b)
	case field.FTString:
		str := segment.GetString(s, fieldNum)
		return ValueOfString(str)
	case field.FTStruct:
		childMapping := descr.Mapping
		if childMapping == nil {
			panic(fmt.Sprintf("field %s has no mapping", descr.Name))
		}
		st := segment.GetNestedStruct(s, fieldNum, childMapping)
		if st == nil {
			return nil
		}
		sd := StructDescrImpl{
			Name:    childMapping.Name,
			Pkg:     childMapping.Pkg,
			Path:    childMapping.Path,
			Mapping: childMapping,
		}
		for _, fd := range childMapping.Fields {
			sd.FieldList = append(sd.FieldList, FieldDescrImpl{FD: fd})
		}
		return ValueOfStruct(NewStruct(st, sd))
	case field.FTListBools:
		items := getListBools(s, fieldNum)
		return ValueOfList(NewListBools(items))
	case field.FTListInt8:
		items := getListNumbers[int8](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListInt16:
		items := getListNumbers[int16](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListInt32:
		items := getListNumbers[int32](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListInt64:
		items := getListNumbers[int64](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListUint8:
		items := getListNumbers[uint8](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListUint16:
		items := getListNumbers[uint16](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListUint32:
		items := getListNumbers[uint32](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListUint64:
		items := getListNumbers[uint64](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListFloat32:
		items := getListNumbers[float32](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListFloat64:
		items := getListNumbers[float64](s, fieldNum)
		return ValueOfList(NewListNumbers(items))
	case field.FTListBytes:
		items := getListBytes(s, fieldNum)
		return ValueOfList(NewListBytes(items))
	case field.FTListStrings:
		items := getListStrings(s, fieldNum)
		return ValueOfList(NewListStrings(items))
	case field.FTListStructs:
		childMapping := descr.Mapping
		if childMapping == nil {
			panic(fmt.Sprintf("field %s has no mapping", descr.Name))
		}
		items := getListStructs(s, fieldNum, childMapping)
		return ValueOfList(NewListStructs(items, childMapping))
	default:
		panic(fmt.Sprintf("unsupported field type %s", descr.Type))
	}
}

// getListBools parses a bool list from segment data.
func getListBools(s *segment.Struct, fieldNum uint16) []bool {
	offset, size := s.FieldOffset(fieldNum)
	if size <= segment.HeaderSize {
		return nil
	}

	data := s.SegmentBytes()[offset : offset+size]
	// Skip header, data is packed bits
	bitData := data[segment.HeaderSize:]

	// Unpack bools from bits
	items := make([]bool, 0)
	for i := 0; i < len(bitData)*8; i++ {
		byteIdx := i / 8
		bitIdx := uint(i % 8)
		if byteIdx < len(bitData) {
			items = append(items, (bitData[byteIdx]&(1<<bitIdx)) != 0)
		}
	}
	return items
}

// getListNumbers parses a number list from segment data.
func getListNumbers[N segment.Number](s *segment.Struct, fieldNum uint16) []N {
	// Create a temporary segment.Numbers to read from the parent
	nums := segment.NewNumbers[N](s, fieldNum)
	result := make([]N, nums.Len())
	for i := 0; i < nums.Len(); i++ {
		result[i] = nums.Get(i)
	}
	return result
}

// getListBytes parses a bytes list from segment data.
func getListBytes(s *segment.Struct, fieldNum uint16) [][]byte {
	bytes := segment.NewBytes(s, fieldNum)
	result := make([][]byte, bytes.Len())
	for i := 0; i < bytes.Len(); i++ {
		result[i] = bytes.Get(i)
	}
	return result
}

// getListStrings parses a strings list from segment data.
func getListStrings(s *segment.Struct, fieldNum uint16) []string {
	strs := segment.NewStrings(s, fieldNum)
	result := make([]string, strs.Len())
	for i := 0; i < strs.Len(); i++ {
		result[i] = strs.Get(i)
	}
	return result
}

// getListStructs parses a struct list from segment data.
func getListStructs(s *segment.Struct, fieldNum uint16, m *mapping.Map) []*segment.Struct {
	structs := segment.NewStructs(context.Background(), s, fieldNum, m)
	result := make([]*segment.Struct, structs.Len())
	for i := 0; i < structs.Len(); i++ {
		result[i] = structs.Get(i)
	}
	return result
}
