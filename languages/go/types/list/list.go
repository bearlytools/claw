// Package list contains types that implement lists of scalar values that can be store
// in Claw Struct fields. All fields starting with XXX are not convered by any compatibility
// promise and should not be used.
package list

// Note: This wraps values in the structs package that is not for direct use by a user.
// This only deals with lists of scalar values, not []Struct.

import (
	"iter"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/structs"
	"golang.org/x/exp/constraints"
)

// Number represents all int, uint and float types.
type Number interface {
	constraints.Integer | constraints.Float
}

// Bools is a wrapper around a list of boolean values.
type Bools struct {
	b *structs.Bools
}

// NewBools creates a new Bool that will be stored in a Struct field with number fieldNum.
func NewBools() Bools {
	return Bools{b: structs.NewBools(0)}
}

// Internal use only.
func XXXFromBools(b *structs.Bools) Bools {
	return Bools{b: b}
}

// Internal use only.
func (b Bools) XXXBools() *structs.Bools {
	return b.b
}

// Len returns the number of items in this list of bools.
func (b Bools) Len() int {
	return b.b.Len()
}

// Get gets a value in the list[pos].
func (b Bools) Get(index int) bool {
	return b.b.Get(index)
}

// All returns an iterator over all boolean values.
func (b Bools) All() iter.Seq[bool] {
	return b.b.All()
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (b Bools) Range(from, to int) iter.Seq[bool] {
	return b.b.Range(from, to)
}

// Set a boolean in position "pos" to "val".
func (b Bools) Set(index int, val bool) Bools {
	b.b.Set(index, val)
	return b
}

// Append appends values to the list of bools.
func (b Bools) Append(i ...bool) Bools {
	b.b.Append(i...)
	return b
}

// Slice converts this into a standard []bool. The values aren't linked, so changing
// []bool or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (b Bools) Slice() []bool {
	return b.b.Slice()
}

// Numbers represents a list of numbers
type Numbers[N Number] struct {
	n *structs.Numbers[N]
}

// NewNumbers creates a new Numbers.
func NewNumbers[N Number]() Numbers[N] {
	return Numbers[N]{n: structs.NewNumbers[N]()}
}

// Internal use only.
func XXXFromNumbers[N Number](n *structs.Numbers[N]) Numbers[N] {
	return Numbers[N]{n: n}
}

// Internal use only.
func (n Numbers[N]) XXXNumbers() *structs.Numbers[N] {
	return n.n
}

// Len returns the number of items in this list.
func (n Numbers[N]) Len() int {
	return n.n.Len()
}

// Get gets a number stored at the index.
func (n Numbers[N]) Get(index int) N {
	return n.n.Get(index)
}

// All returns an iterator over all numeric values.
func (n Numbers[N]) All() iter.Seq[N] {
	return n.n.All()
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (n Numbers[N]) Range(from, to int) iter.Seq[N] {
	return n.n.Range(from, to)
}

// Set a number in position "index" to "value".
func (n Numbers[N]) Set(index int, value N) Numbers[N] {
	n.n.Set(index, value)
	return n
}

// Append appends values to the list of numbers.
func (n Numbers[N]) Append(i ...N) Numbers[N] {
	n.n.Append(i...)
	return n
}

// Slice converts this into a standard []I, where I is a number value. The values aren't linked, so changing
// []I or calling n.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (n Numbers[N]) Slice() []N {
	return n.n.Slice()
}

// Bytes represents a list of bytes.
type Bytes struct {
	b *structs.Bytes
}

// NewBytes returns a new Bytes.
func NewBytes() Bytes {
	return Bytes{b: structs.NewBytes()}
}

// Internal use only.
func XXXFromBytes(b *structs.Bytes) Bytes {
	return Bytes{b: b}
}

// Internal use only.
func (b Bytes) XXXBytes() *structs.Bytes {
	return b.b
}

// Reset resets all the internal fields to their zero value.
func (b *Bytes) Reset() {
	// Instead of calling the underlying Reset() which breaks reusability,
	// we reinitialize with a new Bytes instance
	b.b = structs.NewBytes()
}

// Len returns the number of items in the list.
func (b Bytes) Len() int {
	return b.b.Len()
}

// Get gets a []byte stored at the index.
func (b Bytes) Get(index int) []byte {
	return b.b.Get(index)
}

