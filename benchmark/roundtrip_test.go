package benchmark

import (
	"context"
	"fmt"
	"testing"

	clawpod "github.com/bearlytools/claw/benchmark/msgs/claw"
)

// TestClawRoundTripSimple is a simple test to diagnose unmarshal issues.
func TestClawRoundTripSimple(t *testing.T) {
	// Create a simple struct with nested data
	tm := clawpod.NewTypeMeta(context.Background())
	tm = tm.SetKind("Pod")
	tm = tm.SetApiVersion("v1")

	pod := clawpod.NewPod(context.Background())
	pod = pod.SetTypeMeta(tm)

	data, err := pod.Marshal()
	if err != nil {
		t.Fatalf("TestClawRoundTripSimple: Marshal failed: %v", err)
	}

	t.Logf("Marshaled size: %d bytes", len(data))

	restored := clawpod.NewPod(context.Background())
	if err := restored.Unmarshal(data); err != nil {
		t.Fatalf("TestClawRoundTripSimple: Unmarshal failed: %v", err)
	}

	// Check TypeMeta
	gotTM := restored.TypeMeta()
	if gotTM.Kind() != "Pod" {
		t.Errorf("TypeMeta.Kind: got %q, want %q", gotTM.Kind(), "Pod")
	}
	if gotTM.ApiVersion() != "v1" {
		t.Errorf("TypeMeta.ApiVersion: got %q, want %q", gotTM.ApiVersion(), "v1")
	}
}

// TestClawRoundTrip tests that marshaling and unmarshaling a Claw Pod
// preserves all values correctly.
func TestClawRoundTrip(t *testing.T) {
	original := createClawPod()

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("TestClawRoundTrip: Marshal failed: %v", err)
	}

	t.Logf("Marshaled size: %d bytes", len(data))

	restored := clawpod.NewPod(context.Background())
	if err := restored.Unmarshal(data); err != nil {
		t.Fatalf("TestClawRoundTrip: Unmarshal failed: %v", err)
	}

	// Verify TypeMeta
	verifyTypeMeta(t, original.TypeMeta(), restored.TypeMeta())

	// Verify Metadata (ObjectMeta)
	verifyObjectMeta(t, original.Metadata(), restored.Metadata())

	// Verify Spec (PodSpec)
	verifyPodSpec(t, original.Spec(), restored.Spec())

	// Verify Status (PodStatus)
	verifyPodStatus(t, original.Status(), restored.Status())
}

func verifyTypeMeta(t *testing.T, want, got clawpod.TypeMeta) {
	t.Helper()
	if want.Kind() != got.Kind() {
		t.Errorf("TypeMeta.Kind: got %q, want %q", got.Kind(), want.Kind())
	}
	if want.ApiVersion() != got.ApiVersion() {
		t.Errorf("TypeMeta.ApiVersion: got %q, want %q", got.ApiVersion(), want.ApiVersion())
	}
}

