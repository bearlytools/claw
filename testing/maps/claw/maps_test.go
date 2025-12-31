package maps

import (
	"context"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/field"
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

func TestConfigMapsRoundtrip(t *testing.T) {
	ctx := t.Context()

	c := NewConfig(ctx)
	c.LabelsSet("env", "production")
	c.LabelsSet("team", "platform")
	c.PortsSet("http", 8080)
	c.PortsSet("https", 8443)
	c.EnabledSet("debug", true)
	c.EnabledSet("logging", false)
	c.CountsSet(1, 100)
	c.CountsSet(2, 200)
	c.RatiosSet(3.14, "pi")
	c.RatiosSet(2.71, "e")

	// Marshal
	data, err := c.Marshal()
	if err != nil {
		t.Fatalf("TestConfigMapsRoundtrip: Marshal() error: %v", err)
	}

	// Unmarshal into new struct
	c2 := NewConfig(ctx)
	if err := c2.Unmarshal(data); err != nil {
		t.Fatalf("TestConfigMapsRoundtrip: Unmarshal() error: %v", err)
	}

	// Verify Labels
	if v, ok := c2.LabelsGet("env"); !ok || v != "production" {
		t.Errorf("TestConfigMapsRoundtrip: LabelsGet(env) = %q, %v, want production, true", v, ok)
	}
	if v, ok := c2.LabelsGet("team"); !ok || v != "platform" {
		t.Errorf("TestConfigMapsRoundtrip: LabelsGet(team) = %q, %v, want platform, true", v, ok)
	}

	// Verify Ports
	if v, ok := c2.PortsGet("http"); !ok || v != 8080 {
		t.Errorf("TestConfigMapsRoundtrip: PortsGet(http) = %d, %v, want 8080, true", v, ok)
	}

	// Verify Enabled
	if v, ok := c2.EnabledGet("debug"); !ok || v != true {
		t.Errorf("TestConfigMapsRoundtrip: EnabledGet(debug) = %v, %v, want true, true", v, ok)
	}
	if v, ok := c2.EnabledGet("logging"); !ok || v != false {
		t.Errorf("TestConfigMapsRoundtrip: EnabledGet(logging) = %v, %v, want false, true", v, ok)
	}

	// Verify Counts
	if v, ok := c2.CountsGet(1); !ok || v != 100 {
		t.Errorf("TestConfigMapsRoundtrip: CountsGet(1) = %d, %v, want 100, true", v, ok)
	}

	// Verify Ratios
	if v, ok := c2.RatiosGet(3.14); !ok || v != "pi" {
		t.Errorf("TestConfigMapsRoundtrip: RatiosGet(3.14) = %q, %v, want pi, true", v, ok)
	}
}

func TestComplexMapsRoundtrip(t *testing.T) {
	ctx := t.Context()

	c := NewComplexMaps(ctx)

	// Create settings
	s1 := NewSetting(ctx).SetName("timeout").SetValue("30s").SetPriority(1)
	s2 := NewSetting(ctx).SetName("retries").SetValue("3").SetPriority(2)

	c.SettingsSet("config1", s1)
	c.SettingsSet("config2", s2)

	// Marshal
	data, err := c.Marshal()
	if err != nil {
		t.Fatalf("TestComplexMapsRoundtrip: Marshal() error: %v", err)
	}

	// Unmarshal into new struct
	c2 := NewComplexMaps(ctx)
	if err := c2.Unmarshal(data); err != nil {
		t.Fatalf("TestComplexMapsRoundtrip: Unmarshal() error: %v", err)
	}

	// Verify Settings
	if v, ok := c2.SettingsGet("config1"); !ok {
		t.Errorf("TestComplexMapsRoundtrip: SettingsGet(config1) not found")
	} else {
		if v.Name() != "timeout" {
			t.Errorf("TestComplexMapsRoundtrip: config1.Name() = %q, want timeout", v.Name())
		}
		if v.Value() != "30s" {
			t.Errorf("TestComplexMapsRoundtrip: config1.Value() = %q, want 30s", v.Value())
		}
		if v.Priority() != 1 {
			t.Errorf("TestComplexMapsRoundtrip: config1.Priority() = %d, want 1", v.Priority())
		}
	}

	if v, ok := c2.SettingsGet("config2"); !ok {
		t.Errorf("TestComplexMapsRoundtrip: SettingsGet(config2) not found")
	} else {
		if v.Name() != "retries" {
			t.Errorf("TestComplexMapsRoundtrip: config2.Name() = %q, want retries", v.Name())
		}
	}
}

func TestConfigToRawFromRaw(t *testing.T) {
	ctx := t.Context()

	// Create original
	c := NewConfig(ctx)
	c.LabelsSet("key1", "value1")
	c.PortsSet("port1", 1234)

	// Convert to raw
	raw := c.ToRaw(ctx)

	// Verify raw values
	wantLabels := map[string]string{"key1": "value1"}
	if diff := pretty.Compare(wantLabels, raw.Labels); diff != "" {
		t.Errorf("TestConfigToRawFromRaw: Labels diff:\n%s", diff)
	}

	wantPorts := map[string]int32{"port1": 1234}
	if diff := pretty.Compare(wantPorts, raw.Ports); diff != "" {
		t.Errorf("TestConfigToRawFromRaw: Ports diff:\n%s", diff)
	}

	// Create from raw
	c2 := NewConfigFromRaw(ctx, raw)

	// Verify roundtrip
	if v, ok := c2.LabelsGet("key1"); !ok || v != "value1" {
		t.Errorf("TestConfigToRawFromRaw: After FromRaw, LabelsGet(key1) = %q, %v, want value1, true", v, ok)
	}
	if v, ok := c2.PortsGet("port1"); !ok || v != 1234 {
		t.Errorf("TestConfigToRawFromRaw: After FromRaw, PortsGet(port1) = %d, %v, want 1234, true", v, ok)
	}
}

func TestMapOperations(t *testing.T) {
	ctx := t.Context()

	c := NewConfig(ctx)

	// Test Set and Get
	c.LabelsSet("key1", "value1")
	if v, ok := c.LabelsGet("key1"); !ok || v != "value1" {
		t.Errorf("TestMapOperations: Get after Set failed")
	}

	// Test Has
	if !c.LabelsHas("key1") {
		t.Errorf("TestMapOperations: Has(key1) = false, want true")
	}
	if c.LabelsHas("nonexistent") {
		t.Errorf("TestMapOperations: Has(nonexistent) = true, want false")
	}

	// Test Len
	c.LabelsSet("key2", "value2")
	if c.LabelsLen() != 2 {
		t.Errorf("TestMapOperations: Len() = %d, want 2", c.LabelsLen())
	}

	// Test Delete
	c.LabelsDelete("key1")
	if c.LabelsHas("key1") {
		t.Errorf("TestMapOperations: Has(key1) after delete = true, want false")
	}
	if c.LabelsLen() != 1 {
		t.Errorf("TestMapOperations: Len() after delete = %d, want 1", c.LabelsLen())
	}

	// Test Update existing key
	c.LabelsSet("key2", "updated")
	if v, ok := c.LabelsGet("key2"); !ok || v != "updated" {
		t.Errorf("TestMapOperations: Get(key2) after update = %q, %v, want updated, true", v, ok)
	}
	if c.LabelsLen() != 1 {
		t.Errorf("TestMapOperations: Len() after update = %d, want 1", c.LabelsLen())
	}
}

func TestMapIteration(t *testing.T) {
	ctx := t.Context()

	c := NewConfig(ctx)
	c.LabelsSet("alpha", "a")
	c.LabelsSet("beta", "b")
	c.LabelsSet("gamma", "g")

	// Iterate using Map().All()
	m := c.LabelsMap()
	count := 0
	for k, v := range m.All() {
		count++
		switch k {
		case "alpha":
			if v != "a" {
				t.Errorf("TestMapIteration: alpha = %q, want a", v)
			}
		case "beta":
			if v != "b" {
				t.Errorf("TestMapIteration: beta = %q, want b", v)
			}
		case "gamma":
			if v != "g" {
				t.Errorf("TestMapIteration: gamma = %q, want g", v)
			}
		default:
			t.Errorf("TestMapIteration: unexpected key %q", k)
		}
	}
	if count != 3 {
		t.Errorf("TestMapIteration: iterated %d items, want 3", count)
	}

	// Keys should be sorted
	keys := m.Keys()
	wantKeys := []string{"alpha", "beta", "gamma"}
	if diff := pretty.Compare(wantKeys, keys); diff != "" {
		t.Errorf("TestMapIteration: Keys() diff:\n%s", diff)
	}
}

func TestWalkIngestConfig(t *testing.T) {
	ctx := t.Context()

	// Create original struct with map data
	c := NewConfig(ctx)
	c.LabelsSet("env", "production")
	c.LabelsSet("team", "platform")
	c.PortsSet("http", 8080)
	c.PortsSet("https", 8443)
	c.EnabledSet("debug", true)
	c.EnabledSet("logging", false)
	c.CountsSet(1, 100)
	c.CountsSet(2, 200)
	c.RatiosSet(3.14, "pi")

	// Create new struct and ingest from Walk
	c2 := NewConfig(ctx)
	if err := c2.Ingest(ctx, toWalker(ctx, c)); err != nil {
		t.Fatalf("TestWalkIngestConfig: Ingest() error: %v", err)
	}

	// Verify Labels
	if v, ok := c2.LabelsGet("env"); !ok || v != "production" {
		t.Errorf("TestWalkIngestConfig: LabelsGet(env) = %q, %v, want production, true", v, ok)
	}
	if v, ok := c2.LabelsGet("team"); !ok || v != "platform" {
		t.Errorf("TestWalkIngestConfig: LabelsGet(team) = %q, %v, want platform, true", v, ok)
	}

	// Verify Ports
	if v, ok := c2.PortsGet("http"); !ok || v != 8080 {
		t.Errorf("TestWalkIngestConfig: PortsGet(http) = %d, %v, want 8080, true", v, ok)
	}
	if v, ok := c2.PortsGet("https"); !ok || v != 8443 {
		t.Errorf("TestWalkIngestConfig: PortsGet(https) = %d, %v, want 8443, true", v, ok)
	}

	// Verify Enabled
	if v, ok := c2.EnabledGet("debug"); !ok || v != true {
		t.Errorf("TestWalkIngestConfig: EnabledGet(debug) = %v, %v, want true, true", v, ok)
	}
	if v, ok := c2.EnabledGet("logging"); !ok || v != false {
		t.Errorf("TestWalkIngestConfig: EnabledGet(logging) = %v, %v, want false, true", v, ok)
	}

	// Verify Counts
	if v, ok := c2.CountsGet(1); !ok || v != 100 {
		t.Errorf("TestWalkIngestConfig: CountsGet(1) = %d, %v, want 100, true", v, ok)
	}

	// Verify Ratios
	if v, ok := c2.RatiosGet(3.14); !ok || v != "pi" {
		t.Errorf("TestWalkIngestConfig: RatiosGet(3.14) = %q, %v, want pi, true", v, ok)
	}
}

func TestWalkIngestComplexMaps(t *testing.T) {
	ctx := t.Context()

	// Create original struct with struct-valued map
	c := NewComplexMaps(ctx)

	s1 := NewSetting(ctx).SetName("timeout").SetValue("30s").SetPriority(1)
	s2 := NewSetting(ctx).SetName("retries").SetValue("3").SetPriority(2)

	c.SettingsSet("config1", s1)
	c.SettingsSet("config2", s2)

	// Create new struct and ingest from Walk
	c2 := NewComplexMaps(ctx)
	if err := c2.Ingest(ctx, toWalker(ctx, c)); err != nil {
		t.Fatalf("TestWalkIngestComplexMaps: Ingest() error: %v", err)
	}

	// Verify Settings
	if v, ok := c2.SettingsGet("config1"); !ok {
		t.Errorf("TestWalkIngestComplexMaps: SettingsGet(config1) not found")
	} else {
		if v.Name() != "timeout" {
			t.Errorf("TestWalkIngestComplexMaps: config1.Name() = %q, want timeout", v.Name())
		}
		if v.Value() != "30s" {
			t.Errorf("TestWalkIngestComplexMaps: config1.Value() = %q, want 30s", v.Value())
		}
		if v.Priority() != 1 {
			t.Errorf("TestWalkIngestComplexMaps: config1.Priority() = %d, want 1", v.Priority())
		}
	}

	if v, ok := c2.SettingsGet("config2"); !ok {
		t.Errorf("TestWalkIngestComplexMaps: SettingsGet(config2) not found")
	} else {
		if v.Name() != "retries" {
			t.Errorf("TestWalkIngestComplexMaps: config2.Name() = %q, want retries", v.Name())
		}
	}
}

func TestWalkIngestEmptyMaps(t *testing.T) {
	ctx := t.Context()

	// Create struct with empty maps
	c := NewConfig(ctx)

	// Create new struct and ingest from Walk
	c2 := NewConfig(ctx)
	if err := c2.Ingest(ctx, toWalker(ctx, c)); err != nil {
		t.Fatalf("TestWalkIngestEmptyMaps: Ingest() error: %v", err)
	}

	// Verify maps are empty
	if c2.LabelsLen() != 0 {
		t.Errorf("TestWalkIngestEmptyMaps: LabelsLen() = %d, want 0", c2.LabelsLen())
	}
	if c2.PortsLen() != 0 {
		t.Errorf("TestWalkIngestEmptyMaps: PortsLen() = %d, want 0", c2.PortsLen())
	}
}

func TestReflectionMapFieldDescr(t *testing.T) {
	// Get the package descriptor
	pkgDescr := PackageDescr()
	configDescr := pkgDescr.Structs().ByName("Config")
	if configDescr == nil {
		t.Fatalf("TestReflectionMapFieldDescr: Config struct not found")
	}

	// Test Labels field (map[string]string)
	labelsField := configDescr.FieldDescrByName("Labels")
	if labelsField == nil {
		t.Fatalf("TestReflectionMapFieldDescr: Labels field not found")
	}
	if !labelsField.IsMap() {
		t.Errorf("TestReflectionMapFieldDescr: Labels.IsMap() = false, want true")
	}
	if labelsField.MapKeyType() != field.FTString {
		t.Errorf("TestReflectionMapFieldDescr: Labels.MapKeyType() = %v, want FTString", labelsField.MapKeyType())
	}
	if labelsField.MapValueType() != field.FTString {
		t.Errorf("TestReflectionMapFieldDescr: Labels.MapValueType() = %v, want FTString", labelsField.MapValueType())
	}

	// Test Ports field (map[string]int32)
	portsField := configDescr.FieldDescrByName("Ports")
	if portsField == nil {
		t.Fatalf("TestReflectionMapFieldDescr: Ports field not found")
	}
	if !portsField.IsMap() {
		t.Errorf("TestReflectionMapFieldDescr: Ports.IsMap() = false, want true")
	}
	if portsField.MapKeyType() != field.FTString {
		t.Errorf("TestReflectionMapFieldDescr: Ports.MapKeyType() = %v, want FTString", portsField.MapKeyType())
	}
	if portsField.MapValueType() != field.FTInt32 {
		t.Errorf("TestReflectionMapFieldDescr: Ports.MapValueType() = %v, want FTInt32", portsField.MapValueType())
	}

	// Test Counts field (map[int32]int64)
	countsField := configDescr.FieldDescrByName("Counts")
	if countsField == nil {
		t.Fatalf("TestReflectionMapFieldDescr: Counts field not found")
	}
	if !countsField.IsMap() {
		t.Errorf("TestReflectionMapFieldDescr: Counts.IsMap() = false, want true")
	}
	if countsField.MapKeyType() != field.FTInt32 {
		t.Errorf("TestReflectionMapFieldDescr: Counts.MapKeyType() = %v, want FTInt32", countsField.MapKeyType())
	}
	if countsField.MapValueType() != field.FTInt64 {
		t.Errorf("TestReflectionMapFieldDescr: Counts.MapValueType() = %v, want FTInt64", countsField.MapValueType())
	}

	// Test ComplexMaps Settings field (map[string]Setting)
	complexDescr := pkgDescr.Structs().ByName("ComplexMaps")
	if complexDescr == nil {
		t.Fatalf("TestReflectionMapFieldDescr: ComplexMaps struct not found")
	}
	settingsField := complexDescr.FieldDescrByName("Settings")
	if settingsField == nil {
		t.Fatalf("TestReflectionMapFieldDescr: Settings field not found")
	}
	if !settingsField.IsMap() {
		t.Errorf("TestReflectionMapFieldDescr: Settings.IsMap() = false, want true")
	}
	if settingsField.MapKeyType() != field.FTString {
		t.Errorf("TestReflectionMapFieldDescr: Settings.MapKeyType() = %v, want FTString", settingsField.MapKeyType())
	}
	if settingsField.MapValueType() != field.FTStruct {
		t.Errorf("TestReflectionMapFieldDescr: Settings.MapValueType() = %v, want FTStruct", settingsField.MapValueType())
	}
}

func TestReflectionMapClawStruct(t *testing.T) {
	ctx := t.Context()

	// Create a Config with map data
	c := NewConfig(ctx)
	c.LabelsSet("env", "production")
	c.LabelsSet("team", "platform")
	c.PortsSet("http", 8080)

	// Marshal and unmarshal to sync the dirty maps to the segment
	// HasField only returns true for fields written to the segment wire format
	data, err := c.Marshal()
	if err != nil {
		t.Fatalf("TestReflectionMapClawStruct: Marshal() error: %v", err)
	}
	c2 := NewConfig(ctx)
	if err := c2.Unmarshal(data); err != nil {
		t.Fatalf("TestReflectionMapClawStruct: Unmarshal() error: %v", err)
	}

	// Get the reflection Struct
	rs := c2.ClawStruct()
	descr := rs.Descriptor()

	// Verify Labels field via reflection
	labelsField := descr.FieldDescrByName("Labels")
	if !rs.Has(labelsField) {
		t.Errorf("TestReflectionMapClawStruct: Has(Labels) = false, want true")
	}

	// Verify Ports field via reflection
	portsField := descr.FieldDescrByName("Ports")
	if !rs.Has(portsField) {
		t.Errorf("TestReflectionMapClawStruct: Has(Ports) = false, want true")
	}
}