// All returns an iterator over all byte slices.
// You should NOT modify the returned []byte slices.
func (b Bytes) All() iter.Seq[[]byte] {
	return b.b.All()
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
// You should NOT modify the returned []byte slices.
func (b Bytes) Range(from, to int) iter.Seq[[]byte] {
	return b.b.Range(from, to)
}

// Set a number in position "index" to "value".
func (b Bytes) Set(index int, value []byte) Bytes {
	b.b.Set(index, value)
	return b
}

// Append appends values to the list of []byte.
func (b Bytes) Append(values ...[]byte) Bytes {
	b.b.Append(values...)
	return b
}

// Slice converts this into a standard [][]byte. The values aren't linked, so changing
// []bool or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (b Bytes) Slice() [][]byte {
	length := b.b.Len()
	if length == 0 {
		return nil
	}
	result := make([][]byte, length)
	for i := 0; i < length; i++ {
		result[i] = b.b.Get(i)
	}
	return result
}

// String represents a list of strings.
type Strings struct {
	b *structs.Bytes
}

// NewString creates a new Strings.
func NewString() Strings {
	return Strings{b: structs.NewBytes()}
}

// Internal use only.
func XXXFromStrings(b *structs.Bytes) Strings {
	return Strings{b: b}
}

// Internal use only.
func (s Strings) XXXBytes() *structs.Bytes {
	return s.b
}

// Reset resets all the internal fields to their zero value.
func (s *Strings) Reset() {
	// Instead of calling the underlying Reset() which breaks reusability,
	// we reinitialize with a new Bytes instance
	s.b = structs.NewBytes()
}

// Len returns the number of items in the list.
func (s Strings) Len() int {
	return s.b.Len()
}

// Get gets a string stored at the index.
func (s Strings) Get(index int) string {
	b := s.b.Get(index)
	if b == nil {
		return ""
	}
	return conversions.ByteSlice2String(b)
}

// All returns an iterator over all strings.
func (s Strings) All() iter.Seq[string] {
	return func(yield func(string) bool) {
		for b := range s.b.All() {
			if !yield(conversions.ByteSlice2String(b)) {
				return
			}
		}
	}
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (s Strings) Range(from, to int) iter.Seq[string] {
	return func(yield func(string) bool) {
		for b := range s.b.Range(from, to) {
			if !yield(conversions.ByteSlice2String(b)) {
				return
			}
		}
	}
}

// Set a number in position "index" to "value".
func (s Strings) Set(index int, value string) Strings {
	s.b.Set(index, conversions.UnsafeGetBytes(value))
	return s
}

// Append appends values to the list of []byte.
func (s Strings) Append(values ...string) Strings {
	x := make([][]byte, len(values))
	for i, v := range values {
		x[i] = conversions.UnsafeGetBytes(v)
	}
	s.b.Append(x...)
	return s
}

// Slice converts this into a standard []string. The values aren't linked, so changing
// []string or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (s Strings) Slice() []string {
	length := s.b.Len()
	if length == 0 {
		return nil
	}
	x := make([]string, length)
	index := 0
	for v := range s.Range(0, length) {
		x[index] = v
		index++
	}
	return x
}

// Enum represents an enum entry in a list.
type Enum interface {
	~uint8 | ~uint16
	String() string
}

// Enums represents a list of enums.
type Enums[E Enum] struct {
	n *structs.Numbers[E]
}

// NewEnums creates a new Enums.
func NewEnums[E Enum]() Enums[E] {
	return Enums[E]{n: structs.NewNumbers[E]()}
}

// Internal use only.
func XXXEnumsFromNumbers[E Enum](n *structs.Numbers[E]) Enums[E] {
	return Enums[E]{n: n}
}

// Internal use only.
func (n Enums[E]) XXXNumbers() *structs.Numbers[E] {
	return n.n
}

// Len returns the number of items in this list.
func (n Enums[E]) Len() int {
	return n.n.Len()
}

// Get gets a number stored at the index.
func (n Enums[E]) Get(index int) E {
	return n.n.Get(index)
}

// All returns an iterator over all enum values.
func (n Enums[E]) All() iter.Seq[E] {
	return n.n.All()
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (n Enums[E]) Range(from, to int) iter.Seq[E] {
	return n.n.Range(from, to)
}

// Set a number in position "index" to "value".
func (n Enums[E]) Set(index int, value E) Enums[E] {
	n.n.Set(index, value)
	return n
}

// Append appends values to the list of numbers.
func (n Enums[E]) Append(i ...E) Enums[E] {
	n.n.Append(i...)
	return n
}

// Slice converts this into a standard []I, where I is a Enum. The values aren't linked, so changing
// []I or calling n.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (n Enums[E]) Slice() []E {
	return n.n.Slice()
}
