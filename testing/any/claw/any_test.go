package any

import (
	"context"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/kylelemons/godebug/pretty"
)

// walkable is an interface for types that have a Walk method.
type walkable interface {
	Walk(context.Context, clawiter.YieldToken, ...clawiter.WalkOption)
}

// toWalker converts a walkable into a clawiter.Walker for use with Ingest.
func toWalker(ctx context.Context, w walkable) clawiter.Walker {
	return func(yield clawiter.YieldToken) {
		w.Walk(ctx, yield)
	}
}

func TestAnySingleRoundtrip(t *testing.T) {
	ctx := t.Context()

	// Create an Inner struct to store in the Any field
	inner := NewInner(ctx).SetID(12345).SetValue("test value")

	// Create a Container with the Any field
	container := NewContainer(ctx).SetName("test container")
	if err := container.SetData(inner); err != nil {
		t.Fatalf("[TestAnySingleRoundtrip]: SetData() error: %v", err)
	}

	// Verify IsSet
	if !container.IsSetData() {
		t.Errorf("[TestAnySingleRoundtrip]: IsSetData() = false, want true")
	}

	// Marshal
	data, err := container.Marshal()
	if err != nil {
		t.Fatalf("[TestAnySingleRoundtrip]: Marshal() error: %v", err)
	}

	// Unmarshal into new struct
	container2 := NewContainer(ctx)
	if err := container2.Unmarshal(data); err != nil {
		t.Fatalf("[TestAnySingleRoundtrip]: Unmarshal() error: %v", err)
	}

	// Verify Name
	if container2.Name() != "test container" {
		t.Errorf("[TestAnySingleRoundtrip]: Name() = %q, want %q", container2.Name(), "test container")
	}

	// Decode the Any field into a new Inner
	inner2 := NewInner(ctx)
	if err := container2.Data(inner2); err != nil {
		t.Fatalf("[TestAnySingleRoundtrip]: Data() error: %v", err)
	}

	// Verify Inner contents
	if inner2.ID() != 12345 {
		t.Errorf("[TestAnySingleRoundtrip]: inner.ID() = %d, want 12345", inner2.ID())
	}
	if inner2.Value() != "test value" {
		t.Errorf("[TestAnySingleRoundtrip]: inner.Value() = %q, want %q", inner2.Value(), "test value")
	}
}

func TestAnyWithDifferentTypes(t *testing.T) {
	ctx := t.Context()

	// Store an Outer struct (which has a nested Inner)
	inner := NewInner(ctx).SetID(999).SetValue("nested")
	outer := NewOuter(ctx).SetLabel("outer label").SetCount(42).SetNested(inner)

	container := NewContainer(ctx).SetName("complex container")
	if err := container.SetData(outer); err != nil {
		t.Fatalf("[TestAnyWithDifferentTypes]: SetData(outer) error: %v", err)
	}

	// Marshal and unmarshal
	data, err := container.Marshal()
	if err != nil {
		t.Fatalf("[TestAnyWithDifferentTypes]: Marshal() error: %v", err)
	}

	container2 := NewContainer(ctx)
	if err := container2.Unmarshal(data); err != nil {
		t.Fatalf("[TestAnyWithDifferentTypes]: Unmarshal() error: %v", err)
	}

	// Decode as Outer
	outer2 := NewOuter(ctx)
	if err := container2.Data(outer2); err != nil {
		t.Fatalf("[TestAnyWithDifferentTypes]: Data() error: %v", err)
	}

	if outer2.Label() != "outer label" {
		t.Errorf("[TestAnyWithDifferentTypes]: outer.Label() = %q, want %q", outer2.Label(), "outer label")
	}
	if outer2.Count() != 42 {
		t.Errorf("[TestAnyWithDifferentTypes]: outer.Count() = %d, want 42", outer2.Count())
	}

	nested := outer2.Nested()
	if nested.ID() != 999 {
		t.Errorf("[TestAnyWithDifferentTypes]: nested.ID() = %d, want 999", nested.ID())
	}
}

func TestAnyTypeMismatch(t *testing.T) {
	ctx := t.Context()

	// Store an Inner
	inner := NewInner(ctx).SetID(100).SetValue("inner value")
	container := NewContainer(ctx)
	if err := container.SetData(inner); err != nil {
		t.Fatalf("[TestAnyTypeMismatch]: SetData() error: %v", err)
	}

	// Marshal and unmarshal
	data, err := container.Marshal()
	if err != nil {
		t.Fatalf("[TestAnyTypeMismatch]: Marshal() error: %v", err)
	}

	container2 := NewContainer(ctx)
	if err := container2.Unmarshal(data); err != nil {
		t.Fatalf("[TestAnyTypeMismatch]: Unmarshal() error: %v", err)
	}

	// Try to decode as Outer - should fail due to type hash mismatch
	outer := NewOuter(ctx)
	if err := container2.Data(outer); err == nil {
		t.Errorf("[TestAnyTypeMismatch]: Data(outer) should have failed but succeeded")
	}
}

