package segment

import (
	"bytes"
	"testing"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// benchMapping creates a mapping similar to a real Pod struct for benchmarking.
func benchMapping() *mapping.Map {
	return &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Name", Type: field.FTString},
			{Name: "Namespace", Type: field.FTString},
			{Name: "UID", Type: field.FTString},
			{Name: "Generation", Type: field.FTInt64},
			{Name: "CreationTimestamp", Type: field.FTInt64},
			{Name: "DeletionTimestamp", Type: field.FTInt64},
			{Name: "Labels", Type: field.FTListStrings},
			{Name: "Annotations", Type: field.FTListStrings},
			{Name: "OwnerRefs", Type: field.FTListStructs},
			{Name: "Finalizers", Type: field.FTListStrings},
		},
	}
}

// BenchmarkSegmentMarshal benchmarks the segment-based marshal.
func BenchmarkSegmentMarshal(b *testing.B) {
	m := benchMapping()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(m)

		// Set scalar fields
		SetString(s, 0, "my-pod-name")
		SetString(s, 1, "default")
		SetString(s, 2, "abc123-def456-ghi789")
		SetInt64(s, 3, 1)
		SetInt64(s, 4, 1703721600)
		SetInt64(s, 5, 0)

		// Set list fields
		labels := NewStrings(s, 6)
		labels.Append("app=myapp", "env=prod", "version=v1.0.0", "team=platform")

		annotations := NewStrings(s, 7)
		annotations.Append("kubernetes.io/created-by=controller", "prometheus.io/scrape=true")

		finalizers := NewStrings(s, 9)
		finalizers.Append("kubernetes.io/pv-protection")

		// Marshal
		var buf bytes.Buffer
		s.Marshal(&buf)
	}
}

// BenchmarkSegmentMarshalComplex benchmarks with more complex nested data.
func BenchmarkSegmentMarshalComplex(b *testing.B) {
	m := benchMapping()
	innerMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Name", Type: field.FTString},
			{Name: "Kind", Type: field.FTString},
			{Name: "APIVersion", Type: field.FTString},
			{Name: "UID", Type: field.FTString},
			{Name: "Controller", Type: field.FTBool},
			{Name: "BlockOwnerDeletion", Type: field.FTBool},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(m)

		// Set scalar fields
		SetString(s, 0, "my-pod-name-with-longer-identifier-12345")
		SetString(s, 1, "kube-system")
		SetString(s, 2, "abc123-def456-ghi789-jkl012-mno345")
		SetInt64(s, 3, 42)
		SetInt64(s, 4, 1703721600000000000)
		SetInt64(s, 5, 1703808000000000000)

		// Set list fields with more data
		labels := NewStrings(s, 6)
		labels.Append(
			"app=myapp",
			"env=prod",
			"version=v1.0.0",
			"team=platform",
			"component=frontend",
			"managed-by=helm",
			"chart=myapp-1.2.3",
			"heritage=Helm",
		)

		annotations := NewStrings(s, 7)
		annotations.Append(
			"kubernetes.io/created-by=controller",
			"prometheus.io/scrape=true",
			"prometheus.io/port=8080",
			"prometheus.io/path=/metrics",
			"kubectl.kubernetes.io/last-applied-configuration={\"apiVersion\":\"v1\",\"kind\":\"Pod\"}",
		)

		// Nested structs
		ownerRefs := NewStructs(s, 8, innerMapping)
		for j := 0; j < 3; j++ {
			ref := ownerRefs.NewItem()
			SetString(ref, 0, "my-deployment")
			SetString(ref, 1, "Deployment")
			SetString(ref, 2, "apps/v1")
			SetString(ref, 3, "owner-uid-12345")
			SetBool(ref, 4, true)
			SetBool(ref, 5, true)
			ownerRefs.Append(ref)
		}

		finalizers := NewStrings(s, 9)
		finalizers.Append(
			"kubernetes.io/pv-protection",
			"foregroundDeletion",
		)

		// Marshal
		var buf bytes.Buffer
		s.Marshal(&buf)
	}
}

// BenchmarkSegmentScalarsOnly benchmarks just scalar field operations.
func BenchmarkSegmentScalarsOnly(b *testing.B) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Field0", Type: field.FTInt32},
			{Name: "Field1", Type: field.FTInt32},
			{Name: "Field2", Type: field.FTInt32},
			{Name: "Field3", Type: field.FTInt64},
			{Name: "Field4", Type: field.FTInt64},
			{Name: "Field5", Type: field.FTString},
			{Name: "Field6", Type: field.FTString},
			{Name: "Field7", Type: field.FTBool},
			{Name: "Field8", Type: field.FTBool},
			{Name: "Field9", Type: field.FTFloat64},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(m)

		SetInt32(s, 0, 100)
		SetInt32(s, 1, 200)
		SetInt32(s, 2, 300)
		SetInt64(s, 3, 1000000)
		SetInt64(s, 4, 2000000)
		SetString(s, 5, "hello world")
		SetString(s, 6, "foo bar baz")
		SetBool(s, 7, true)
		SetBool(s, 8, false)
		SetFloat64(s, 9, 3.14159265359)

		var buf bytes.Buffer
		s.Marshal(&buf)
	}
}

// BenchmarkSegmentPooledMarshal benchmarks the pooled segment-based marshal.
func BenchmarkSegmentPooledMarshal(b *testing.B) {
	ctx := context.Background()
	m := benchMapping()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewPooled(ctx, m)

		// Set scalar fields
		SetString(s, 0, "my-pod-name")
		SetString(s, 1, "default")
		SetString(s, 2, "abc123-def456-ghi789")
		SetInt64(s, 3, 1)
		SetInt64(s, 4, 1703721600)
		SetInt64(s, 5, 0)

		// Set list fields
		labels := NewStrings(s, 6)
		labels.Append("app=myapp", "env=prod", "version=v1.0.0", "team=platform")

		annotations := NewStrings(s, 7)
		annotations.Append("kubernetes.io/created-by=controller", "prometheus.io/scrape=true")

		finalizers := NewStrings(s, 9)
		finalizers.Append("kubernetes.io/pv-protection")

		// Marshal
		var buf bytes.Buffer
		s.Marshal(&buf)

		// Release back to pool
		Release(ctx, s)
	}
}

// BenchmarkSegmentPooledScalarsOnly benchmarks pooled scalar field operations.
func BenchmarkSegmentPooledScalarsOnly(b *testing.B) {
	ctx := context.Background()
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Field0", Type: field.FTInt32},
			{Name: "Field1", Type: field.FTInt32},
			{Name: "Field2", Type: field.FTInt32},
			{Name: "Field3", Type: field.FTInt64},
			{Name: "Field4", Type: field.FTInt64},
			{Name: "Field5", Type: field.FTString},
			{Name: "Field6", Type: field.FTString},
			{Name: "Field7", Type: field.FTBool},
			{Name: "Field8", Type: field.FTBool},
			{Name: "Field9", Type: field.FTFloat64},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewPooled(ctx, m)

		SetInt32(s, 0, 100)
		SetInt32(s, 1, 200)
		SetInt32(s, 2, 300)
		SetInt64(s, 3, 1000000)
		SetInt64(s, 4, 2000000)
		SetString(s, 5, "hello world")
		SetString(s, 6, "foo bar baz")
		SetBool(s, 7, true)
		SetBool(s, 8, false)
		SetFloat64(s, 9, 3.14159265359)

		var buf bytes.Buffer
		s.Marshal(&buf)

		Release(ctx, s)
	}
}
