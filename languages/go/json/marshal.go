package json

import (
	"bytes"
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
	w       io.Writer
	total   int
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
	a.w = w
	n, err := w.Write([]byte(`{`))
	if err != nil {
		return err
	}
	a.total = n
	return nil
}

// Write writes the Struct to the io.Writer.
func (a *Array) Write(v reflect.ClawStruct) error {
	n, err := a.options.Write(a.w, v)
	if err != nil {
		return err
	}
	a.total += n
	return nil
}

// Close closes the Array for writing and writes out the closing }.
func (a *Array) Close() (n int, err error) {
	n, err = a.w.Write([]byte(`}`))
	a.total += n
	return a.total, err
}

// Options provides options for writing Claw output to JSON.
type Options struct {
	// UseEnumNumbers emits enum values as numbers.
	UseEnumNumbers bool
}

func (o *Options) Write(w io.Writer, v reflect.ClawStruct) (n int, err error) {
	return o.writeStruct(v.ClawReflect(), w)
}

func (o Options) writeStruct(r reflect.Struct, w io.Writer) (n int, err error) {
	buff := &bytes.Buffer{}
	i := -1
	buff.WriteRune('{')

	r.Range(
		func(fd reflect.FieldDescr, v reflect.Value) bool {
			i++
			if i != 0 {
				buff.WriteRune(',')
			}
			writeFieldName(fd, buff)

			switch fd.Type() {
			case field.FTBool:
				writeBool(v.Bool(), buff)
			case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64:
				writeInt(buff, v.Int())
			case field.FTUint8, field.FTUint16:
				if fd.IsEnum() {
					if o.UseEnumNumbers {
						writeUint(buff, v.Uint())
					} else {
						writeString(buff, v.Enum().Descriptor().Name())
					}
				} else {
					writeUint(buff, v.Uint())
				}
			case field.FTUint32, field.FTUint64:
				writeUint(buff, v.Uint())
			case field.FTFloat32:
				writeFloat(buff, 32, v.Float())
			case field.FTFloat64:
				writeFloat(buff, 64, v.Float())
			case field.FTBytes:
				writeBytes(buff, v.Bytes())
			case field.FTString:
				writeString(buff, v.String())
			case field.FTStruct:
				_, err = o.writeStruct(v.Struct(), w)
				if err != nil {
					return false
				}
			case field.FTListBools:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeBool(v.Bool(), buff)
				}
				buff.WriteRune(']')
			case field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeInt(buff, v.Int())

				}
				buff.WriteRune(']')
			case field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeUint(buff, v.Uint())

				}
				buff.WriteRune(']')
			case field.FTListFloat32:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeFloat(buff, 32, v.Float())
				}
				buff.WriteRune(']')
			case field.FTListFloat64:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeFloat(buff, 32, v.Float())
				}
				buff.WriteRune(']')
			case field.FTListBytes:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeBytes(buff, v.Bytes())
				}
				buff.WriteRune(']')
			case field.FTListStrings:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					writeString(buff, v.String())
				}
				buff.WriteRune(']')
			case field.FTListStructs:
				l := v.List()
				buff.WriteRune('[')
				for i := 0; i < l.Len(); i++ {
					if i < 0 {
						buff.WriteRune(',')
					}
					v := l.Get(i)
					if _, err = o.writeStruct(v.Struct(), w); err != nil {
						return false
					}
				}
				buff.WriteRune(']')
			default:
				err = fmt.Errorf("problem: encountered unsupported field type: %s", fd.Type())
				return false
			}
			return true
		},
	)
	if err != nil {
		return 0, err
	}

	buff.WriteRune('}')
	_, err = w.Write(buff.Bytes())
	return buff.Len(), err
}

func writeFieldName(fd reflect.FieldDescr, buff *bytes.Buffer) {
	buff.WriteRune('"')
	buff.WriteString(fd.Name())
	buff.WriteString(`":`)
}

func writeBool(b bool, buff *bytes.Buffer) {
	if b {
		buff.WriteString("true")
		return
	}
	buff.WriteString("false")
}

func writeInt(buff *bytes.Buffer, n int64) {
	buff.WriteString(strconv.FormatInt(n, 10))
}

func writeUint(buff *bytes.Buffer, n uint64) {
	buff.WriteString(strconv.FormatUint(n, 10))
}

// writeFloat writes the floating point value to buff in string format.
// This code is borrowed from protojson, which borrowed it from encoding/json.
func writeFloat(buff *bytes.Buffer, bitSize uint8, n float64) {
	switch {
	case math.IsNaN(n):
		buff.WriteString(`"NaN"`)
		return
	case math.IsInf(n, +1):
		buff.WriteString(`"Infinity"`)
		return
	case math.IsInf(n, -1):
		buff.WriteString(`"-Infinity"`)
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
	buff.Write(out)
}

func writeBytes(buff *bytes.Buffer, b []byte) {
	encoded := base64.StdEncoding.EncodeToString(b)
	buff.WriteString(`"` + encoded + `"`)
}

func writeString(buff *bytes.Buffer, s string) {
	buff.WriteString(`"` + s + `"`)
}