func verifyObjectMeta(t *testing.T, want, got clawpod.ObjectMeta) {
	t.Helper()
	ctx := context.Background()

	if want.Name() != got.Name() {
		t.Errorf("ObjectMeta.Name: got %q, want %q", got.Name(), want.Name())
	}
	if want.GenerateName() != got.GenerateName() {
		t.Errorf("ObjectMeta.GenerateName: got %q, want %q", got.GenerateName(), want.GenerateName())
	}
	if want.Namespace() != got.Namespace() {
		t.Errorf("ObjectMeta.Namespace: got %q, want %q", got.Namespace(), want.Namespace())
	}
	if want.SelfLink() != got.SelfLink() {
		t.Errorf("ObjectMeta.SelfLink: got %q, want %q", got.SelfLink(), want.SelfLink())
	}
	if want.Uid() != got.Uid() {
		t.Errorf("ObjectMeta.Uid: got %q, want %q", got.Uid(), want.Uid())
	}
	if want.ResourceVersion() != got.ResourceVersion() {
		t.Errorf("ObjectMeta.ResourceVersion: got %q, want %q", got.ResourceVersion(), want.ResourceVersion())
	}
	if want.Generation() != got.Generation() {
		t.Errorf("ObjectMeta.Generation: got %d, want %d", got.Generation(), want.Generation())
	}

	// Verify CreationTimestamp
	wantCT := want.CreationTimestamp()
	gotCT := got.CreationTimestamp()
	if wantCT.Seconds() != gotCT.Seconds() {
		t.Errorf("ObjectMeta.CreationTimestamp.Seconds: got %d, want %d", gotCT.Seconds(), wantCT.Seconds())
	}
	if wantCT.Nanos() != gotCT.Nanos() {
		t.Errorf("ObjectMeta.CreationTimestamp.Nanos: got %d, want %d", gotCT.Nanos(), wantCT.Nanos())
	}

	// Verify Labels
	if want.LabelsLen(ctx) != got.LabelsLen(ctx) {
		t.Errorf("ObjectMeta.LabelsLen: got %d, want %d", got.LabelsLen(ctx), want.LabelsLen(ctx))
	} else {
		for i := 0; i < want.LabelsLen(ctx); i++ {
			wantKV := want.LabelsGet(ctx, i)
			gotKV := got.LabelsGet(ctx, i)
			if wantKV.Key() != gotKV.Key() {
				t.Errorf("ObjectMeta.Labels[%d].Key: got %q, want %q", i, gotKV.Key(), wantKV.Key())
			}
			if wantKV.Value() != gotKV.Value() {
				t.Errorf("ObjectMeta.Labels[%d].Value: got %q, want %q", i, gotKV.Value(), wantKV.Value())
			}
		}
	}

	// Verify Annotations
	if want.AnnotationsLen(ctx) != got.AnnotationsLen(ctx) {
		t.Errorf("ObjectMeta.AnnotationsLen: got %d, want %d", got.AnnotationsLen(ctx), want.AnnotationsLen(ctx))
	} else {
		for i := 0; i < want.AnnotationsLen(ctx); i++ {
			wantKV := want.AnnotationsGet(ctx, i)
			gotKV := got.AnnotationsGet(ctx, i)
			if wantKV.Key() != gotKV.Key() {
				t.Errorf("ObjectMeta.Annotations[%d].Key: got %q, want %q", i, gotKV.Key(), wantKV.Key())
			}
			if wantKV.Value() != gotKV.Value() {
				t.Errorf("ObjectMeta.Annotations[%d].Value: got %q, want %q", i, gotKV.Value(), wantKV.Value())
			}
		}
	}

	// Verify OwnerReferences
	if want.OwnerReferencesLen(ctx) != got.OwnerReferencesLen(ctx) {
		t.Errorf("ObjectMeta.OwnerReferencesLen: got %d, want %d", got.OwnerReferencesLen(ctx), want.OwnerReferencesLen(ctx))
	} else {
		for i := 0; i < want.OwnerReferencesLen(ctx); i++ {
			wantOR := want.OwnerReferencesGet(ctx, i)
			gotOR := got.OwnerReferencesGet(ctx, i)
			if wantOR.ApiVersion() != gotOR.ApiVersion() {
				t.Errorf("OwnerReferences[%d].ApiVersion: got %q, want %q", i, gotOR.ApiVersion(), wantOR.ApiVersion())
			}
			if wantOR.Kind() != gotOR.Kind() {
				t.Errorf("OwnerReferences[%d].Kind: got %q, want %q", i, gotOR.Kind(), wantOR.Kind())
			}
			if wantOR.Name() != gotOR.Name() {
				t.Errorf("OwnerReferences[%d].Name: got %q, want %q", i, gotOR.Name(), wantOR.Name())
			}
			if wantOR.Uid() != gotOR.Uid() {
				t.Errorf("OwnerReferences[%d].Uid: got %q, want %q", i, gotOR.Uid(), wantOR.Uid())
			}
		}
	}

	// Verify Finalizers
	wantFin := want.Finalizers()
	gotFin := got.Finalizers()
	if wantFin.Len() != gotFin.Len() {
		t.Errorf("ObjectMeta.Finalizers.Len: got %d, want %d", gotFin.Len(), wantFin.Len())
	} else {
		for i := 0; i < wantFin.Len(); i++ {
			if wantFin.Get(i) != gotFin.Get(i) {
				t.Errorf("ObjectMeta.Finalizers[%d]: got %q, want %q", i, gotFin.Get(i), wantFin.Get(i))
			}
		}
	}
}

