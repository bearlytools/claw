package segment

import (
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/kylelemons/godebug/pretty"
)

func TestMapsStringString(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	// Create a map[string]string
	maps := NewMaps[string, string](s, 1, field.FTString, field.FTString, nil)

	// Test Set and Get
	maps.Set("key1", "value1")
	maps.Set("key2", "value2")
	maps.Set("key3", "value3")

	if got, ok := maps.Get("key1"); !ok || got != "value1" {
		t.Errorf("TestMapsStringString: Get(key1) = %q, %v, want value1, true", got, ok)
	}
	if got, ok := maps.Get("key2"); !ok || got != "value2" {
		t.Errorf("TestMapsStringString: Get(key2) = %q, %v, want value2, true", got, ok)
	}

	// Test Has
	if !maps.Has("key1") {
		t.Errorf("TestMapsStringString: Has(key1) = false, want true")
	}
	if maps.Has("nonexistent") {
		t.Errorf("TestMapsStringString: Has(nonexistent) = true, want false")
	}

	// Test Len
	if maps.Len() != 3 {
		t.Errorf("TestMapsStringString: Len() = %d, want 3", maps.Len())
	}

	// Test Delete
	maps.Delete("key2")
	if maps.Has("key2") {
		t.Errorf("TestMapsStringString: Has(key2) after delete = true, want false")
	}
	if maps.Len() != 2 {
		t.Errorf("TestMapsStringString: Len() after delete = %d, want 2", maps.Len())
	}

	// Test Keys are sorted
	keys := maps.Keys()
	wantKeys := []string{"key1", "key3"}
	if diff := pretty.Compare(wantKeys, keys); diff != "" {
		t.Errorf("TestMapsStringString: Keys() diff:\n%s", diff)
	}

	// Test All iterator
	var iterKeys []string
	var iterVals []string
	for k, v := range maps.All() {
		iterKeys = append(iterKeys, k)
		iterVals = append(iterVals, v)
	}
	if diff := pretty.Compare(wantKeys, iterKeys); diff != "" {
		t.Errorf("TestMapsStringString: All() keys diff:\n%s", diff)
	}
}

func TestMapsInt32String(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	// Create a map[int32]string
	maps := NewMaps[int32, string](s, 1, field.FTInt32, field.FTString, nil)

	// Test Set and Get with various keys
	maps.Set(100, "hundred")
	maps.Set(-50, "negative fifty")
	maps.Set(0, "zero")
	maps.Set(50, "fifty")

	if got, ok := maps.Get(100); !ok || got != "hundred" {
		t.Errorf("TestMapsInt32String: Get(100) = %q, %v, want hundred, true", got, ok)
	}
	if got, ok := maps.Get(-50); !ok || got != "negative fifty" {
		t.Errorf("TestMapsInt32String: Get(-50) = %q, %v, want negative fifty, true", got, ok)
	}

	// Verify keys are sorted
	keys := maps.Keys()
	wantKeys := []int32{-50, 0, 50, 100}
	if diff := pretty.Compare(wantKeys, keys); diff != "" {
		t.Errorf("TestMapsInt32String: Keys() diff:\n%s", diff)
	}
}

func TestMapsSyncToSegment(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	// Create and populate a map
	maps := NewMaps[string, int32](s, 1, field.FTString, field.FTInt32, nil)
	maps.Set("alpha", 1)
	maps.Set("beta", 2)
	maps.Set("gamma", 3)

	// Sync to segment
	if err := maps.SyncToSegment(); err != nil {
		t.Fatalf("TestMapsSyncToSegment: SyncToSegment() error: %v", err)
	}

	// Verify field is set
	offset, size := s.FieldOffset(1)
	if size == 0 {
		t.Errorf("TestMapsSyncToSegment: FieldOffset(1) size = 0, want > 0")
	}

	// Verify header
	data := s.seg.data[offset : offset+size]
	keyType, valType, totalSize := DecodeMapHeader(data)
	if keyType != field.FTString {
		t.Errorf("TestMapsSyncToSegment: keyType = %v, want FTString", keyType)
	}
	if valType != field.FTInt32 {
		t.Errorf("TestMapsSyncToSegment: valType = %v, want FTInt32", valType)
	}
	if totalSize != uint32(size) {
		t.Errorf("TestMapsSyncToSegment: totalSize = %d, want %d", totalSize, size)
	}
}

func TestMapsEmptySync(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	// Create an empty map
	maps := NewMaps[string, string](s, 1, field.FTString, field.FTString, nil)

	// Sync to segment - should remove the field
	if err := maps.SyncToSegment(); err != nil {
		t.Fatalf("TestMapsEmptySync: SyncToSegment() error: %v", err)
	}

	// Verify field is not set
	_, size := s.FieldOffset(1)
	if size != 0 {
		t.Errorf("TestMapsEmptySync: FieldOffset(1) size = %d, want 0", size)
	}
}

func TestMapsClear(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	maps := NewMaps[string, string](s, 1, field.FTString, field.FTString, nil)
	maps.Set("key1", "value1")
	maps.Set("key2", "value2")

	if maps.Len() != 2 {
		t.Errorf("TestMapsClear: Len() before clear = %d, want 2", maps.Len())
	}

	maps.Clear()

	if maps.Len() != 0 {
		t.Errorf("TestMapsClear: Len() after clear = %d, want 0", maps.Len())
	}
}

func TestMapsUpdateExisting(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	maps := NewMaps[string, string](s, 1, field.FTString, field.FTString, nil)
	maps.Set("key1", "value1")
	maps.Set("key1", "updated")

	if maps.Len() != 1 {
		t.Errorf("TestMapsUpdateExisting: Len() = %d, want 1", maps.Len())
	}

	if got, ok := maps.Get("key1"); !ok || got != "updated" {
		t.Errorf("TestMapsUpdateExisting: Get(key1) = %q, %v, want updated, true", got, ok)
	}
}

func TestMapsBoolKey(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	maps := NewMaps[bool, string](s, 1, field.FTBool, field.FTString, nil)
	maps.Set(true, "yes")
	maps.Set(false, "no")

	if got, ok := maps.Get(true); !ok || got != "yes" {
		t.Errorf("TestMapsBoolKey: Get(true) = %q, %v, want yes, true", got, ok)
	}
	if got, ok := maps.Get(false); !ok || got != "no" {
		t.Errorf("TestMapsBoolKey: Get(false) = %q, %v, want no, true", got, ok)
	}

	// Keys should be sorted: false < true
	keys := maps.Keys()
	if len(keys) != 2 || keys[0] != false || keys[1] != true {
		t.Errorf("TestMapsBoolKey: Keys() = %v, want [false, true]", keys)
	}
}

func TestMapsFloat64Key(t *testing.T) {
	ctx := t.Context()
	m := &mapping.Map{
		Fields: make([]*mapping.FieldDescr, 2),
	}
	s := New(ctx, m)

	maps := NewMaps[float64, string](s, 1, field.FTFloat64, field.FTString, nil)
	maps.Set(3.14, "pi")
	maps.Set(-1.0, "negative")
	maps.Set(2.71, "e")

	if got, ok := maps.Get(3.14); !ok || got != "pi" {
		t.Errorf("TestMapsFloat64Key: Get(3.14) = %q, %v, want pi, true", got, ok)
	}

	// Keys should be sorted: -1.0 < 2.71 < 3.14
	keys := maps.Keys()
	wantKeys := []float64{-1.0, 2.71, 3.14}
	if diff := pretty.Compare(wantKeys, keys); diff != "" {
		t.Errorf("TestMapsFloat64Key: Keys() diff:\n%s", diff)
	}
}