func TestAnyRaw(t *testing.T) {
	ctx := t.Context()

	inner := NewInner(ctx).SetID(555).SetValue("raw test")
	container := NewContainer(ctx)
	if err := container.SetData(inner); err != nil {
		t.Fatalf("[TestAnyRaw]: SetData() error: %v", err)
	}

	// Get raw data
	rawData, typeHash, ok := container.DataRaw()
	if !ok {
		t.Fatalf("[TestAnyRaw]: DataRaw() returned not ok")
	}
	if len(rawData) == 0 {
		t.Errorf("[TestAnyRaw]: DataRaw() returned empty data")
	}
	if typeHash == [16]byte{} {
		t.Errorf("[TestAnyRaw]: DataRaw() returned empty typeHash")
	}

	// Set raw data on a new container
	container2 := NewContainer(ctx)
	if err := container2.SetDataRaw(rawData, typeHash[:]); err != nil {
		t.Fatalf("[TestAnyRaw]: SetDataRaw() error: %v", err)
	}

	// Decode and verify
	inner2 := NewInner(ctx)
	if err := container2.Data(inner2); err != nil {
		t.Fatalf("[TestAnyRaw]: Data() error: %v", err)
	}
	if inner2.ID() != 555 {
		t.Errorf("[TestAnyRaw]: inner.ID() = %d, want 555", inner2.ID())
	}
}

func TestListAnyRoundtrip(t *testing.T) {
	ctx := t.Context()

	// Create multiple items of different types
	inner1 := NewInner(ctx).SetID(1).SetValue("first")
	inner2 := NewInner(ctx).SetID(2).SetValue("second")
	outer := NewOuter(ctx).SetLabel("outer").SetCount(10)

	// Set the list
	listContainer := NewListContainer(ctx).SetName("list test")
	if err := listContainer.SetItems([]any{inner1, inner2, outer}); err != nil {
		t.Fatalf("[TestListAnyRoundtrip]: SetItems() error: %v", err)
	}

	// Verify length
	if listContainer.ItemsLen() != 3 {
		t.Errorf("[TestListAnyRoundtrip]: ItemsLen() = %d, want 3", listContainer.ItemsLen())
	}

	// Marshal and unmarshal
	data, err := listContainer.Marshal()
	if err != nil {
		t.Fatalf("[TestListAnyRoundtrip]: Marshal() error: %v", err)
	}

	listContainer2 := NewListContainer(ctx)
	if err := listContainer2.Unmarshal(data); err != nil {
		t.Fatalf("[TestListAnyRoundtrip]: Unmarshal() error: %v", err)
	}

	// Verify name
	if listContainer2.Name() != "list test" {
		t.Errorf("[TestListAnyRoundtrip]: Name() = %q, want %q", listContainer2.Name(), "list test")
	}

	// Verify length
	if listContainer2.ItemsLen() != 3 {
		t.Errorf("[TestListAnyRoundtrip]: ItemsLen() = %d, want 3", listContainer2.ItemsLen())
	}

	// Decode first item as Inner
	decoded1 := NewInner(ctx)
	if err := listContainer2.ItemsGet(0, decoded1); err != nil {
		t.Fatalf("[TestListAnyRoundtrip]: ItemsGet(0) error: %v", err)
	}
	if decoded1.ID() != 1 || decoded1.Value() != "first" {
		t.Errorf("[TestListAnyRoundtrip]: item 0 = %d/%q, want 1/first", decoded1.ID(), decoded1.Value())
	}

	// Decode second item
	decoded2 := NewInner(ctx)
	if err := listContainer2.ItemsGet(1, decoded2); err != nil {
		t.Fatalf("[TestListAnyRoundtrip]: ItemsGet(1) error: %v", err)
	}
	if decoded2.ID() != 2 || decoded2.Value() != "second" {
		t.Errorf("[TestListAnyRoundtrip]: item 1 = %d/%q, want 2/second", decoded2.ID(), decoded2.Value())
	}

	// Decode third item as Outer
	decoded3 := NewOuter(ctx)
	if err := listContainer2.ItemsGet(2, decoded3); err != nil {
		t.Fatalf("[TestListAnyRoundtrip]: ItemsGet(2) error: %v", err)
	}
	if decoded3.Label() != "outer" || decoded3.Count() != 10 {
		t.Errorf("[TestListAnyRoundtrip]: item 2 = %q/%d, want outer/10", decoded3.Label(), decoded3.Count())
	}
}

