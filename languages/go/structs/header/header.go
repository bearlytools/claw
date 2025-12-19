package header

import (
	"fmt"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/field"
)

const maxDataSize = 1099511627775

// Masks to use to pull information from a bitpacked uint64.
var (
	dataSizeMask = bits.Mask[uint64](24, 64)
)

// Generic is the header of struct.
type Generic []byte

func New() Generic {
	return Generic(make([]byte, 8))
}

// FieldNum returns the field number that the entry the header represents is set to.
func (g Generic) FieldNum() uint16 {
	return binary.Get[uint16](g[:2])
}

// SetFieldNum sets the field number in the header.
func (g Generic) SetFieldNum(u uint16) {
	binary.Put(g[:2], u)
}

// FieldType returns the type of field the header is for.
func (g Generic) FieldType() field.Type {
	return field.Type(g[2])
}

// SetFieldType sets the field type the header is for.
func (g Generic) SetFieldType(u field.Type) {
	g[2] = byte(u)
}

// Final40 returns the value of the final 40 bits. This is usually used to store either the size of
// an entry or the number of items.
func (g Generic) Final40() uint64 {
	u := binary.Get[uint64](g)
	return bits.GetValue[uint64, uint64](u, dataSizeMask, 24)
}

// SetFinal40 sets the final 40 bits in the header.
func (g Generic) SetFinal40(u uint64) {
	if u > maxDataSize {
		panic(fmt.Sprintf("can't put %d in a 40bit register, max value is 1099511627775", u))
	}
	bits.ClearBytes(g[0:8], 3, 8)
	n := conversions.BytesToNum[uint64](g[0:8])
	*n = bits.SetValue(u, *n, 24, 64)
}
