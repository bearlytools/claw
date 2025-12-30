package segment

import (
	"cmp"
	"encoding/binary"
	"iter"
	"math"
	"reflect"
	"slices"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// MapKey is a constraint for valid map key types.
// All comparable scalar types are allowed.
type MapKey interface {
	~string | ~bool |
		~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// MapValue is a constraint for valid map value types.
// This includes all scalars, string, bytes, and nested maps (any).
type MapValue interface {
	~string | ~bool | ~[]byte |
		~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Maps represents a map with sorted keys for O(log n) lookups.
// Keys are maintained in sorted order for deterministic encoding and binary search.
type Maps[K MapKey, V any] struct {
	parent       *Struct
	fieldNum     uint16
	keyType      field.Type
	valType      field.Type
	keys         []K
	values       []V
	dirty        bool
	valueMapping *mapping.Map // for struct values
}

// MapSyncer interface for maps that can sync to segment.
type MapSyncer interface {
	SyncToSegment() error
}

// NewMaps creates a new Maps attached to a struct field.
func NewMaps[K MapKey, V any](parent *Struct, fieldNum uint16, keyType, valType field.Type, valueMapping *mapping.Map) *Maps[K, V] {
	m := &Maps[K, V]{
		parent:       parent,
		fieldNum:     fieldNum,
		keyType:      keyType,
		valType:      valType,
		keys:         make([]K, 0, 8),
		values:       make([]V, 0, 8),
		valueMapping: valueMapping,
	}
	parent.SetList(fieldNum, m)
	parent.RegisterDirtyList(fieldNum, m)
	return m
}

// Len returns the number of entries in the map.
func (m *Maps[K, V]) Len() int {
	return len(m.keys)
}

// Get returns the value for the given key and whether it exists.
func (m *Maps[K, V]) Get(key K) (V, bool) {
	idx := m.findKey(key)
	if idx >= 0 {
		return m.values[idx], true
	}
	var zero V
	return zero, false
}

// Set sets a key-value pair, maintaining sorted order.
func (m *Maps[K, V]) Set(key K, value V) {
	idx := m.findKey(key)
	if idx >= 0 {
		// Key exists, update value
		m.values[idx] = value
	} else {
		// Key doesn't exist, insert in sorted position
		insertPos := m.findInsertPos(key)
		m.keys = slices.Insert(m.keys, insertPos, key)
		m.values = slices.Insert(m.values, insertPos, value)
	}
	m.dirty = true

	if m.parent.recording {
		m.parent.RecordOp(RecordedOp{
			FieldNum: m.fieldNum,
			OpType:   OpMapSet,
			Index:    NoListIndex,
			Data:     m.encodeKeyValue(key, value),
		})
	}
}

// Delete removes a key from the map.
func (m *Maps[K, V]) Delete(key K) {
	idx := m.findKey(key)
	if idx >= 0 {
		m.keys = slices.Delete(m.keys, idx, idx+1)
		m.values = slices.Delete(m.values, idx, idx+1)
		m.dirty = true

		if m.parent.recording {
			m.parent.RecordOp(RecordedOp{
				FieldNum: m.fieldNum,
				OpType:   OpMapDelete,
				Index:    NoListIndex,
				Data:     m.encodeKey(key),
			})
		}
	}
}

// Has returns true if the key exists in the map.
func (m *Maps[K, V]) Has(key K) bool {
	return m.findKey(key) >= 0
}

// Keys returns all keys in sorted order.
func (m *Maps[K, V]) Keys() []K {
	result := make([]K, len(m.keys))
	copy(result, m.keys)
	return result
}

// Values returns all values in key-sorted order.
func (m *Maps[K, V]) Values() []V {
	result := make([]V, len(m.values))
	copy(result, m.values)
	return result
}

// All returns an iterator over all key-value pairs in sorted order.
func (m *Maps[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i := 0; i < len(m.keys); i++ {
			if !yield(m.keys[i], m.values[i]) {
				return
			}
		}
	}
}

// Clear removes all entries from the map.
func (m *Maps[K, V]) Clear() {
	if len(m.keys) > 0 {
		m.keys = m.keys[:0]
		m.values = m.values[:0]
		m.dirty = true

		if m.parent.recording {
			m.parent.RecordOp(RecordedOp{
				FieldNum: m.fieldNum,
				OpType:   OpClear,
				Index:    NoListIndex,
			})
		}
	}
}

// findKey returns the index of the key, or -1 if not found.
// Uses binary search for O(log n) lookups.
func (m *Maps[K, V]) findKey(key K) int {
	idx, found := slices.BinarySearchFunc(m.keys, key, compareKeys[K])
	if found {
		return idx
	}
	return -1
}

// findInsertPos returns the position where a key should be inserted.
func (m *Maps[K, V]) findInsertPos(key K) int {
	idx, _ := slices.BinarySearchFunc(m.keys, key, compareKeys[K])
	return idx
}

// compareKeys compares two keys for sorting.
func compareKeys[K MapKey](a, b K) int {
	// Use type assertion to handle comparison
	switch av := any(a).(type) {
	case string:
		return cmp.Compare(av, any(b).(string))
	case bool:
		ab, bb := av, any(b).(bool)
		if ab == bb {
			return 0
		}
		if !ab {
			return -1
		}
		return 1
	case int8:
		return cmp.Compare(av, any(b).(int8))
	case int16:
		return cmp.Compare(av, any(b).(int16))
	case int32:
		return cmp.Compare(av, any(b).(int32))
	case int64:
		return cmp.Compare(av, any(b).(int64))
	case uint8:
		return cmp.Compare(av, any(b).(uint8))
	case uint16:
		return cmp.Compare(av, any(b).(uint16))
	case uint32:
		return cmp.Compare(av, any(b).(uint32))
	case uint64:
		return cmp.Compare(av, any(b).(uint64))
	case float32:
		return cmp.Compare(av, any(b).(float32))
	case float64:
		return cmp.Compare(av, any(b).(float64))
	default:
		// For custom types based on the allowed underlying types
		va := reflect.ValueOf(a)
		vb := reflect.ValueOf(b)
		switch va.Kind() {
		case reflect.String:
			return cmp.Compare(va.String(), vb.String())
		case reflect.Bool:
			ab, bb := va.Bool(), vb.Bool()
			if ab == bb {
				return 0
			}
			if !ab {
				return -1
			}
			return 1
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return cmp.Compare(va.Int(), vb.Int())
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return cmp.Compare(va.Uint(), vb.Uint())
		case reflect.Float32, reflect.Float64:
			return cmp.Compare(va.Float(), vb.Float())
		}
		return 0
	}
}

// SyncToSegment syncs the map data to the parent struct's segment.
func (m *Maps[K, V]) SyncToSegment() error {
	if !m.dirty {
		return nil
	}

	if len(m.keys) == 0 {
		m.parent.removeField(m.fieldNum)
		m.dirty = false
		return nil
	}

	// Calculate total size
	dataSize := m.calculateDataSize()
	padding := paddingNeeded(dataSize)
	totalSize := HeaderSize + dataSize + padding

	// Create field data
	fieldData := make([]byte, totalSize)

	// Write map header
	EncodeMapHeader(fieldData[:HeaderSize], m.fieldNum, m.keyType, m.valType, uint32(totalSize))

	// Write entries
	offset := HeaderSize
	for i := 0; i < len(m.keys); i++ {
		offset += m.encodeKeyAt(fieldData, offset, m.keys[i])
		offset += m.encodeValueAt(fieldData, offset, m.values[i])
	}

	m.parent.insertField(m.fieldNum, fieldData)
	m.dirty = false
	return nil
}

// calculateDataSize calculates the total size of all key-value pairs.
func (m *Maps[K, V]) calculateDataSize() int {
	size := 0
	for i := 0; i < len(m.keys); i++ {
		size += m.keySize(m.keys[i])
		size += m.valueSize(m.values[i])
	}
	return size
}

// keySize returns the encoded size of a key.
func (m *Maps[K, V]) keySize(key K) int {
	switch m.keyType {
	case field.FTString:
		// 4-byte length + string data
		return 4 + len(any(key).(string))
	case field.FTBool, field.FTInt8, field.FTUint8:
		return 1
	case field.FTInt16, field.FTUint16:
		return 2
	case field.FTInt32, field.FTUint32, field.FTFloat32:
		return 4
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		return 8
	default:
		return 0
	}
}

// valueSize returns the encoded size of a value.
func (m *Maps[K, V]) valueSize(value V) int {
	switch m.valType {
	case field.FTString:
		// 4-byte length + string data
		return 4 + len(any(value).(string))
	case field.FTBytes:
		// 4-byte length + byte data
		return 4 + len(any(value).([]byte))
	case field.FTBool, field.FTInt8, field.FTUint8:
		return 1
	case field.FTInt16, field.FTUint16:
		return 2
	case field.FTInt32, field.FTUint32, field.FTFloat32:
		return 4
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		return 8
	case field.FTStruct:
		// Struct: header + data
		s := any(value).(*Struct)
		s.syncDirtyLists()
		return s.seg.Len()
	case field.FTMap:
		// Nested map - need to sync and get size
		if syncer, ok := any(value).(MapSyncer); ok {
			syncer.SyncToSegment()
		}
		// For nested maps, we'd need the actual size from the encoded data
		// This is a placeholder - actual implementation needs nested map handling
		return 0
	default:
		return 0
	}
}

// encodeKeyAt encodes a key at the given offset and returns bytes written.
func (m *Maps[K, V]) encodeKeyAt(buf []byte, offset int, key K) int {
	kv := reflect.ValueOf(key)
	switch m.keyType {
	case field.FTString:
		s := kv.String()
		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(s)))
		copy(buf[offset+4:], s)
		return 4 + len(s)
	case field.FTBool:
		if kv.Bool() {
			buf[offset] = 1
		} else {
			buf[offset] = 0
		}
		return 1
	case field.FTInt8:
		buf[offset] = byte(kv.Int())
		return 1
	case field.FTUint8:
		buf[offset] = byte(kv.Uint())
		return 1
	case field.FTInt16:
		binary.LittleEndian.PutUint16(buf[offset:], uint16(kv.Int()))
		return 2
	case field.FTUint16:
		binary.LittleEndian.PutUint16(buf[offset:], uint16(kv.Uint()))
		return 2
	case field.FTInt32:
		binary.LittleEndian.PutUint32(buf[offset:], uint32(kv.Int()))
		return 4
	case field.FTUint32:
		binary.LittleEndian.PutUint32(buf[offset:], uint32(kv.Uint()))
		return 4
	case field.FTFloat32:
		binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(float32(kv.Float())))
		return 4
	case field.FTInt64:
		binary.LittleEndian.PutUint64(buf[offset:], uint64(kv.Int()))
		return 8
	case field.FTUint64:
		binary.LittleEndian.PutUint64(buf[offset:], kv.Uint())
		return 8
	case field.FTFloat64:
		binary.LittleEndian.PutUint64(buf[offset:], math.Float64bits(kv.Float()))
		return 8
	}
	return 0
}

