package json

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/reflect"
)

type number interface {
	int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

// decoder decodes a JSON representation of a Struct into a reflect.ClawStruct object.
type decoder struct {
	// allowUnknown indicates if we allow unknown fields to be decoded.
	// If so, we store the raw decodes in struct fields XXXUnknown.
	allowUnknown bool

	// descr is the struct description from reflection.
	descr reflect.StructDescr
	// fields is a list of field names to field descriptions.
	fields map[string]reflect.FieldDescr
	// subs is a decoder for sub Structs or []Struct. The decoder
	// will decode a single value in the case of []Struct.
	subs map[string]*decoder

	r *bufio.Reader
}

// newDecoder is the constructor for decoder. This can be reused by
// calling .decode on new data of the same type.
func newDecoder(r io.Reader) *decoder {
	return &decoder{r: bufio.NewReader(r)}
}

// decode is used to decode data in dec into v. It is the entry point for using a decoder.
func (d *decoder) decode(dec *json.Decoder, v reflect.ClawStruct) error {
	clawStruct := v.ClawReflect()
	if d.fields == nil {
		d.prep(clawStruct)
	}

	dec.UseNumber()
	m := map[string]any{}
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return d.decodeStruct(m, clawStruct)
}

// prep is used to setup our attributes for a new decode.
func (d *decoder) prep(v reflect.Struct) {
	d.descr = v.Descriptor()
	d.fields = map[string]reflect.FieldDescr{}
	for _, fd := range v.Descriptor().Fields() {
		d.fields[fd.Name()] = fd
		if fd.Type() == field.FTStruct || fd.Type() == field.FTListStructs {
			f := v.NewField(fd)
			dec := &decoder{}
			dec.prep(f.Struct())
			d.subs[fd.Name()] = dec
		}
	}
}

func (d *decoder) decodeStruct(m map[string]any, r reflect.Struct) error {
	for key, val := range m {
		fd, ok := d.fields[key]
		if ok {
			if d.allowUnknown {
				continue
			}
			return fmt.Errorf("received field %q in Struct %q we don't know", key, d.descr.StructName())
		}
		switch fd.Type() {
		case field.FTBool:
			b, ok := val.(bool)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q that contained %v, not a bool", key, d.descr.StructName(), val)
			}
			r.Set(fd, reflect.ValueOfBool(b))
		case field.FTInt8:
			if err := setNumber[int8](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTInt16:
			if err := setNumber[int16](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTInt32:
			if err := setNumber[int32](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTInt64:
			if err := setNumber[int64](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTUint8:
			if fd.IsEnum() {
				i, err := val.(json.Number).Int64()
				if err != nil {
					return err
				}
				r.Set(fd, reflect.ValueOfEnum(uint8(i)))
			}
			if err := setNumber[uint32](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTUint16:
			if fd.IsEnum() {
				i, err := val.(json.Number).Int64()
				if err != nil {
					return err
				}
				r.Set(fd, reflect.ValueOfEnum(uint16(i)))
			}
			if err := setNumber[uint16](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTUint32:
			if err := setNumber[uint32](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTUint64:
			if err := setNumber[uint64](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTFloat32:
			if err := setNumber[float32](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTFloat64:
			if err := setNumber[float64](fd, val, r); err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
		case field.FTBytes:
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected []byte", key, d.descr.StructName())
			}
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
			}
			r.Set(fd, reflect.ValueOfBytes(b))
		case field.FTString:
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected string", key, d.descr.StructName())
			}
			r.Set(fd, reflect.ValueOfString(s))

		case field.FTListBools:
			l, ok := val.([]any)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected []bool", key, d.descr.StructName())
			}
			b := make([]bool, 0, len(l))
			for _, aItem := range l {
				item, ok := aItem.(bool)
				if !ok {
					return fmt.Errorf("received field %q in Struct %q, but wasn't expected []bool", key, d.descr.StructName())
				}
				b = append(b, item)
			}
			r.Set(fd, reflect.ValueOfList(reflect.ListFrom(b)))
		case field.FTListInt8:
			setListNumber[int8](fd, val, r)
		case field.FTListInt16:
			setListNumber[int16](fd, val, r)
		case field.FTListInt32:
			setListNumber[int32](fd, val, r)
		case field.FTListInt64:
			setListNumber[int64](fd, val, r)
		case field.FTListUint8:
			setListNumber[uint8](fd, val, r)
		case field.FTListUint16:
			setListNumber[uint16](fd, val, r)
		case field.FTListUint32:
			setListNumber[uint32](fd, val, r)
		case field.FTListUint64:
			setListNumber[uint64](fd, val, r)
		case field.FTListFloat32:
			setListNumber[float32](fd, val, r)
		case field.FTListFloat64:
			setListNumber[float64](fd, val, r)
		case field.FTListBytes:
			l, ok := val.([]any)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected [][]byte", key, d.descr.StructName())
			}
			b := make([][]byte, 0, len(l))
			for _, aItem := range l {
				item, ok := aItem.(string)
				if !ok {
					return fmt.Errorf("received field %q in Struct %q, but wasn't expected [][]byte", key, d.descr.StructName())
				}
				data, err := base64.StdEncoding.DecodeString(item)
				if err != nil {
					return fmt.Errorf("received field %q in Struct %q, %w", key, d.descr.StructName(), err)
				}
				b = append(b, data)
			}
			r.Set(fd, reflect.ValueOfList(reflect.ListFrom(b)))
		case field.FTListStrings:
			l, ok := val.([]any)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected []string", key, d.descr.StructName())
			}
			b := make([]string, 0, len(l))
			for _, aItem := range l {
				item, ok := aItem.(string)
				if !ok {
					return fmt.Errorf("received field %q in Struct %q, but wasn't expected []string", key, d.descr.StructName())
				}
				b = append(b, item)
			}
			r.Set(fd, reflect.ValueOfList(reflect.ListFrom(b)))
		case field.FTStruct:
			m, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected Struct", key, d.descr.StructName())
			}
			v := r.NewField(fd)
			if err := d.subs[fd.Name()].decodeStruct(m, v.Struct()); err != nil {
				return fmt.Errorf("received field %q in Struct %q, but had problem decoding it: %s", key, d.descr.StructName(), err)
			}
			r.Set(fd, v)
		case field.FTListStructs:
			l, ok := val.([]any)
			if !ok {
				return fmt.Errorf("received field %q in Struct %q, but wasn't expected []Struct", key, d.descr.StructName())
			}
			n := r.NewField(fd).List()

			// l is a []any, with v being a map[string]any.
			for _, v := range l {
				s := n.New()
				m := v.(map[string]any)
				if err := d.subs[fd.Name()].decodeStruct(m, s); err != nil {
					return fmt.Errorf("received field %q in []Struct %q, but had problem decoding it: %s", key, d.descr.StructName(), err)
				}
				n.Append(reflect.ValueOfStruct(s))
			}
			r.Set(fd, reflect.ValueOfList(n))
		default:
			return fmt.Errorf("problem: encountered unsupported field type: %s", fd.Type())
		}
	}
	panic("shoud never get here")
}

func setNumber[N number](fd reflect.FieldDescr, val any, r reflect.Struct) error {
	n, ok := val.(json.Number)
	if !ok {
		return fmt.Errorf("not an int8", val)
	}
	var t N
	switch any(t).(type) {
	case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		i, err := n.Int64()
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		r.Set(fd, reflect.ValueOfNumber(N(i)))
	case float32, float64:
		f, err := n.Float64()
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		r.Set(fd, reflect.ValueOfNumber(N(f)))
	default:
		panic("passed some new special number type???")
	}
	return nil
}

func setListNumber[N number](fd reflect.FieldDescr, val any, r reflect.Struct) error {
	l, ok := val.([]any)
	if !ok {
		return fmt.Errorf("was not a []Number")
	}

	var t N
	b := make([]N, 0, len(l))

	for _, aItem := range l {
		n, ok := aItem.(json.Number)
		if !ok {
			return fmt.Errorf("list item expected json.Number, but got %T", aItem)
		}

		switch any(t).(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			i, err := n.Int64()
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}
			b = append(b, N(i))
		case float32, float64:
			f, err := n.Float64()
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}
			b = append(b, N(f))
		default:
			panic("passed some new special number type???")
		}
	}
	r.Set(fd, reflect.ValueOfList(reflect.ListFrom(b)))
	return nil
}
