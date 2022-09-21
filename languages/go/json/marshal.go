package json

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/reflect"
)

// Array is used to write out an array of JSON objects.
type Array struct {
	options Options
	w       *bufio.Writer
	entries int
}

// NewArray creates a new Array writer that writes to w. You can reuse the Array instead
// of creating a new one by using .Reset().
func NewArray(options Options, w io.Writer) (*Array, error) {
	a := &Array{options: options}
	if err := a.Reset(w); err != nil {
		return nil, err
	}
	return a, nil
}

// Reset resets the Array with a new io.Writer.
func (a *Array) Reset(w io.Writer) error {
	a.w = bufio.NewWriter(w)
	return nil
}

// Write writes the Struct to the io.Writer.
func (a *Array) Write(v reflect.ClawStruct) error {
	err := a.options.write(a.w, v, a.entries)
	if err != nil {
		return err
	}
	a.entries++
	return nil
}

// Close closes the Array for writing and writes out the closing }.
func (a *Array) Close() error {
	a.w.Write([]byte(`]`))
	return a.w.Flush()
}

// Options provides options for writing Claw output to JSON.
type Options struct {
	// UseEnumNumbers emits enum values as numbers.
	UseEnumNumbers bool
}

func (o *Options) write(w *bufio.Writer, v reflect.ClawStruct, entry int) error {
	if entry == 0 {
		w.Write([]byte(`[`))
	} else {
		w.WriteRune(',')
	}
	if err := o.writeStruct(w, v.ClawStruct()); err != nil {
		return err
	}
	return nil
}

func (o Options) writeStruct(w *bufio.Writer, r reflect.Struct) error {
	i := -1
	w.WriteRune('{')

	var err error
	r.Range(
		func(fd reflect.FieldDescr, v reflect.Value) bool {
			if v == nil { // Value wasn't set.
				return true
			}

			i++
			if i != 0 {
				w.WriteRune(',')
			}
			writeFieldName(fd, w)

			switch fd.Type() {
			case field.FTBool:
				writeBool(v.Bool(), w)
			case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64:
				writeInt(w, v.Int())
			case field.FTUint8, field.FTUint16:
				if fd.IsEnum() && o.UseEnumNumbers {
					writeString(w, v.Enum().Name())
				} else {
					writeUint(w, v.Uint())
				}
			case field.FTUint32, field.FTUint64:
				writeUint(w, v.Uint())
			case field.FTFloat32:
				writeFloat(w, 32, v.Float())
			case field.FTFloat64:
				writeFloat(w, 64, v.Float())
			case field.FTBytes:
				writeBytes(w, v.Bytes())
			case field.FTString:
				writeString(w, v.String())
			case field.FTStruct:
				err = o.writeStruct(w, v.Struct())
				if err != nil {
					return false
				}
			case field.FTListBools:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeBool(v.Bool(), w)
				}
				w.WriteRune(']')
			case field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeInt(w, v.Int())

				}
				w.WriteRune(']')
			case field.FTListUint8, field.FTListUint16:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					if fd.IsEnum() && o.UseEnumNumbers {
						writeUint(w, v.Uint())
					} else {
						writeString(w, v.Enum().Name())
					}
				}
				w.WriteRune(']')
			case field.FTListUint32, field.FTListUint64:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeUint(w, v.Uint())
				}
				w.WriteRune(']')
			case field.FTListFloat32:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeFloat(w, 32, v.Float())
				}
				w.WriteRune(']')
			case field.FTListFloat64:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeFloat(w, 32, v.Float())
				}
				w.WriteRune(']')
			case field.FTListBytes:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeBytes(w, v.Bytes())
				}
				w.WriteRune(']')
			case field.FTListStrings:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					writeString(w, v.String())
				}
				w.WriteRune(']')
			case field.FTListStructs:
				l := v.List()
				w.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						w.WriteRune(',')
					}
					v := l.Get(i)
					if err = o.writeStruct(w, v.Struct()); err != nil {
						return false
					}
				}
				w.WriteRune(']')
			default:
				err = fmt.Errorf("problem: encountered unsupported field type: %s", fd.Type())
				return false
			}
			return true
		},
	)
	if err != nil {
		return err
	}

	w.WriteRune('}')
	return w.Flush()
}

func writeFieldName(fd reflect.FieldDescr, w *bufio.Writer) {
	w.WriteRune('"')
	w.WriteString(fd.Name())
	w.WriteString(`":`)
}

func writeBool(b bool, w *bufio.Writer) {
	if b {
		w.WriteString("true")
		return
	}
	w.WriteString("false")
}

func writeInt(w *bufio.Writer, n int64) {
	w.WriteString(strconv.FormatInt(n, 10))
}

func writeUint(w *bufio.Writer, n uint64) {
	w.WriteString(strconv.FormatUint(n, 10))
}

// writeFloat writes the floating point value to buff in string format.
// This code is borrowed from protojson, which borrowed it from encoding/json.
func writeFloat(w *bufio.Writer, bitSize uint8, n float64) {
	switch {
	case math.IsNaN(n):
		w.WriteString(`"NaN"`)
		return
	case math.IsInf(n, +1):
		w.WriteString(`"Infinity"`)
		return
	case math.IsInf(n, -1):
		w.WriteString(`"-Infinity"`)
		return
	}

	// JSON number formatting logic based on encoding/json.
	// See floatEncoder.encode for reference.
	fmt := byte('f')
	if abs := math.Abs(n); abs != 0 {
		if bitSize == 64 && (abs < 1e-6 || abs >= 1e21) ||
			bitSize == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			fmt = 'e'
		}
	}
	out := strconv.AppendFloat([]byte{}, n, fmt, -1, int(bitSize))
	if fmt == 'e' {
		n := len(out)
		if n >= 4 && out[n-4] == 'e' && out[n-3] == '-' && out[n-2] == '0' {
			out[n-2] = out[n-1]
			out = out[:n-1]
		}
	}
	w.Write(out)
}

func writeBytes(w *bufio.Writer, b []byte) {
	encoded := base64.StdEncoding.EncodeToString(b)
	w.WriteString(`"` + encoded + `"`)
}

func writeString(w *bufio.Writer, s string) {
	w.WriteString(`"` + s + `"`)
}