// encodeValueAt encodes a value at the given offset and returns bytes written.
func (m *Maps[K, V]) encodeValueAt(buf []byte, offset int, value V) int {
	vv := reflect.ValueOf(value)
	switch m.valType {
	case field.FTString:
		s := vv.String()
		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(s)))
		copy(buf[offset+4:], s)
		return 4 + len(s)
	case field.FTBytes:
		b := vv.Bytes()
		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(b)))
		copy(buf[offset+4:], b)
		return 4 + len(b)
	case field.FTBool:
		if vv.Bool() {
			buf[offset] = 1
		} else {
			buf[offset] = 0
		}
		return 1
	case field.FTInt8:
		buf[offset] = byte(vv.Int())
		return 1
	case field.FTUint8:
		buf[offset] = byte(vv.Uint())
		return 1
	case field.FTInt16:
		binary.LittleEndian.PutUint16(buf[offset:], uint16(vv.Int()))
		return 2
	case field.FTUint16:
		binary.LittleEndian.PutUint16(buf[offset:], uint16(vv.Uint()))
		return 2
	case field.FTInt32:
		binary.LittleEndian.PutUint32(buf[offset:], uint32(vv.Int()))
		return 4
	case field.FTUint32:
		binary.LittleEndian.PutUint32(buf[offset:], uint32(vv.Uint()))
		return 4
	case field.FTFloat32:
		binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(float32(vv.Float())))
		return 4
	case field.FTInt64:
		binary.LittleEndian.PutUint64(buf[offset:], uint64(vv.Int()))
		return 8
	case field.FTUint64:
		binary.LittleEndian.PutUint64(buf[offset:], vv.Uint())
		return 8
	case field.FTFloat64:
		binary.LittleEndian.PutUint64(buf[offset:], math.Float64bits(vv.Float()))
		return 8
	case field.FTStruct:
		s := any(value).(*Struct)
		s.syncDirtyLists()
		copy(buf[offset:], s.seg.data)
		return s.seg.Len()
	}
	return 0
}

