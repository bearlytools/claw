package metadata

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		kv   []string
		want MD
	}{
		{
			name: "Success: single key-value",
			kv:   []string{"key1", "value1"},
			want: MD{"key1": []byte("value1")},
		},
		{
			name: "Success: multiple key-values",
			kv:   []string{"key1", "value1", "key2", "value2"},
			want: MD{"key1": []byte("value1"), "key2": []byte("value2")},
		},
		{
			name: "Success: keys are lowercased",
			kv:   []string{"KEY1", "value1", "Key2", "VALUE2"},
			want: MD{"key1": []byte("value1"), "key2": []byte("VALUE2")},
		},
		{
			name: "Success: empty",
			kv:   []string{},
			want: MD{},
		},
	}

	for _, test := range tests {
		got := New(test.kv...)
		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("[TestNew](%s): mismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestNewPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("[TestNewPanic]: expected panic for odd number of args")
		}
	}()
	New("key1") // Should panic
}

func TestMDGetSet(t *testing.T) {
	md := New("key1", "value1")

	// Test Get
	if got := md.Get("key1"); string(got) != "value1" {
		t.Errorf("[TestMDGetSet]: Get(key1) = %q, want %q", got, "value1")
	}
	if got := md.Get("KEY1"); string(got) != "value1" {
		t.Errorf("[TestMDGetSet]: Get(KEY1) = %q, want %q (case-insensitive)", got, "value1")
	}
	if got := md.Get("nonexistent"); got != nil {
		t.Errorf("[TestMDGetSet]: Get(nonexistent) = %v, want nil", got)
	}

	// Test GetString
	if got := md.GetString("key1"); got != "value1" {
		t.Errorf("[TestMDGetSet]: GetString(key1) = %q, want %q", got, "value1")
	}
	if got := md.GetString("nonexistent"); got != "" {
		t.Errorf("[TestMDGetSet]: GetString(nonexistent) = %q, want empty", got)
	}

	// Test Set
	md.Set("key2", []byte("value2"))
	if got := md.Get("key2"); string(got) != "value2" {
		t.Errorf("[TestMDGetSet]: Get(key2) = %q, want %q", got, "value2")
	}

	// Test SetString
	md.SetString("key3", "value3")
	if got := md.GetString("key3"); got != "value3" {
		t.Errorf("[TestMDGetSet]: GetString(key3) = %q, want %q", got, "value3")
	}
}

func TestMDDelete(t *testing.T) {
	md := New("key1", "value1", "key2", "value2")
	md.Delete("key1")

	if got := md.Get("key1"); got != nil {
		t.Errorf("[TestMDDelete]: Get(key1) after delete = %v, want nil", got)
	}
	if got := md.Get("key2"); string(got) != "value2" {
		t.Errorf("[TestMDDelete]: Get(key2) = %q, want %q", got, "value2")
	}
}

func TestMDClone(t *testing.T) {
	md := New("key1", "value1")
	clone := md.Clone()

	// Modify original
	md.SetString("key2", "value2")

	// Clone should not be affected
	if got := clone.Get("key2"); got != nil {
		t.Errorf("[TestMDClone]: clone affected by original modification")
	}
	if got := clone.GetString("key1"); got != "value1" {
		t.Errorf("[TestMDClone]: clone.GetString(key1) = %q, want %q", got, "value1")
	}

	// Nil clone
	var nilMD MD
	if nilMD.Clone() != nil {
		t.Errorf("[TestMDClone]: nil.Clone() should return nil")
	}
}

func TestMDLen(t *testing.T) {
	md := New("key1", "value1", "key2", "value2")
	if got := md.Len(); got != 2 {
		t.Errorf("[TestMDLen]: Len() = %d, want 2", got)
	}

	empty := New()
	if got := empty.Len(); got != 0 {
		t.Errorf("[TestMDLen]: empty.Len() = %d, want 0", got)
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := t.Context()
	md := New("key1", "value1")

	// NewContext and FromContext
	ctx = NewContext(ctx, md)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatalf("[TestContextHelpers]: FromContext returned false")
	}
	if diff := pretty.Compare(md, got); diff != "" {
		t.Errorf("[TestContextHelpers]: FromContext mismatch (-want +got):\n%s", diff)
	}

	// FromContext with no metadata
	emptyCtx := t.Context()
	_, ok = FromContext(emptyCtx)
	if ok {
		t.Errorf("[TestContextHelpers]: FromContext on empty context should return false")
	}

	// AppendToContext on empty context
	ctx = t.Context()
	ctx = AppendToContext(ctx, "key1", "value1")
	got, ok = FromContext(ctx)
	if !ok {
		t.Fatalf("[TestContextHelpers]: FromContext after AppendToContext returned false")
	}
	if got.GetString("key1") != "value1" {
		t.Errorf("[TestContextHelpers]: AppendToContext value = %q, want %q", got.GetString("key1"), "value1")
	}

	// AppendToContext on existing metadata
	ctx = AppendToContext(ctx, "key2", "value2")
	got, _ = FromContext(ctx)
	if got.GetString("key1") != "value1" || got.GetString("key2") != "value2" {
		t.Errorf("[TestContextHelpers]: AppendToContext should preserve existing and add new")
	}
}

func TestPairs(t *testing.T) {
	md := Pairs("key1", "string-value", "key2", []byte("bytes-value"))
	if md.GetString("key1") != "string-value" {
		t.Errorf("[TestPairs]: key1 = %q, want %q", md.GetString("key1"), "string-value")
	}
	if string(md.Get("key2")) != "bytes-value" {
		t.Errorf("[TestPairs]: key2 = %q, want %q", md.Get("key2"), "bytes-value")
	}
}
