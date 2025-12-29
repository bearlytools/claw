package benchmark

import (
	"fmt"
	"testing"
	
	clawpod "github.com/bearlytools/claw/benchmark/msgs/claw"
	"github.com/bearlytools/claw/clawc/languages/go/structs"
)

func TestDebugSizeTracking(t *testing.T) {
	om := clawpod.NewObjectMeta()
	s := om.XXXGetStruct()
	t.Logf("After NewObjectMeta: structTotal=%d, header=%d", structs.XXXGetStructTotal(s), structs.XXXGetHeaderSize(s))
	
	om = om.SetName("test-pod")
	t.Logf("After SetName: structTotal=%d, header=%d", structs.XXXGetStructTotal(s), structs.XXXGetHeaderSize(s))
	
	// Create labels
	labels := make([]clawpod.KeyValue, listSize)
	for i := 0; i < listSize; i++ {
		kv := clawpod.NewKeyValue()
		kv = kv.SetKey(fmt.Sprintf("label-key-%d", i))
		kv = kv.SetValue(fmt.Sprintf("label-value-%d", i))
		labels[i] = kv
	}
	t.Logf("Before AppendLabels: structTotal=%d, header=%d", structs.XXXGetStructTotal(s), structs.XXXGetHeaderSize(s))
	om.AppendLabels(labels...)
	t.Logf("After AppendLabels: structTotal=%d, header=%d", structs.XXXGetStructTotal(s), structs.XXXGetHeaderSize(s))
	
	// Create annotations
	annotations := make([]clawpod.KeyValue, listSize)
	for i := 0; i < listSize; i++ {
		kv := clawpod.NewKeyValue()
		kv = kv.SetKey(fmt.Sprintf("annotation-key-%d", i))
		kv = kv.SetValue(fmt.Sprintf("annotation-value-%d", i))
		annotations[i] = kv
	}
	t.Logf("Before AppendAnnotations: structTotal=%d, header=%d", structs.XXXGetStructTotal(s), structs.XXXGetHeaderSize(s))
	om.AppendAnnotations(annotations...)
	t.Logf("After AppendAnnotations: structTotal=%d, header=%d", structs.XXXGetStructTotal(s), structs.XXXGetHeaderSize(s))
	
	data, err := om.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Marshal size: %d", len(data))
}