func verifyPodSpec(t *testing.T, want, got clawpod.PodSpec) {
	t.Helper()
	ctx := context.Background()

	// Verify Volumes
	if want.VolumesLen(ctx) != got.VolumesLen(ctx) {
		t.Errorf("PodSpec.VolumesLen: got %d, want %d", got.VolumesLen(ctx), want.VolumesLen(ctx))
	} else {
		for i := 0; i < want.VolumesLen(ctx); i++ {
			wantV := want.VolumesGet(ctx, i)
			gotV := got.VolumesGet(ctx, i)
			if wantV.Name() != gotV.Name() {
				t.Errorf("Volumes[%d].Name: got %q, want %q", i, gotV.Name(), wantV.Name())
			}
			if wantV.VolumeSource().EmptyDir().Medium() != gotV.VolumeSource().EmptyDir().Medium() {
				t.Errorf("Volumes[%d].VolumeSource.EmptyDir.Medium: got %v, want %v", i, gotV.VolumeSource().EmptyDir().Medium(), wantV.VolumeSource().EmptyDir().Medium())
			}
		}
	}

	// Verify InitContainers
	if want.InitContainersLen(ctx) != got.InitContainersLen(ctx) {
		t.Errorf("PodSpec.InitContainersLen: got %d, want %d", got.InitContainersLen(ctx), want.InitContainersLen(ctx))
	} else {
		for i := 0; i < want.InitContainersLen(ctx); i++ {
			verifyContainer(t, ctx, fmt.Sprintf("InitContainers[%d]", i), want.InitContainersGet(ctx, i), got.InitContainersGet(ctx, i))
		}
	}

	// Verify Containers
	if want.ContainersLen(ctx) != got.ContainersLen(ctx) {
		t.Errorf("PodSpec.ContainersLen: got %d, want %d", got.ContainersLen(ctx), want.ContainersLen(ctx))
	} else {
		for i := 0; i < want.ContainersLen(ctx); i++ {
			verifyContainer(t, ctx, fmt.Sprintf("Containers[%d]", i), want.ContainersGet(ctx, i), got.ContainersGet(ctx, i))
		}
	}

	// Verify scalar fields
	if want.RestartPolicy() != got.RestartPolicy() {
		t.Errorf("PodSpec.RestartPolicy: got %v, want %v", got.RestartPolicy(), want.RestartPolicy())
	}
	if want.TerminationGracePeriodSeconds() != got.TerminationGracePeriodSeconds() {
		t.Errorf("PodSpec.TerminationGracePeriodSeconds: got %d, want %d", got.TerminationGracePeriodSeconds(), want.TerminationGracePeriodSeconds())
	}
	if want.DnsPolicy() != got.DnsPolicy() {
		t.Errorf("PodSpec.DnsPolicy: got %v, want %v", got.DnsPolicy(), want.DnsPolicy())
	}

	// Verify NodeSelector
	if want.NodeSelectorLen(ctx) != got.NodeSelectorLen(ctx) {
		t.Errorf("PodSpec.NodeSelectorLen: got %d, want %d", got.NodeSelectorLen(ctx), want.NodeSelectorLen(ctx))
	} else {
		for i := 0; i < want.NodeSelectorLen(ctx); i++ {
			wantKV := want.NodeSelectorGet(ctx, i)
			gotKV := got.NodeSelectorGet(ctx, i)
			if wantKV.Key() != gotKV.Key() {
				t.Errorf("NodeSelector[%d].Key: got %q, want %q", i, gotKV.Key(), wantKV.Key())
			}
			if wantKV.Value() != gotKV.Value() {
				t.Errorf("NodeSelector[%d].Value: got %q, want %q", i, gotKV.Value(), wantKV.Value())
			}
		}
	}

	if want.ServiceAccountName() != got.ServiceAccountName() {
		t.Errorf("PodSpec.ServiceAccountName: got %q, want %q", got.ServiceAccountName(), want.ServiceAccountName())
	}
	if want.NodeName() != got.NodeName() {
		t.Errorf("PodSpec.NodeName: got %q, want %q", got.NodeName(), want.NodeName())
	}
	if want.HostNetwork() != got.HostNetwork() {
		t.Errorf("PodSpec.HostNetwork: got %v, want %v", got.HostNetwork(), want.HostNetwork())
	}
	if want.HostPid() != got.HostPid() {
		t.Errorf("PodSpec.HostPid: got %v, want %v", got.HostPid(), want.HostPid())
	}
	if want.HostIpc() != got.HostIpc() {
		t.Errorf("PodSpec.HostIpc: got %v, want %v", got.HostIpc(), want.HostIpc())
	}

	// Verify SecurityContext
	verifyPodSecurityContext(t, want.SecurityContext(), got.SecurityContext())

	if want.Hostname() != got.Hostname() {
		t.Errorf("PodSpec.Hostname: got %q, want %q", got.Hostname(), want.Hostname())
	}
	if want.Subdomain() != got.Subdomain() {
		t.Errorf("PodSpec.Subdomain: got %q, want %q", got.Subdomain(), want.Subdomain())
	}
	if want.SchedulerName() != got.SchedulerName() {
		t.Errorf("PodSpec.SchedulerName: got %q, want %q", got.SchedulerName(), want.SchedulerName())
	}

	// Verify Tolerations
	if want.TolerationsLen(ctx) != got.TolerationsLen(ctx) {
		t.Errorf("PodSpec.TolerationsLen: got %d, want %d", got.TolerationsLen(ctx), want.TolerationsLen(ctx))
	} else {
		for i := 0; i < want.TolerationsLen(ctx); i++ {
			wantT := want.TolerationsGet(ctx, i)
			gotT := got.TolerationsGet(ctx, i)
			if wantT.Key() != gotT.Key() {
				t.Errorf("Tolerations[%d].Key: got %q, want %q", i, gotT.Key(), wantT.Key())
			}
			if wantT.Operator() != gotT.Operator() {
				t.Errorf("Tolerations[%d].Operator: got %v, want %v", i, gotT.Operator(), wantT.Operator())
			}
			if wantT.Value() != gotT.Value() {
				t.Errorf("Tolerations[%d].Value: got %q, want %q", i, gotT.Value(), wantT.Value())
			}
			if wantT.Effect() != gotT.Effect() {
				t.Errorf("Tolerations[%d].Effect: got %v, want %v", i, gotT.Effect(), wantT.Effect())
			}
		}
	}

	// Verify HostAliases
	if want.HostAliasesLen(ctx) != got.HostAliasesLen(ctx) {
		t.Errorf("PodSpec.HostAliasesLen: got %d, want %d", got.HostAliasesLen(ctx), want.HostAliasesLen(ctx))
	} else {
		for i := 0; i < want.HostAliasesLen(ctx); i++ {
			wantHA := want.HostAliasesGet(ctx, i)
			gotHA := got.HostAliasesGet(ctx, i)
			if wantHA.Ip() != gotHA.Ip() {
				t.Errorf("HostAliases[%d].Ip: got %q, want %q", i, gotHA.Ip(), wantHA.Ip())
			}
			wantHN := wantHA.Hostnames()
			gotHN := gotHA.Hostnames()
			if wantHN.Len() != gotHN.Len() {
				t.Errorf("HostAliases[%d].Hostnames.Len: got %d, want %d", i, gotHN.Len(), wantHN.Len())
			} else {
				for j := 0; j < wantHN.Len(); j++ {
					if wantHN.Get(j) != gotHN.Get(j) {
						t.Errorf("HostAliases[%d].Hostnames[%d]: got %q, want %q", i, j, gotHN.Get(j), wantHN.Get(j))
					}
				}
			}
		}
	}

	if want.PriorityClassName() != got.PriorityClassName() {
		t.Errorf("PodSpec.PriorityClassName: got %q, want %q", got.PriorityClassName(), want.PriorityClassName())
	}
	if want.Priority() != got.Priority() {
		t.Errorf("PodSpec.Priority: got %d, want %d", got.Priority(), want.Priority())
	}

	// Verify DnsConfig
	verifyDnsConfig(t, want.DnsConfig(), got.DnsConfig())

	// Verify Affinity
	verifyAffinity(t, want.Affinity(), got.Affinity())
}