// encodeKey encodes a key for patch recording.
func (m *Maps[K, V]) encodeKey(key K) []byte {
	size := m.keySize(key)
	buf := make([]byte, size)
	m.encodeKeyAt(buf, 0, key)
	return buf
}

// encodeKeyValue encodes a key-value pair for patch recording.
func (m *Maps[K, V]) encodeKeyValue(key K, value V) []byte {
	keySize := m.keySize(key)
	valSize := m.valueSize(value)
	buf := make([]byte, keySize+valSize)
	offset := m.encodeKeyAt(buf, 0, key)
	m.encodeValueAt(buf, offset, value)
	return buf
}

// ParseMapFromSegment parses a map from segment data.
// Returns the keys and values slices.
func ParseMapFromSegment[K MapKey, V any](data []byte, keyType, valType field.Type, valueMapping *mapping.Map) ([]K, []V) {
	if len(data) < HeaderSize {
		return nil, nil
	}

	keys := make([]K, 0)
	values := make([]V, 0)

	pos := HeaderSize
	for pos < len(data) {
		// Decode key
		key, keyLen := decodeKey[K](data[pos:], keyType)
		if keyLen == 0 {
			break
		}
		pos += keyLen

		// Decode value
		value, valLen := decodeValue[V](data[pos:], valType, valueMapping)
		if valLen == 0 {
			break
		}
		pos += valLen

		keys = append(keys, key)
		values = append(values, value)
	}

	return keys, values
}

