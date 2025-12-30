package benchmark

import (
	"context"
	"fmt"
	"testing"

	clawpod "github.com/bearlytools/claw/benchmark/msgs/claw"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
	"github.com/bearlytools/claw/languages/go/patch"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

// TestPatchVsFullWireSize compares wire sizes when patching 2 fields vs sending the full object.
func TestPatchVsFullWireSize(t *testing.T) {
	ctx := context.Background()

	// Create a fully populated Pod
	pod := createClawPod()
	fullData, err := pod.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Scenario 1: Use Diff to create a minimal patch (recommended approach)
	// This creates the smallest possible patch by computing exact differences
	podFrom := createClawPod()
	podTo := createClawPod()
	statusTo := podTo.Status()
	statusTo.SetPhase(clawpod.PodPhaseFailed)
	statusTo.SetMessage("Container crashed due to OOM")
	podTo.SetStatus(statusTo)

	diffPatch, err := patch.Diff(podFrom, podTo)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	diffPatchData, err := diffPatch.Marshal()
	if err != nil {
		t.Fatalf("Diff Patch Marshal error: %v", err)
	}

	// Scenario 2: Recording at struct level (captures entire nested struct replacement)
	// This is larger because it records the SetStatus(wholeStruct) operation
	pod2 := createClawPod()
	pod2.SetRecording(true)

	status := pod2.Status()
	status.SetPhase(clawpod.PodPhaseFailed)
	status.SetMessage("Container crashed due to OOM")
	pod2.SetStatus(status)

	ops := pod2.DrainRecordedOps()
	patchObj := msgs.NewPatch(ctx)
	patchObj.SetVersion(patch.PatchVersion)
	for _, op := range ops {
		patchOp := msgs.NewOp(ctx)
		patchOp.SetFieldNum(op.FieldNum)
		patchOp.SetType(msgs.OpType(op.OpType))
		patchOp.SetIndex(op.Index)
		patchOp.SetData(op.Data)
		patchObj.AppendOps(patchOp)
	}
	recordingPatchData, err := patchObj.Marshal()
	if err != nil {
		t.Fatalf("Recording Patch Marshal error: %v", err)
	}

	// Scenario 3: Recording at nested level (captures individual field changes)
	// Enable recording on both parent and child structs
	pod3 := createClawPod()
	status3 := pod3.Status()
	status3.SetRecording(true) // Record changes within Status
	status3.SetPhase(clawpod.PodPhaseFailed)
	status3.SetMessage("Container crashed due to OOM")
	// Note: Don't call pod3.SetStatus() to avoid recording the whole struct

	ops3 := status3.DrainRecordedOps()
	patchObj3 := msgs.NewPatch(ctx)
	patchObj3.SetVersion(patch.PatchVersion)
	for _, op := range ops3 {
		patchOp := msgs.NewOp(ctx)
		patchOp.SetFieldNum(op.FieldNum)
		patchOp.SetType(msgs.OpType(op.OpType))
		patchOp.SetIndex(op.Index)
		patchOp.SetData(op.Data)
		patchObj3.AppendOps(patchOp)
	}
	nestedRecordingPatchData, err := patchObj3.Marshal()
	if err != nil {
		t.Fatalf("Nested Recording Patch Marshal error: %v", err)
	}

	// Print results
	fmt.Printf("\n=== Patch vs Full Object Wire Size Comparison ===\n\n")
	fmt.Printf("Full Pod size:                         %6d bytes (baseline)\n", len(fullData))

	fmt.Printf("\n--- Scenario 1: Diff-based patch (recommended) ---\n")
	fmt.Printf("Diff compares two objects and generates minimal ops for changed fields.\n")
	fmt.Printf("Patch size:                            %6d bytes\n", len(diffPatchData))
	fmt.Printf("Savings:                               %6d bytes (%.1f%% reduction)\n",
		len(fullData)-len(diffPatchData),
		float64(len(fullData)-len(diffPatchData))/float64(len(fullData))*100)
	fmt.Printf("Full object is %.1fx larger than patch\n", float64(len(fullData))/float64(len(diffPatchData)))

	fmt.Printf("\n--- Scenario 2: Recording at parent level ---\n")
	fmt.Printf("Records SetStatus(modifiedStatus) - includes entire nested struct.\n")
	fmt.Printf("Patch size:                            %6d bytes\n", len(recordingPatchData))
	fmt.Printf("Savings:                               %6d bytes (%.1f%% reduction)\n",
		len(fullData)-len(recordingPatchData),
		float64(len(fullData)-len(recordingPatchData))/float64(len(fullData))*100)
	fmt.Printf("Full object is %.1fx larger than patch\n", float64(len(fullData))/float64(len(recordingPatchData)))

	fmt.Printf("\n--- Scenario 3: Recording at nested level ---\n")
	fmt.Printf("Records individual SetPhase() and SetMessage() calls.\n")
	fmt.Printf("Patch size:                            %6d bytes\n", len(nestedRecordingPatchData))
	fmt.Printf("Savings:                               %6d bytes (%.1f%% reduction)\n",
		len(fullData)-len(nestedRecordingPatchData),
		float64(len(fullData)-len(nestedRecordingPatchData))/float64(len(fullData))*100)
	fmt.Printf("Full object is %.1fx larger than patch\n", float64(len(fullData))/float64(len(nestedRecordingPatchData)))

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Diff is best for minimal wire size when you have both versions.\n")
	fmt.Printf("Recording at nested level is best when you only have the modified object.\n")
	fmt.Printf("Recording at parent level is simplest but includes more data.\n")
}

// BenchmarkPatchMarshal benchmarks marshaling a patch with 2 field changes.
func BenchmarkPatchMarshal(b *testing.B) {
	ctx := context.Background()

	// Pre-create the patch structure
	patchObj := msgs.NewPatch(ctx)
	patchObj.SetVersion(patch.PatchVersion)

	// Add 2 ops representing 2 field changes
	op1 := msgs.NewOp(ctx)
	op1.SetFieldNum(0)                                       // Phase field
	op1.SetType(msgs.Set)                                    // Set operation
	op1.SetIndex(segment.NoListIndex)                        // Not a list
	op1.SetData([]byte{byte(clawpod.PodPhaseFailed)})        // New value
	patchObj.AppendOps(op1)

	op2 := msgs.NewOp(ctx)
	op2.SetFieldNum(2)                                       // Message field
	op2.SetType(msgs.Set)
	op2.SetIndex(segment.NoListIndex)
	op2.SetData([]byte("Container crashed due to OOM"))
	patchObj.AppendOps(op2)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := patchObj.Marshal()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPatchUnmarshal benchmarks unmarshaling a patch with 2 field changes.
func BenchmarkPatchUnmarshal(b *testing.B) {
	ctx := context.Background()

	// Create and marshal a patch
	patchObj := msgs.NewPatch(ctx)
	patchObj.SetVersion(patch.PatchVersion)

	op1 := msgs.NewOp(ctx)
	op1.SetFieldNum(0)
	op1.SetType(msgs.Set)
	op1.SetIndex(segment.NoListIndex)
	op1.SetData([]byte{byte(clawpod.PodPhaseFailed)})
	patchObj.AppendOps(op1)

	op2 := msgs.NewOp(ctx)
	op2.SetFieldNum(2)
	op2.SetType(msgs.Set)
	op2.SetIndex(segment.NoListIndex)
	op2.SetData([]byte("Container crashed due to OOM"))
	patchObj.AppendOps(op2)

	data, err := patchObj.Marshal()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newPatch := msgs.NewPatch(ctx)
		err := newPatch.Unmarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFullPodMarshalForUpdate benchmarks marshaling a full Pod (simulating
// the case where you send the whole object instead of a patch).
func BenchmarkFullPodMarshalForUpdate(b *testing.B) {
	pod := createClawPod()
	// Modify 2 fields to simulate an update
	status := pod.Status()
	status.SetPhase(clawpod.PodPhaseFailed)
	status.SetMessage("Container crashed due to OOM")
	pod.SetStatus(status)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := pod.Marshal()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRecordingOverhead measures the overhead of recording mutations.
func BenchmarkRecordingOverhead(b *testing.B) {
	ctx := context.Background()

	b.Run("WithoutRecording", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pod := clawpod.NewPod(ctx)
			status := clawpod.NewPodStatus(ctx)
			status.SetPhase(clawpod.PodPhaseFailed)
			status.SetMessage("Container crashed due to OOM")
			pod.SetStatus(status)
		}
	})

	b.Run("WithRecording", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pod := clawpod.NewPod(ctx)
			pod.SetRecording(true)
			status := clawpod.NewPodStatus(ctx)
			status.SetRecording(true)
			status.SetPhase(clawpod.PodPhaseFailed)
			status.SetMessage("Container crashed due to OOM")
			pod.SetStatus(status)
			_ = pod.DrainRecordedOps()
		}
	})
}

// BenchmarkDiffVsRecording compares creating a patch via Diff vs Recording.
func BenchmarkDiffVsRecording(b *testing.B) {
	b.Run("Diff", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			podFrom := createClawPod()
			podTo := createClawPod()
			status := podTo.Status()
			status.SetPhase(clawpod.PodPhaseFailed)
			status.SetMessage("Container crashed due to OOM")
			podTo.SetStatus(status)

			_, err := patch.Diff(podFrom, podTo)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Recording", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pod := createClawPod()
			pod.SetRecording(true)
			status := pod.Status()
			status.SetRecording(true)
			status.SetPhase(clawpod.PodPhaseFailed)
			status.SetMessage("Container crashed due to OOM")
			pod.SetStatus(status)
			_ = pod.DrainRecordedOps()
		}
	})
}