func verifyContainer(t *testing.T, ctx context.Context, name string, want, got clawpod.Container) {
	t.Helper()

	if want.Name() != got.Name() {
		t.Errorf("%s.Name: got %q, want %q", name, got.Name(), want.Name())
	}
	if want.Image() != got.Image() {
		t.Errorf("%s.Image: got %q, want %q", name, got.Image(), want.Image())
	}

	// Verify Command
	wantCmd := want.Command()
	gotCmd := got.Command()
	if wantCmd.Len() != gotCmd.Len() {
		t.Errorf("%s.Command.Len: got %d, want %d", name, gotCmd.Len(), wantCmd.Len())
	} else {
		for i := 0; i < wantCmd.Len(); i++ {
			if wantCmd.Get(i) != gotCmd.Get(i) {
				t.Errorf("%s.Command[%d]: got %q, want %q", name, i, gotCmd.Get(i), wantCmd.Get(i))
			}
		}
	}

	// Verify Args
	wantArgs := want.Args()
	gotArgs := got.Args()
	if wantArgs.Len() != gotArgs.Len() {
		t.Errorf("%s.Args.Len: got %d, want %d", name, gotArgs.Len(), wantArgs.Len())
	} else {
		for i := 0; i < wantArgs.Len(); i++ {
			if wantArgs.Get(i) != gotArgs.Get(i) {
				t.Errorf("%s.Args[%d]: got %q, want %q", name, i, gotArgs.Get(i), wantArgs.Get(i))
			}
		}
	}

	if want.WorkingDir() != got.WorkingDir() {
		t.Errorf("%s.WorkingDir: got %q, want %q", name, got.WorkingDir(), want.WorkingDir())
	}

	// Verify Ports
	if want.PortsLen(ctx) != got.PortsLen(ctx) {
		t.Errorf("%s.PortsLen: got %d, want %d", name, got.PortsLen(ctx), want.PortsLen(ctx))
	} else {
		for i := 0; i < want.PortsLen(ctx); i++ {
			wantP := want.PortsGet(ctx, i)
			gotP := got.PortsGet(ctx, i)
			if wantP.Name() != gotP.Name() {
				t.Errorf("%s.Ports[%d].Name: got %q, want %q", name, i, gotP.Name(), wantP.Name())
			}
			if wantP.ContainerPort() != gotP.ContainerPort() {
				t.Errorf("%s.Ports[%d].ContainerPort: got %d, want %d", name, i, gotP.ContainerPort(), wantP.ContainerPort())
			}
			if wantP.Protocol() != gotP.Protocol() {
				t.Errorf("%s.Ports[%d].Protocol: got %v, want %v", name, i, gotP.Protocol(), wantP.Protocol())
			}
		}
	}

	// Verify Env
	if want.EnvLen(ctx) != got.EnvLen(ctx) {
		t.Errorf("%s.EnvLen: got %d, want %d", name, got.EnvLen(ctx), want.EnvLen(ctx))
	} else {
		for i := 0; i < want.EnvLen(ctx); i++ {
			wantE := want.EnvGet(ctx, i)
			gotE := got.EnvGet(ctx, i)
			if wantE.Name() != gotE.Name() {
				t.Errorf("%s.Env[%d].Name: got %q, want %q", name, i, gotE.Name(), wantE.Name())
			}
			if wantE.Value() != gotE.Value() {
				t.Errorf("%s.Env[%d].Value: got %q, want %q", name, i, gotE.Value(), wantE.Value())
			}
		}
	}

	// Verify Resources
	wantRes := want.Resources()
	gotRes := got.Resources()
	if wantRes.LimitsLen(ctx) != gotRes.LimitsLen(ctx) {
		t.Errorf("%s.Resources.LimitsLen: got %d, want %d", name, gotRes.LimitsLen(ctx), wantRes.LimitsLen(ctx))
	} else {
		for i := 0; i < wantRes.LimitsLen(ctx); i++ {
			wantL := wantRes.LimitsGet(ctx, i)
			gotL := gotRes.LimitsGet(ctx, i)
			if wantL.Key() != gotL.Key() {
				t.Errorf("%s.Resources.Limits[%d].Key: got %q, want %q", name, i, gotL.Key(), wantL.Key())
			}
			if wantL.Value() != gotL.Value() {
				t.Errorf("%s.Resources.Limits[%d].Value: got %q, want %q", name, i, gotL.Value(), wantL.Value())
			}
		}
	}
	if wantRes.RequestsLen(ctx) != gotRes.RequestsLen(ctx) {
		t.Errorf("%s.Resources.RequestsLen: got %d, want %d", name, gotRes.RequestsLen(ctx), wantRes.RequestsLen(ctx))
	} else {
		for i := 0; i < wantRes.RequestsLen(ctx); i++ {
			wantR := wantRes.RequestsGet(ctx, i)
			gotR := gotRes.RequestsGet(ctx, i)
			if wantR.Key() != gotR.Key() {
				t.Errorf("%s.Resources.Requests[%d].Key: got %q, want %q", name, i, gotR.Key(), wantR.Key())
			}
			if wantR.Value() != gotR.Value() {
				t.Errorf("%s.Resources.Requests[%d].Value: got %q, want %q", name, i, gotR.Value(), wantR.Value())
			}
		}
	}

	// Verify VolumeMounts
	if want.VolumeMountsLen(ctx) != got.VolumeMountsLen(ctx) {
		t.Errorf("%s.VolumeMountsLen: got %d, want %d", name, got.VolumeMountsLen(ctx), want.VolumeMountsLen(ctx))
	} else {
		for i := 0; i < want.VolumeMountsLen(ctx); i++ {
			wantVM := want.VolumeMountsGet(ctx, i)
			gotVM := got.VolumeMountsGet(ctx, i)
			if wantVM.Name() != gotVM.Name() {
				t.Errorf("%s.VolumeMounts[%d].Name: got %q, want %q", name, i, gotVM.Name(), wantVM.Name())
			}
			if wantVM.MountPath() != gotVM.MountPath() {
				t.Errorf("%s.VolumeMounts[%d].MountPath: got %q, want %q", name, i, gotVM.MountPath(), wantVM.MountPath())
			}
			if wantVM.ReadOnly() != gotVM.ReadOnly() {
				t.Errorf("%s.VolumeMounts[%d].ReadOnly: got %v, want %v", name, i, gotVM.ReadOnly(), wantVM.ReadOnly())
			}
		}
	}

	if want.TerminationMessagePath() != got.TerminationMessagePath() {
		t.Errorf("%s.TerminationMessagePath: got %q, want %q", name, got.TerminationMessagePath(), want.TerminationMessagePath())
	}
	if want.ImagePullPolicy() != got.ImagePullPolicy() {
		t.Errorf("%s.ImagePullPolicy: got %v, want %v", name, got.ImagePullPolicy(), want.ImagePullPolicy())
	}

	// Verify Probes
	verifyProbe(t, name+".LivenessProbe", want.LivenessProbe(), got.LivenessProbe())
	verifyProbe(t, name+".ReadinessProbe", want.ReadinessProbe(), got.ReadinessProbe())
}