// decodeKey decodes a key from bytes and returns the key and bytes consumed.
func decodeKey[K MapKey](data []byte, keyType field.Type) (K, int) {
	var zero K
	if len(data) == 0 {
		return zero, 0
	}

	var result any
	var size int

	switch keyType {
	case field.FTString:
		if len(data) < 4 {
			return zero, 0
		}
		strLen := int(binary.LittleEndian.Uint32(data))
		if len(data) < 4+strLen {
			return zero, 0
		}
		result = string(data[4 : 4+strLen])
		size = 4 + strLen
	case field.FTBool:
		result = data[0] != 0
		size = 1
	case field.FTInt8:
		result = int8(data[0])
		size = 1
	case field.FTUint8:
		result = data[0]
		size = 1
	case field.FTInt16:
		if len(data) < 2 {
			return zero, 0
		}
		result = int16(binary.LittleEndian.Uint16(data))
		size = 2
	case field.FTUint16:
		if len(data) < 2 {
			return zero, 0
		}
		result = binary.LittleEndian.Uint16(data)
		size = 2
	case field.FTInt32:
		if len(data) < 4 {
			return zero, 0
		}
		result = int32(binary.LittleEndian.Uint32(data))
		size = 4
	case field.FTUint32:
		if len(data) < 4 {
			return zero, 0
		}
		result = binary.LittleEndian.Uint32(data)
		size = 4
	case field.FTFloat32:
		if len(data) < 4 {
			return zero, 0
		}
		result = math.Float32frombits(binary.LittleEndian.Uint32(data))
		size = 4
	case field.FTInt64:
		if len(data) < 8 {
			return zero, 0
		}
		result = int64(binary.LittleEndian.Uint64(data))
		size = 8
	case field.FTUint64:
		if len(data) < 8 {
			return zero, 0
		}
		result = binary.LittleEndian.Uint64(data)
		size = 8
	case field.FTFloat64:
		if len(data) < 8 {
			return zero, 0
		}
		result = math.Float64frombits(binary.LittleEndian.Uint64(data))
		size = 8
	default:
		return zero, 0
	}

	// Convert to target type K
	rv := reflect.ValueOf(&zero).Elem()
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(result.(string))
	case reflect.Bool:
		rv.SetBool(result.(bool))
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := result.(type) {
		case int8:
			rv.SetInt(int64(v))
		case int16:
			rv.SetInt(int64(v))
		case int32:
			rv.SetInt(int64(v))
		case int64:
			rv.SetInt(v)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := result.(type) {
		case uint8:
			rv.SetUint(uint64(v))
		case uint16:
			rv.SetUint(uint64(v))
		case uint32:
			rv.SetUint(uint64(v))
		case uint64:
			rv.SetUint(v)
		}
	case reflect.Float32, reflect.Float64:
		switch v := result.(type) {
		case float32:
			rv.SetFloat(float64(v))
		case float64:
			rv.SetFloat(v)
		}
	}

	return zero, size
}