func TestWalkIngestAny(t *testing.T) {
	ctx := t.Context()

	// Create a container with an Any field
	inner := NewInner(ctx).SetID(777).SetValue("walk test")
	container := NewContainer(ctx).SetName("walk container")
	if err := container.SetData(inner); err != nil {
		t.Fatalf("[TestWalkIngestAny]: SetData() error: %v", err)
	}

	// Walk and Ingest round-trip
	container2 := NewContainer(ctx)
	if err := container2.Ingest(ctx, toWalker(ctx, container)); err != nil {
		t.Fatalf("[TestWalkIngestAny]: Ingest() error: %v", err)
	}

	// Verify the container
	if container2.Name() != "walk container" {
		t.Errorf("[TestWalkIngestAny]: Name() = %q, want %q", container2.Name(), "walk container")
	}

	// Decode the Any field
	inner2 := NewInner(ctx)
	if err := container2.Data(inner2); err != nil {
		t.Fatalf("[TestWalkIngestAny]: Data() error: %v", err)
	}

	if inner2.ID() != 777 || inner2.Value() != "walk test" {
		t.Errorf("[TestWalkIngestAny]: inner = %d/%q, want 777/walk test", inner2.ID(), inner2.Value())
	}
}

func TestWalkIngestListAny(t *testing.T) {
	ctx := t.Context()

	// Create a list container
	inner := NewInner(ctx).SetID(888).SetValue("list walk")
	listContainer := NewListContainer(ctx).SetName("list walk container")
	if err := listContainer.SetItems([]any{inner}); err != nil {
		t.Fatalf("[TestWalkIngestListAny]: SetItems() error: %v", err)
	}

	// Walk and Ingest round-trip
	listContainer2 := NewListContainer(ctx)
	if err := listContainer2.Ingest(ctx, toWalker(ctx, listContainer)); err != nil {
		t.Fatalf("[TestWalkIngestListAny]: Ingest() error: %v", err)
	}

	// Verify the list
	if listContainer2.ItemsLen() != 1 {
		t.Fatalf("[TestWalkIngestListAny]: ItemsLen() = %d, want 1", listContainer2.ItemsLen())
	}

	inner2 := NewInner(ctx)
	if err := listContainer2.ItemsGet(0, inner2); err != nil {
		t.Fatalf("[TestWalkIngestListAny]: ItemsGet(0) error: %v", err)
	}

	if inner2.ID() != 888 || inner2.Value() != "list walk" {
		t.Errorf("[TestWalkIngestListAny]: inner = %d/%q, want 888/list walk", inner2.ID(), inner2.Value())
	}
}

func TestAnyNilValue(t *testing.T) {
	ctx := t.Context()

	// Create a container without setting the Any field
	container := NewContainer(ctx).SetName("empty container")

	// Verify IsSet is false
	if container.IsSetData() {
		t.Errorf("[TestAnyNilValue]: IsSetData() = true, want false")
	}

	// Get raw should return not ok
	_, _, ok := container.DataRaw()
	if ok {
		t.Errorf("[TestAnyNilValue]: DataRaw() ok = true, want false")
	}

	// Marshal and unmarshal
	data, err := container.Marshal()
	if err != nil {
		t.Fatalf("[TestAnyNilValue]: Marshal() error: %v", err)
	}

	container2 := NewContainer(ctx)
	if err := container2.Unmarshal(data); err != nil {
		t.Fatalf("[TestAnyNilValue]: Unmarshal() error: %v", err)
	}

	// Verify IsSet is still false
	if container2.IsSetData() {
		t.Errorf("[TestAnyNilValue]: after unmarshal IsSetData() = true, want false")
	}
}

func TestTypeHash(t *testing.T) {
	ctx := t.Context()

	// Verify type hashes are different for different types
	inner := NewInner(ctx)
	outer := NewOuter(ctx)

	innerHash := inner.XXXTypeHash()
	outerHash := outer.XXXTypeHash()

	if innerHash == outerHash {
		t.Errorf("[TestTypeHash]: Inner and Outer have same type hash, want different")
	}

	// Verify same type has same hash
	inner2 := NewInner(ctx)
	if inner.XXXTypeHash() != inner2.XXXTypeHash() {
		t.Errorf("[TestTypeHash]: Two Inner instances have different type hashes")
	}
}

// Ensure import is used
var _ = pretty.Compare