func verifyProbe(t *testing.T, name string, want, got clawpod.Probe) {
	t.Helper()

	wantHandler := want.Handler()
	gotHandler := got.Handler()

	wantHTTP := wantHandler.HttpGet()
	gotHTTP := gotHandler.HttpGet()

	if wantHTTP.Path() != gotHTTP.Path() {
		t.Errorf("%s.Handler.HttpGet.Path: got %q, want %q", name, gotHTTP.Path(), wantHTTP.Path())
	}
	if wantHTTP.Port().IntVal() != gotHTTP.Port().IntVal() {
		t.Errorf("%s.Handler.HttpGet.Port.IntVal: got %d, want %d", name, gotHTTP.Port().IntVal(), wantHTTP.Port().IntVal())
	}

	if want.InitialDelaySeconds() != got.InitialDelaySeconds() {
		t.Errorf("%s.InitialDelaySeconds: got %d, want %d", name, got.InitialDelaySeconds(), want.InitialDelaySeconds())
	}
	if want.PeriodSeconds() != got.PeriodSeconds() {
		t.Errorf("%s.PeriodSeconds: got %d, want %d", name, got.PeriodSeconds(), want.PeriodSeconds())
	}
}

func verifyPodSecurityContext(t *testing.T, want, got clawpod.PodSecurityContext) {
	t.Helper()

	if want.RunAsUser() != got.RunAsUser() {
		t.Errorf("PodSecurityContext.RunAsUser: got %d, want %d", got.RunAsUser(), want.RunAsUser())
	}
	if want.RunAsGroup() != got.RunAsGroup() {
		t.Errorf("PodSecurityContext.RunAsGroup: got %d, want %d", got.RunAsGroup(), want.RunAsGroup())
	}
	if want.RunAsNonRoot() != got.RunAsNonRoot() {
		t.Errorf("PodSecurityContext.RunAsNonRoot: got %v, want %v", got.RunAsNonRoot(), want.RunAsNonRoot())
	}
	if want.FsGroup() != got.FsGroup() {
		t.Errorf("PodSecurityContext.FsGroup: got %d, want %d", got.FsGroup(), want.FsGroup())
	}

	// Verify SupplementalGroups
	wantSG := want.SupplementalGroups()
	gotSG := got.SupplementalGroups()
	if wantSG.Len() != gotSG.Len() {
		t.Errorf("PodSecurityContext.SupplementalGroups.Len: got %d, want %d", gotSG.Len(), wantSG.Len())
	} else {
		for i := 0; i < wantSG.Len(); i++ {
			if wantSG.Get(i) != gotSG.Get(i) {
				t.Errorf("PodSecurityContext.SupplementalGroups[%d]: got %d, want %d", i, gotSG.Get(i), wantSG.Get(i))
			}
		}
	}
}