// decodeValue decodes a value from bytes and returns the value and bytes consumed.
func decodeValue[V any](data []byte, valType field.Type, valueMapping *mapping.Map) (V, int) {
	var zero V
	if len(data) == 0 {
		return zero, 0
	}

	var result any
	var size int

	switch valType {
	case field.FTString:
		if len(data) < 4 {
			return zero, 0
		}
		strLen := int(binary.LittleEndian.Uint32(data))
		if len(data) < 4+strLen {
			return zero, 0
		}
		result = string(data[4 : 4+strLen])
		size = 4 + strLen
	case field.FTBytes:
		if len(data) < 4 {
			return zero, 0
		}
		byteLen := int(binary.LittleEndian.Uint32(data))
		if len(data) < 4+byteLen {
			return zero, 0
		}
		b := make([]byte, byteLen)
		copy(b, data[4:4+byteLen])
		result = b
		size = 4 + byteLen
	case field.FTBool:
		result = data[0] != 0
		size = 1
	case field.FTInt8:
		result = int8(data[0])
		size = 1
	case field.FTUint8:
		result = data[0]
		size = 1
	case field.FTInt16:
		if len(data) < 2 {
			return zero, 0
		}
		result = int16(binary.LittleEndian.Uint16(data))
		size = 2
	case field.FTUint16:
		if len(data) < 2 {
			return zero, 0
		}
		result = binary.LittleEndian.Uint16(data)
		size = 2
	case field.FTInt32:
		if len(data) < 4 {
			return zero, 0
		}
		result = int32(binary.LittleEndian.Uint32(data))
		size = 4
	case field.FTUint32:
		if len(data) < 4 {
			return zero, 0
		}
		result = binary.LittleEndian.Uint32(data)
		size = 4
	case field.FTFloat32:
		if len(data) < 4 {
			return zero, 0
		}
		result = math.Float32frombits(binary.LittleEndian.Uint32(data))
		size = 4
	case field.FTInt64:
		if len(data) < 8 {
			return zero, 0
		}
		result = int64(binary.LittleEndian.Uint64(data))
		size = 8
	case field.FTUint64:
		if len(data) < 8 {
			return zero, 0
		}
		result = binary.LittleEndian.Uint64(data)
		size = 8
	case field.FTFloat64:
		if len(data) < 8 {
			return zero, 0
		}
		result = math.Float64frombits(binary.LittleEndian.Uint64(data))
		size = 8
	case field.FTStruct:
		if len(data) < HeaderSize {
			return zero, 0
		}
		// Decode struct size from header
		_, _, final40 := DecodeHeader(data[:HeaderSize])
		structSize := int(final40)
		if structSize < HeaderSize || len(data) < structSize {
			return zero, 0
		}
		s := &Struct{
			seg:        &Segment{data: data[:structSize]},
			mapping:    valueMapping,
			fieldIndex: make([]fieldEntry, len(valueMapping.Fields)),
		}
		parseFieldIndex(s)
		result = s
		size = structSize
	default:
		return zero, 0
	}

	// Convert to target type V using type assertion
	if v, ok := result.(V); ok {
		return v, size
	}

	// Try reflection-based conversion for custom types
	rv := reflect.ValueOf(&zero).Elem()
	switch rv.Kind() {
	case reflect.String:
		if s, ok := result.(string); ok {
			rv.SetString(s)
		}
	case reflect.Bool:
		if b, ok := result.(bool); ok {
			rv.SetBool(b)
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := result.(type) {
		case int8:
			rv.SetInt(int64(v))
		case int16:
			rv.SetInt(int64(v))
		case int32:
			rv.SetInt(int64(v))
		case int64:
			rv.SetInt(v)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := result.(type) {
		case uint8:
			rv.SetUint(uint64(v))
		case uint16:
			rv.SetUint(uint64(v))
		case uint32:
			rv.SetUint(uint64(v))
		case uint64:
			rv.SetUint(v)
		}
	case reflect.Float32, reflect.Float64:
		switch v := result.(type) {
		case float32:
			rv.SetFloat(float64(v))
		case float64:
			rv.SetFloat(v)
		}
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			if b, ok := result.([]byte); ok {
				rv.SetBytes(b)
			}
		}
	case reflect.Ptr:
		if s, ok := result.(*Struct); ok {
			rv.Set(reflect.ValueOf(s))
		}
	}

	return zero, size
}

// GetMapScalar returns a Maps for scalar key and scalar/string/bytes value types.
func GetMapScalar[K MapKey, V MapValue](s *Struct, fieldNum uint16, keyType, valType field.Type) *Maps[K, V] {
	// Check if we already have this map cached
	if existing := s.GetList(fieldNum); existing != nil {
		if m, ok := existing.(*Maps[K, V]); ok {
			return m
		}
	}

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Parse from segment data
	data := s.seg.data[offset : offset+size]
	keys, values := ParseMapFromSegment[K, V](data, keyType, valType, nil)

	m := &Maps[K, V]{
		parent:   s,
		fieldNum: fieldNum,
		keyType:  keyType,
		valType:  valType,
		keys:     keys,
		values:   values,
		dirty:    false,
	}
	s.SetList(fieldNum, m)
	return m
}

// GetMapStruct returns a Maps for scalar key and struct value types.
func GetMapStruct[K MapKey](s *Struct, fieldNum uint16, keyType field.Type, valueMapping *mapping.Map) *Maps[K, *Struct] {
	// Check if we already have this map cached
	if existing := s.GetList(fieldNum); existing != nil {
		if m, ok := existing.(*Maps[K, *Struct]); ok {
			return m
		}
	}

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Parse from segment data
	data := s.seg.data[offset : offset+size]
	keys, values := ParseMapFromSegment[K, *Struct](data, keyType, field.FTStruct, valueMapping)

	m := &Maps[K, *Struct]{
		parent:       s,
		fieldNum:     fieldNum,
		keyType:      keyType,
		valType:      field.FTStruct,
		keys:         keys,
		values:       values,
		dirty:        false,
		valueMapping: valueMapping,
	}
	s.SetList(fieldNum, m)
	return m
}