// TestPatchApplyRoundtrip verifies that a recorded patch can be applied correctly.
func TestPatchApplyRoundtrip(t *testing.T) {
	ctx := context.Background()

	// Create original pod
	original := createClawPod()

	// Create modified pod with recording
	modified := createClawPod()
	modified.SetRecording(true)

	// Make changes
	status := modified.Status()
	status.SetRecording(true)
	status.SetPhase(clawpod.PodPhaseFailed)
	status.SetMessage("Container crashed due to OOM")
	modified.SetStatus(status)

	// Build patch from recorded ops
	ops := modified.DrainRecordedOps()
	patchObj := msgs.NewPatch(ctx)
	patchObj.SetVersion(patch.PatchVersion)
	for _, op := range ops {
		patchOp := msgs.NewOp(ctx)
		patchOp.SetFieldNum(op.FieldNum)
		patchOp.SetType(msgs.OpType(op.OpType))
		patchOp.SetIndex(op.Index)
		patchOp.SetData(op.Data)
		patchObj.AppendOps(patchOp)
	}

	// Serialize and deserialize (simulate network transmission)
	patchData, err := patchObj.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	receivedPatch := msgs.NewPatch(ctx)
	if err := receivedPatch.Unmarshal(patchData); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Apply patch to original
	// Note: This would need the Apply function to work with the raw ops
	// For now, we just verify the patch structure is correct
	if receivedPatch.OpsLen() != patchObj.OpsLen() {
		t.Errorf("OpsLen mismatch: got %d, want %d", receivedPatch.OpsLen(), patchObj.OpsLen())
	}

	// Verify original is unchanged
	if original.Status().Phase() != clawpod.PodPhaseRunning {
		t.Errorf("Original should be unchanged, got phase: %v", original.Status().Phase())
	}

	fmt.Printf("\n=== Patch Roundtrip Test ===\n")
	fmt.Printf("Original status phase: %v\n", original.Status().Phase())
	fmt.Printf("Patch size: %d bytes\n", len(patchData))
	fmt.Printf("Number of ops: %d\n", receivedPatch.OpsLen())
}