func verifyDnsConfig(t *testing.T, want, got clawpod.PodDNSConfig) {
	t.Helper()

	wantNS := want.Nameservers()
	gotNS := got.Nameservers()
	if wantNS.Len() != gotNS.Len() {
		t.Errorf("DnsConfig.Nameservers.Len: got %d, want %d", gotNS.Len(), wantNS.Len())
	} else {
		for i := 0; i < wantNS.Len(); i++ {
			if wantNS.Get(i) != gotNS.Get(i) {
				t.Errorf("DnsConfig.Nameservers[%d]: got %q, want %q", i, gotNS.Get(i), wantNS.Get(i))
			}
		}
	}

	wantSearches := want.Searches()
	gotSearches := got.Searches()
	if wantSearches.Len() != gotSearches.Len() {
		t.Errorf("DnsConfig.Searches.Len: got %d, want %d", gotSearches.Len(), wantSearches.Len())
	} else {
		for i := 0; i < wantSearches.Len(); i++ {
			if wantSearches.Get(i) != gotSearches.Get(i) {
				t.Errorf("DnsConfig.Searches[%d]: got %q, want %q", i, gotSearches.Get(i), wantSearches.Get(i))
			}
		}
	}
}

func verifyAffinity(t *testing.T, want, got clawpod.Affinity) {
	t.Helper()
	ctx := context.Background()

	// Verify NodeAffinity
	wantNA := want.NodeAffinity()
	gotNA := got.NodeAffinity()

	wantNS := wantNA.RequiredDuringSchedulingIgnoredDuringExecution()
	gotNS := gotNA.RequiredDuringSchedulingIgnoredDuringExecution()

	if wantNS.NodeSelectorTermsLen(ctx) != gotNS.NodeSelectorTermsLen(ctx) {
		t.Errorf("Affinity.NodeAffinity.NodeSelectorTermsLen: got %d, want %d", gotNS.NodeSelectorTermsLen(ctx), wantNS.NodeSelectorTermsLen(ctx))
		return
	}

	for i := 0; i < wantNS.NodeSelectorTermsLen(ctx); i++ {
		wantTerm := wantNS.NodeSelectorTermsGet(ctx, i)
		gotTerm := gotNS.NodeSelectorTermsGet(ctx, i)

		if wantTerm.MatchExpressionsLen(ctx) != gotTerm.MatchExpressionsLen(ctx) {
			t.Errorf("Affinity.NodeSelectorTerms[%d].MatchExpressionsLen: got %d, want %d", i, gotTerm.MatchExpressionsLen(ctx), wantTerm.MatchExpressionsLen(ctx))
			continue
		}

		for j := 0; j < wantTerm.MatchExpressionsLen(ctx); j++ {
			wantExpr := wantTerm.MatchExpressionsGet(ctx, j)
			gotExpr := gotTerm.MatchExpressionsGet(ctx, j)

			if wantExpr.Key() != gotExpr.Key() {
				t.Errorf("Affinity.MatchExpressions[%d].Key: got %q, want %q", j, gotExpr.Key(), wantExpr.Key())
			}
			if wantExpr.Operator() != gotExpr.Operator() {
				t.Errorf("Affinity.MatchExpressions[%d].Operator: got %v, want %v", j, gotExpr.Operator(), wantExpr.Operator())
			}
			wantValues := wantExpr.Values()
			gotValues := gotExpr.Values()
			if wantValues.Len() != gotValues.Len() {
				t.Errorf("Affinity.MatchExpressions[%d].Values.Len: got %d, want %d", j, gotValues.Len(), wantValues.Len())
			} else {
				for k := 0; k < wantValues.Len(); k++ {
					if wantValues.Get(k) != gotValues.Get(k) {
						t.Errorf("Affinity.MatchExpressions[%d].Values[%d]: got %q, want %q", j, k, gotValues.Get(k), wantValues.Get(k))
					}
				}
			}
		}
	}

	// Verify PodAffinity
	wantPA := want.PodAffinity()
	gotPA := got.PodAffinity()

	if wantPA.RequiredDuringSchedulingIgnoredDuringExecutionLen(ctx) != gotPA.RequiredDuringSchedulingIgnoredDuringExecutionLen(ctx) {
		t.Errorf("Affinity.PodAffinity.RequiredTermsLen: got %d, want %d", gotPA.RequiredDuringSchedulingIgnoredDuringExecutionLen(ctx), wantPA.RequiredDuringSchedulingIgnoredDuringExecutionLen(ctx))
		return
	}

	for i := 0; i < wantPA.RequiredDuringSchedulingIgnoredDuringExecutionLen(ctx); i++ {
		wantPATerm := wantPA.RequiredDuringSchedulingIgnoredDuringExecutionGet(ctx, i)
		gotPATerm := gotPA.RequiredDuringSchedulingIgnoredDuringExecutionGet(ctx, i)

		if wantPATerm.TopologyKey() != gotPATerm.TopologyKey() {
			t.Errorf("Affinity.PodAffinityTerms[%d].TopologyKey: got %q, want %q", i, gotPATerm.TopologyKey(), wantPATerm.TopologyKey())
		}

		wantLS := wantPATerm.LabelSelector()
		gotLS := gotPATerm.LabelSelector()

		if wantLS.MatchLabelsLen(ctx) != gotLS.MatchLabelsLen(ctx) {
			t.Errorf("Affinity.PodAffinityTerms[%d].LabelSelector.MatchLabelsLen: got %d, want %d", i, gotLS.MatchLabelsLen(ctx), wantLS.MatchLabelsLen(ctx))
		} else {
			for j := 0; j < wantLS.MatchLabelsLen(ctx); j++ {
				wantML := wantLS.MatchLabelsGet(ctx, j)
				gotML := gotLS.MatchLabelsGet(ctx, j)
				if wantML.Key() != gotML.Key() {
					t.Errorf("Affinity.MatchLabels[%d].Key: got %q, want %q", j, gotML.Key(), wantML.Key())
				}
				if wantML.Value() != gotML.Value() {
					t.Errorf("Affinity.MatchLabels[%d].Value: got %q, want %q", j, gotML.Value(), wantML.Value())
				}
			}
		}
	}
}

func verifyPodStatus(t *testing.T, want, got clawpod.PodStatus) {
	t.Helper()
	ctx := context.Background()

	if want.Phase() != got.Phase() {
		t.Errorf("PodStatus.Phase: got %v, want %v", got.Phase(), want.Phase())
	}

	// Verify Conditions
	if want.ConditionsLen(ctx) != got.ConditionsLen(ctx) {
		t.Errorf("PodStatus.ConditionsLen: got %d, want %d", got.ConditionsLen(ctx), want.ConditionsLen(ctx))
	} else {
		for i := 0; i < want.ConditionsLen(ctx); i++ {
			wantC := want.ConditionsGet(ctx, i)
			gotC := got.ConditionsGet(ctx, i)
			if wantC.Type() != gotC.Type() {
				t.Errorf("PodConditions[%d].Type: got %v, want %v", i, gotC.Type(), wantC.Type())
			}
			if wantC.Status() != gotC.Status() {
				t.Errorf("PodConditions[%d].Status: got %v, want %v", i, gotC.Status(), wantC.Status())
			}
			if wantC.LastTransitionTime().Seconds() != gotC.LastTransitionTime().Seconds() {
				t.Errorf("PodConditions[%d].LastTransitionTime.Seconds: got %d, want %d", i, gotC.LastTransitionTime().Seconds(), wantC.LastTransitionTime().Seconds())
			}
		}
	}

	if want.Message() != got.Message() {
		t.Errorf("PodStatus.Message: got %q, want %q", got.Message(), want.Message())
	}
	if want.HostIp() != got.HostIp() {
		t.Errorf("PodStatus.HostIp: got %q, want %q", got.HostIp(), want.HostIp())
	}
	if want.PodIp() != got.PodIp() {
		t.Errorf("PodStatus.PodIp: got %q, want %q", got.PodIp(), want.PodIp())
	}

	// Verify PodIPs
	if want.PodIpsLen(ctx) != got.PodIpsLen(ctx) {
		t.Errorf("PodStatus.PodIpsLen: got %d, want %d", got.PodIpsLen(ctx), want.PodIpsLen(ctx))
	} else {
		for i := 0; i < want.PodIpsLen(ctx); i++ {
			if want.PodIpsGet(ctx, i).Ip() != got.PodIpsGet(ctx, i).Ip() {
				t.Errorf("PodIPs[%d].Ip: got %q, want %q", i, got.PodIpsGet(ctx, i).Ip(), want.PodIpsGet(ctx, i).Ip())
			}
		}
	}

	// Verify StartTime
	if want.StartTime().Seconds() != got.StartTime().Seconds() {
		t.Errorf("PodStatus.StartTime.Seconds: got %d, want %d", got.StartTime().Seconds(), want.StartTime().Seconds())
	}

	// Verify ContainerStatuses
	if want.ContainerStatusesLen(ctx) != got.ContainerStatusesLen(ctx) {
		t.Errorf("PodStatus.ContainerStatusesLen: got %d, want %d", got.ContainerStatusesLen(ctx), want.ContainerStatusesLen(ctx))
	} else {
		for i := 0; i < want.ContainerStatusesLen(ctx); i++ {
			wantCS := want.ContainerStatusesGet(ctx, i)
			gotCS := got.ContainerStatusesGet(ctx, i)
			if wantCS.Name() != gotCS.Name() {
				t.Errorf("ContainerStatuses[%d].Name: got %q, want %q", i, gotCS.Name(), wantCS.Name())
			}
			if wantCS.State().Running().StartedAt().Seconds() != gotCS.State().Running().StartedAt().Seconds() {
				t.Errorf("ContainerStatuses[%d].State.Running.StartedAt.Seconds: got %d, want %d", i, gotCS.State().Running().StartedAt().Seconds(), wantCS.State().Running().StartedAt().Seconds())
			}
			if wantCS.Ready() != gotCS.Ready() {
				t.Errorf("ContainerStatuses[%d].Ready: got %v, want %v", i, gotCS.Ready(), wantCS.Ready())
			}
			if wantCS.RestartCount() != gotCS.RestartCount() {
				t.Errorf("ContainerStatuses[%d].RestartCount: got %d, want %d", i, gotCS.RestartCount(), wantCS.RestartCount())
			}
			if wantCS.Image() != gotCS.Image() {
				t.Errorf("ContainerStatuses[%d].Image: got %q, want %q", i, gotCS.Image(), wantCS.Image())
			}
			if wantCS.ImageId() != gotCS.ImageId() {
				t.Errorf("ContainerStatuses[%d].ImageId: got %q, want %q", i, gotCS.ImageId(), wantCS.ImageId())
			}
			if wantCS.ContainerId() != gotCS.ContainerId() {
				t.Errorf("ContainerStatuses[%d].ContainerId: got %q, want %q", i, gotCS.ContainerId(), wantCS.ContainerId())
			}
		}
	}

	if want.QosClass() != got.QosClass() {
		t.Errorf("PodStatus.QosClass: got %v, want %v", got.QosClass(), want.QosClass())
	}
}
