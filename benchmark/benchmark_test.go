package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/go-json-experiment/json"
	"google.golang.org/protobuf/proto"

	capnpod "github.com/bearlytools/claw/benchmark/msgs/capn"
	clawpod "github.com/bearlytools/claw/benchmark/msgs/claw"
	protopod "github.com/bearlytools/claw/benchmark/msgs/proto"
)

// jsonMarshal is a wrapper for the go-json-experiment Marshal function
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

const listSize = 10

// Helper to create a populated Proto Pod
func createProtoPod() *protopod.Pod {
	pod := &protopod.Pod{
		TypeMeta: &protopod.TypeMeta{
			Kind:       "Pod",
			ApiVersion: "v1",
		},
		Metadata: createProtoObjectMeta(),
		Spec:     createProtoPodSpec(),
		Status:   createProtoPodStatus(),
	}
	return pod
}

func createProtoObjectMeta() *protopod.ObjectMeta {
	labels := make(map[string]string, listSize)
	annotations := make(map[string]string, listSize)
	for i := 0; i < listSize; i++ {
		labels[fmt.Sprintf("label-key-%d", i)] = fmt.Sprintf("label-value-%d", i)
		annotations[fmt.Sprintf("annotation-key-%d", i)] = fmt.Sprintf("annotation-value-%d", i)
	}

	ownerRefs := make([]*protopod.OwnerReference, listSize)
	for i := 0; i < listSize; i++ {
		ownerRefs[i] = &protopod.OwnerReference{
			ApiVersion: "v1",
			Kind:       "ReplicaSet",
			Name:       fmt.Sprintf("owner-%d", i),
			Uid:        fmt.Sprintf("uid-%d", i),
		}
	}

	finalizers := make([]string, listSize)
	for i := 0; i < listSize; i++ {
		finalizers[i] = fmt.Sprintf("finalizer-%d", i)
	}

	return &protopod.ObjectMeta{
		Name:            "test-pod",
		GenerateName:    "test-pod-",
		Namespace:       "default",
		SelfLink:        "/api/v1/namespaces/default/pods/test-pod",
		Uid:             "12345678-1234-1234-1234-123456789012",
		ResourceVersion: "12345",
		Generation:      1,
		CreationTimestamp: &protopod.Time{
			Seconds: 1703721600,
			Nanos:   0,
		},
		Labels:          labels,
		Annotations:     annotations,
		OwnerReferences: ownerRefs,
		Finalizers:      finalizers,
	}
}

func createProtoPodSpec() *protopod.PodSpec {
	containers := make([]*protopod.Container, listSize)
	for i := 0; i < listSize; i++ {
		containers[i] = createProtoContainer(fmt.Sprintf("container-%d", i))
	}

	initContainers := make([]*protopod.Container, 2)
	for i := 0; i < 2; i++ {
		initContainers[i] = createProtoContainer(fmt.Sprintf("init-container-%d", i))
	}

	volumes := make([]*protopod.Volume, listSize)
	for i := 0; i < listSize; i++ {
		volumes[i] = &protopod.Volume{
			Name: fmt.Sprintf("volume-%d", i),
			VolumeSource: &protopod.VolumeSource{
				EmptyDir: &protopod.EmptyDirVolumeSource{
					Medium: protopod.StorageMedium_STORAGE_MEDIUM_MEMORY,
				},
			},
		}
	}

	tolerations := make([]*protopod.Toleration, listSize)
	for i := 0; i < listSize; i++ {
		tolerations[i] = &protopod.Toleration{
			Key:      fmt.Sprintf("key-%d", i),
			Operator: protopod.TolerationOperator_TOLERATION_OPERATOR_EQUAL,
			Value:    fmt.Sprintf("value-%d", i),
			Effect:   protopod.TaintEffect_TAINT_EFFECT_NO_SCHEDULE,
		}
	}

	hostAliases := make([]*protopod.HostAlias, listSize)
	for i := 0; i < listSize; i++ {
		hostnames := make([]string, 3)
		for j := 0; j < 3; j++ {
			hostnames[j] = fmt.Sprintf("host-%d-%d.example.com", i, j)
		}
		hostAliases[i] = &protopod.HostAlias{
			Ip:        fmt.Sprintf("10.0.0.%d", i),
			Hostnames: hostnames,
		}
	}

	nodeSelector := make(map[string]string, listSize)
	for i := 0; i < listSize; i++ {
		nodeSelector[fmt.Sprintf("node-key-%d", i)] = fmt.Sprintf("node-value-%d", i)
	}

	return &protopod.PodSpec{
		Volumes:                       volumes,
		InitContainers:                initContainers,
		Containers:                    containers,
		RestartPolicy:                 protopod.RestartPolicy_RESTART_POLICY_ALWAYS,
		TerminationGracePeriodSeconds: proto.Int64(30),
		DnsPolicy:                     protopod.DNSPolicy_DNS_POLICY_CLUSTER_FIRST,
		NodeSelector:                  nodeSelector,
		ServiceAccountName:            "default",
		NodeName:                      "node-1",
		HostNetwork:                   false,
		HostPid:                       false,
		HostIpc:                       false,
		SecurityContext:               createProtoPodSecurityContext(),
		Hostname:                      "test-pod",
		Subdomain:                     "test-subdomain",
		SchedulerName:                 "default-scheduler",
		Tolerations:                   tolerations,
		HostAliases:                   hostAliases,
		PriorityClassName:             "high-priority",
		Priority:                      proto.Int32(1000),
		DnsConfig: &protopod.PodDNSConfig{
			Nameservers: []string{"8.8.8.8", "8.8.4.4"},
			Searches:    []string{"default.svc.cluster.local", "svc.cluster.local"},
		},
		Affinity: createProtoAffinity(),
	}
}

func createProtoContainer(name string) *protopod.Container {
	command := make([]string, 3)
	args := make([]string, listSize)
	for i := 0; i < 3; i++ {
		command[i] = fmt.Sprintf("/bin/cmd%d", i)
	}
	for i := 0; i < listSize; i++ {
		args[i] = fmt.Sprintf("--arg%d=value%d", i, i)
	}

	ports := make([]*protopod.ContainerPort, 3)
	for i := 0; i < 3; i++ {
		ports[i] = &protopod.ContainerPort{
			Name:          fmt.Sprintf("port-%d", i),
			ContainerPort: int32(8080 + i),
			Protocol:      protopod.Protocol_PROTOCOL_TCP,
		}
	}

	env := make([]*protopod.EnvVar, listSize)
	for i := 0; i < listSize; i++ {
		env[i] = &protopod.EnvVar{
			Name:  fmt.Sprintf("ENV_%d", i),
			Value: fmt.Sprintf("value-%d", i),
		}
	}

	volumeMounts := make([]*protopod.VolumeMount, listSize)
	for i := 0; i < listSize; i++ {
		volumeMounts[i] = &protopod.VolumeMount{
			Name:      fmt.Sprintf("volume-%d", i),
			MountPath: fmt.Sprintf("/mnt/volume-%d", i),
			ReadOnly:  i%2 == 0,
		}
	}

	return &protopod.Container{
		Name:       name,
		Image:      "nginx:latest",
		Command:    command,
		Args:       args,
		WorkingDir: "/app",
		Ports:      ports,
		Env:        env,
		Resources: &protopod.ResourceRequirements{
			Limits: map[string]string{
				"cpu":    "1000m",
				"memory": "1Gi",
			},
			Requests: map[string]string{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		},
		VolumeMounts:           volumeMounts,
		TerminationMessagePath: "/dev/termination-log",
		ImagePullPolicy:        protopod.PullPolicy_PULL_POLICY_IF_NOT_PRESENT,
		LivenessProbe: &protopod.Probe{
			Handler: &protopod.ProbeHandler{
				HttpGet: &protopod.HTTPGetAction{
					Path: "/healthz",
					Port: &protopod.IntOrString{Value: &protopod.IntOrString_IntVal{IntVal: 8080}},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
		},
		ReadinessProbe: &protopod.Probe{
			Handler: &protopod.ProbeHandler{
				HttpGet: &protopod.HTTPGetAction{
					Path: "/ready",
					Port: &protopod.IntOrString{Value: &protopod.IntOrString_IntVal{IntVal: 8080}},
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}
}

func createProtoPodSecurityContext() *protopod.PodSecurityContext {
	supplementalGroups := make([]int64, listSize)
	for i := 0; i < listSize; i++ {
		supplementalGroups[i] = int64(1000 + i)
	}

	return &protopod.PodSecurityContext{
		RunAsUser:          proto.Int64(1000),
		RunAsGroup:         proto.Int64(1000),
		RunAsNonRoot:       proto.Bool(true),
		SupplementalGroups: supplementalGroups,
		FsGroup:            proto.Int64(2000),
	}
}

func createProtoAffinity() *protopod.Affinity {
	return &protopod.Affinity{
		NodeAffinity: &protopod.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &protopod.NodeSelector{
				NodeSelectorTerms: []*protopod.NodeSelectorTerm{
					{
						MatchExpressions: []*protopod.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/os",
								Operator: protopod.NodeSelectorOperator_NODE_SELECTOR_OPERATOR_IN,
								Values:   []string{"linux"},
							},
						},
					},
				},
			},
		},
		PodAffinity: &protopod.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []*protopod.PodAffinityTerm{
				{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: &protopod.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
			},
		},
	}
}

func createProtoPodStatus() *protopod.PodStatus {
	conditions := make([]*protopod.PodCondition, 4)
	condTypes := []protopod.PodConditionType{
		protopod.PodConditionType_POD_CONDITION_TYPE_INITIALIZED,
		protopod.PodConditionType_POD_CONDITION_TYPE_READY,
		protopod.PodConditionType_POD_CONDITION_TYPE_CONTAINERS_READY,
		protopod.PodConditionType_POD_CONDITION_TYPE_POD_SCHEDULED,
	}
	for i := 0; i < 4; i++ {
		conditions[i] = &protopod.PodCondition{
			Type:   condTypes[i],
			Status: protopod.ConditionStatus_CONDITION_STATUS_TRUE,
			LastTransitionTime: &protopod.Time{
				Seconds: 1703721600,
				Nanos:   0,
			},
		}
	}

	containerStatuses := make([]*protopod.ContainerStatus, listSize)
	for i := 0; i < listSize; i++ {
		containerStatuses[i] = &protopod.ContainerStatus{
			Name: fmt.Sprintf("container-%d", i),
			State: &protopod.ContainerState{
				Running: &protopod.ContainerStateRunning{
					StartedAt: &protopod.Time{Seconds: 1703721600},
				},
			},
			Ready:        true,
			RestartCount: 0,
			Image:        "nginx:latest",
			ImageId:      "docker://sha256:abc123",
			ContainerId:  fmt.Sprintf("docker://container-%d", i),
		}
	}

	podIPs := make([]*protopod.PodIP, 2)
	podIPs[0] = &protopod.PodIP{Ip: "10.244.0.5"}
	podIPs[1] = &protopod.PodIP{Ip: "fd00::5"}

	return &protopod.PodStatus{
		Phase:      protopod.PodPhase_POD_PHASE_RUNNING,
		Conditions: conditions,
		Message:    "Pod is running",
		HostIp:     "192.168.1.100",
		PodIp:      "10.244.0.5",
		PodIps:     podIPs,
		StartTime: &protopod.Time{
			Seconds: 1703721600,
			Nanos:   0,
		},
		ContainerStatuses: containerStatuses,
		QosClass:          protopod.PodQOSClass_POD_QOS_CLASS_BURSTABLE,
	}
}

// Helper to create a populated Claw Pod
func createClawPod() clawpod.Pod {
	pod := clawpod.NewPod(context.Background())
	pod = pod.SetTypeMeta(createClawTypeMeta())
	pod = pod.SetMetadata(createClawObjectMeta())
	pod = pod.SetSpec(createClawPodSpec())
	pod = pod.SetStatus(createClawPodStatus())
	return pod
}

func createClawTypeMeta() clawpod.TypeMeta {
	tm := clawpod.NewTypeMeta(context.Background())
	tm = tm.SetKind("Pod")
	tm = tm.SetApiVersion("v1")
	return tm
}

func createClawObjectMeta() clawpod.ObjectMeta {
	om := clawpod.NewObjectMeta(context.Background())
	om = om.SetName("test-pod")
	om = om.SetGenerateName("test-pod-")
	om = om.SetNamespace("default")
	om = om.SetSelfLink("/api/v1/namespaces/default/pods/test-pod")
	om = om.SetUid("12345678-1234-1234-1234-123456789012")
	om = om.SetResourceVersion("12345")
	om = om.SetGeneration(1)

	ct := clawpod.NewTime(context.Background())
	ct = ct.SetSeconds(1703721600)
	ct = ct.SetNanos(0)
	om = om.SetCreationTimestamp(ct)

	labels := make([]clawpod.KeyValue, listSize)
	for i := 0; i < listSize; i++ {
		kv := clawpod.NewKeyValue(context.Background())
		kv = kv.SetKey(fmt.Sprintf("label-key-%d", i))
		kv = kv.SetValue(fmt.Sprintf("label-value-%d", i))
		labels[i] = kv
	}
	om.LabelsAppend(context.Background(), labels...)

	annotations := make([]clawpod.KeyValue, listSize)
	for i := 0; i < listSize; i++ {
		kv := clawpod.NewKeyValue(context.Background())
		kv = kv.SetKey(fmt.Sprintf("annotation-key-%d", i))
		kv = kv.SetValue(fmt.Sprintf("annotation-value-%d", i))
		annotations[i] = kv
	}
	om.AnnotationsAppend(context.Background(), annotations...)

	ownerRefs := make([]clawpod.OwnerReference, listSize)
	for i := 0; i < listSize; i++ {
		or := clawpod.NewOwnerReference(context.Background())
		or = or.SetApiVersion("v1")
		or = or.SetKind("ReplicaSet")
		or = or.SetName(fmt.Sprintf("owner-%d", i))
		or = or.SetUid(fmt.Sprintf("uid-%d", i))
		ownerRefs[i] = or
	}
	om.OwnerReferencesAppend(context.Background(), ownerRefs...)

	finalizers := om.Finalizers()
	for i := 0; i < listSize; i++ {
		finalizers.Append(fmt.Sprintf("finalizer-%d", i))
	}

	return om
}

func createClawPodSpec() clawpod.PodSpec {
	ps := clawpod.NewPodSpec(context.Background())

	volumes := make([]clawpod.Volume, listSize)
	for i := 0; i < listSize; i++ {
		v := clawpod.NewVolume(context.Background())
		v = v.SetName(fmt.Sprintf("volume-%d", i))
		vs := clawpod.NewVolumeSource(context.Background())
		ed := clawpod.NewEmptyDirVolumeSource(context.Background())
		ed = ed.SetMedium(clawpod.StorageMediumMemory)
		vs = vs.SetEmptyDir(ed)
		v = v.SetVolumeSource(vs)
		volumes[i] = v
	}
	ps.VolumesAppend(context.Background(), volumes...)

	initContainers := make([]clawpod.Container, 2)
	for i := 0; i < 2; i++ {
		initContainers[i] = createClawContainer(fmt.Sprintf("init-container-%d", i))
	}
	ps.InitContainersAppend(context.Background(), initContainers...)

	containers := make([]clawpod.Container, listSize)
	for i := 0; i < listSize; i++ {
		containers[i] = createClawContainer(fmt.Sprintf("container-%d", i))
	}
	ps.ContainersAppend(context.Background(), containers...)

	ps = ps.SetRestartPolicy(clawpod.RestartPolicyAlways)
	ps = ps.SetTerminationGracePeriodSeconds(30)
	ps = ps.SetDnsPolicy(clawpod.DNSPolicyClusterFirst)

	nodeSelector := make([]clawpod.KeyValue, listSize)
	for i := 0; i < listSize; i++ {
		kv := clawpod.NewKeyValue(context.Background())
		kv = kv.SetKey(fmt.Sprintf("node-key-%d", i))
		kv = kv.SetValue(fmt.Sprintf("node-value-%d", i))
		nodeSelector[i] = kv
	}
	ps.NodeSelectorAppend(context.Background(), nodeSelector...)

	ps = ps.SetServiceAccountName("default")
	ps = ps.SetNodeName("node-1")
	ps = ps.SetHostNetwork(false)
	ps = ps.SetHostPid(false)
	ps = ps.SetHostIpc(false)
	ps = ps.SetSecurityContext(createClawPodSecurityContext())
	ps = ps.SetHostname("test-pod")
	ps = ps.SetSubdomain("test-subdomain")
	ps = ps.SetSchedulerName("default-scheduler")

	tolerations := make([]clawpod.Toleration, listSize)
	for i := 0; i < listSize; i++ {
		t := clawpod.NewToleration(context.Background())
		t = t.SetKey(fmt.Sprintf("key-%d", i))
		t = t.SetOperator(clawpod.TolerationOperatorEqual)
		t = t.SetValue(fmt.Sprintf("value-%d", i))
		t = t.SetEffect(clawpod.TaintEffectNoSchedule)
		tolerations[i] = t
	}
	ps.TolerationsAppend(context.Background(), tolerations...)

	hostAliases := make([]clawpod.HostAlias, listSize)
	for i := 0; i < listSize; i++ {
		ha := clawpod.NewHostAlias(context.Background())
		ha = ha.SetIp(fmt.Sprintf("10.0.0.%d", i))
		hostnames := ha.Hostnames()
		for j := 0; j < 3; j++ {
			hostnames.Append(fmt.Sprintf("host-%d-%d.example.com", i, j))
		}
		hostAliases[i] = ha
	}
	ps.HostAliasesAppend(context.Background(), hostAliases...)

	ps = ps.SetPriorityClassName("high-priority")
	ps = ps.SetPriority(1000)

	dnsConfig := clawpod.NewPodDNSConfig(context.Background())
	dnsConfig.Nameservers().Append("8.8.8.8", "8.8.4.4")
	dnsConfig.Searches().Append("default.svc.cluster.local", "svc.cluster.local")
	ps = ps.SetDnsConfig(dnsConfig)

	ps = ps.SetAffinity(createClawAffinity())

	return ps
}

func createClawContainer(name string) clawpod.Container {
	c := clawpod.NewContainer(context.Background())
	c = c.SetName(name)
	c = c.SetImage("nginx:latest")

	command := c.Command()
	for i := 0; i < 3; i++ {
		command.Append(fmt.Sprintf("/bin/cmd%d", i))
	}

	args := c.Args()
	for i := 0; i < listSize; i++ {
		args.Append(fmt.Sprintf("--arg%d=value%d", i, i))
	}

	c = c.SetWorkingDir("/app")

	ports := make([]clawpod.ContainerPort, 3)
	for i := 0; i < 3; i++ {
		p := clawpod.NewContainerPort(context.Background())
		p = p.SetName(fmt.Sprintf("port-%d", i))
		p = p.SetContainerPort(int32(8080 + i))
		p = p.SetProtocol(clawpod.ProtocolTcp)
		ports[i] = p
	}
	c.PortsAppend(context.Background(), ports...)

	env := make([]clawpod.EnvVar, listSize)
	for i := 0; i < listSize; i++ {
		e := clawpod.NewEnvVar(context.Background())
		e = e.SetName(fmt.Sprintf("ENV_%d", i))
		e = e.SetValue(fmt.Sprintf("value-%d", i))
		env[i] = e
	}
	c.EnvAppend(context.Background(), env...)

	resources := clawpod.NewResourceRequirements(context.Background())
	limits := make([]clawpod.KeyValue, 2)
	kv1 := clawpod.NewKeyValue(context.Background())
	kv1 = kv1.SetKey("cpu")
	kv1 = kv1.SetValue("1000m")
	limits[0] = kv1
	kv2 := clawpod.NewKeyValue(context.Background())
	kv2 = kv2.SetKey("memory")
	kv2 = kv2.SetValue("1Gi")
	limits[1] = kv2
	resources.LimitsAppend(context.Background(), limits...)

	requests := make([]clawpod.KeyValue, 2)
	kv3 := clawpod.NewKeyValue(context.Background())
	kv3 = kv3.SetKey("cpu")
	kv3 = kv3.SetValue("500m")
	requests[0] = kv3
	kv4 := clawpod.NewKeyValue(context.Background())
	kv4 = kv4.SetKey("memory")
	kv4 = kv4.SetValue("512Mi")
	requests[1] = kv4
	resources.RequestsAppend(context.Background(), requests...)
	c = c.SetResources(resources)

	volumeMounts := make([]clawpod.VolumeMount, listSize)
	for i := 0; i < listSize; i++ {
		vm := clawpod.NewVolumeMount(context.Background())
		vm = vm.SetName(fmt.Sprintf("volume-%d", i))
		vm = vm.SetMountPath(fmt.Sprintf("/mnt/volume-%d", i))
		vm = vm.SetReadOnly(i%2 == 0)
		volumeMounts[i] = vm
	}
	c.VolumeMountsAppend(context.Background(), volumeMounts...)

	c = c.SetTerminationMessagePath("/dev/termination-log")
	c = c.SetImagePullPolicy(clawpod.PullPolicyIfNotPresent)

	livenessProbe := clawpod.NewProbe(context.Background())
	handler := clawpod.NewProbeHandler(context.Background())
	httpGet := clawpod.NewHTTPGetAction(context.Background())
	httpGet = httpGet.SetPath("/healthz")
	port := clawpod.NewIntOrString(context.Background())
	port = port.SetIntVal(8080)
	httpGet = httpGet.SetPort(port)
	handler = handler.SetHttpGet(httpGet)
	livenessProbe = livenessProbe.SetHandler(handler)
	livenessProbe = livenessProbe.SetInitialDelaySeconds(10)
	livenessProbe = livenessProbe.SetPeriodSeconds(30)
	c = c.SetLivenessProbe(livenessProbe)

	readinessProbe := clawpod.NewProbe(context.Background())
	handler2 := clawpod.NewProbeHandler(context.Background())
	httpGet2 := clawpod.NewHTTPGetAction(context.Background())
	httpGet2 = httpGet2.SetPath("/ready")
	port2 := clawpod.NewIntOrString(context.Background())
	port2 = port2.SetIntVal(8080)
	httpGet2 = httpGet2.SetPort(port2)
	handler2 = handler2.SetHttpGet(httpGet2)
	readinessProbe = readinessProbe.SetHandler(handler2)
	readinessProbe = readinessProbe.SetInitialDelaySeconds(5)
	readinessProbe = readinessProbe.SetPeriodSeconds(10)
	c = c.SetReadinessProbe(readinessProbe)

	return c
}

func createClawPodSecurityContext() clawpod.PodSecurityContext {
	psc := clawpod.NewPodSecurityContext(context.Background())
	psc = psc.SetRunAsUser(1000)
	psc = psc.SetRunAsGroup(1000)
	psc = psc.SetRunAsNonRoot(true)

	supplementalGroups := psc.SupplementalGroups()
	for i := 0; i < listSize; i++ {
		supplementalGroups.Append(int64(1000 + i))
	}
	psc = psc.SetFsGroup(2000)

	return psc
}

func createClawAffinity() clawpod.Affinity {
	aff := clawpod.NewAffinity(context.Background())

	nodeAff := clawpod.NewNodeAffinity(context.Background())
	nodeSelector := clawpod.NewNodeSelector(context.Background())
	terms := make([]clawpod.NodeSelectorTerm, 1)
	term := clawpod.NewNodeSelectorTerm(context.Background())
	reqs := make([]clawpod.NodeSelectorRequirement, 1)
	req := clawpod.NewNodeSelectorRequirement(context.Background())
	req = req.SetKey("kubernetes.io/os")
	req = req.SetOperator(clawpod.NodeSelectorOperatorIn)
	req.Values().Append("linux")
	reqs[0] = req
	term.MatchExpressionsAppend(context.Background(), reqs...)
	terms[0] = term
	nodeSelector.NodeSelectorTermsAppend(context.Background(), terms...)
	nodeAff = nodeAff.SetRequiredDuringSchedulingIgnoredDuringExecution(nodeSelector)
	aff = aff.SetNodeAffinity(nodeAff)

	podAff := clawpod.NewPodAffinity(context.Background())
	podTerms := make([]clawpod.PodAffinityTerm, 1)
	podTerm := clawpod.NewPodAffinityTerm(context.Background())
	podTerm = podTerm.SetTopologyKey("kubernetes.io/hostname")
	ls := clawpod.NewLabelSelector(context.Background())
	matchLabels := make([]clawpod.KeyValue, 1)
	kv := clawpod.NewKeyValue(context.Background())
	kv = kv.SetKey("app")
	kv = kv.SetValue("test")
	matchLabels[0] = kv
	ls.MatchLabelsAppend(context.Background(), matchLabels...)
	podTerm = podTerm.SetLabelSelector(ls)
	podTerms[0] = podTerm
	podAff.RequiredDuringSchedulingIgnoredDuringExecutionAppend(context.Background(), podTerms...)
	aff = aff.SetPodAffinity(podAff)

	return aff
}

func createClawPodStatus() clawpod.PodStatus {
	ps := clawpod.NewPodStatus(context.Background())
	ps = ps.SetPhase(clawpod.PodPhaseRunning)

	conditions := make([]clawpod.PodCondition, 4)
	condTypes := []clawpod.PodConditionType{
		clawpod.PodConditionTypeInitialized,
		clawpod.PodConditionTypeReady,
		clawpod.PodConditionTypeContainersReady,
		clawpod.PodConditionTypePodScheduled,
	}
	for i := 0; i < 4; i++ {
		c := clawpod.NewPodCondition(context.Background())
		c = c.SetType(condTypes[i])
		c = c.SetStatus(clawpod.ConditionStatusTrue)
		t := clawpod.NewTime(context.Background())
		t = t.SetSeconds(1703721600)
		c = c.SetLastTransitionTime(t)
		conditions[i] = c
	}
	ps.ConditionsAppend(context.Background(), conditions...)

	ps = ps.SetMessage("Pod is running")
	ps = ps.SetHostIp("192.168.1.100")
	ps = ps.SetPodIp("10.244.0.5")

	podIPs := make([]clawpod.PodIP, 2)
	pip1 := clawpod.NewPodIP(context.Background())
	pip1 = pip1.SetIp("10.244.0.5")
	podIPs[0] = pip1
	pip2 := clawpod.NewPodIP(context.Background())
	pip2 = pip2.SetIp("fd00::5")
	podIPs[1] = pip2
	ps.PodIpsAppend(context.Background(), podIPs...)

	startTime := clawpod.NewTime(context.Background())
	startTime = startTime.SetSeconds(1703721600)
	ps = ps.SetStartTime(startTime)

	containerStatuses := make([]clawpod.ContainerStatus, listSize)
	for i := 0; i < listSize; i++ {
		cs := clawpod.NewContainerStatus(context.Background())
		cs = cs.SetName(fmt.Sprintf("container-%d", i))
		state := clawpod.NewContainerState(context.Background())
		running := clawpod.NewContainerStateRunning(context.Background())
		t := clawpod.NewTime(context.Background())
		t = t.SetSeconds(1703721600)
		running = running.SetStartedAt(t)
		state = state.SetRunning(running)
		cs = cs.SetState(state)
		cs = cs.SetReady(true)
		cs = cs.SetRestartCount(0)
		cs = cs.SetImage("nginx:latest")
		cs = cs.SetImageId("docker://sha256:abc123")
		cs = cs.SetContainerId(fmt.Sprintf("docker://container-%d", i))
		containerStatuses[i] = cs
	}
	ps.ContainerStatusesAppend(context.Background(), containerStatuses...)

	ps = ps.SetQosClass(clawpod.PodQOSClassBurstable)

	return ps
}

// Helper to create a populated Cap'n Proto Pod
func createCapnpPod() (*capnp.Message, capnpod.Pod, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, capnpod.Pod{}, err
	}

	pod, err := capnpod.NewRootPod(seg)
	if err != nil {
		return nil, capnpod.Pod{}, err
	}

	// TypeMeta
	typeMeta, err := capnpod.NewTypeMeta(seg)
	if err != nil {
		return nil, capnpod.Pod{}, err
	}
	typeMeta.SetKind("Pod")
	typeMeta.SetApiVersion("v1")
	pod.SetTypeMeta(typeMeta)

	// Metadata
	metadata, err := createCapnpObjectMeta(seg)
	if err != nil {
		return nil, capnpod.Pod{}, err
	}
	pod.SetMetadata(metadata)

	// Spec
	spec, err := createCapnpPodSpec(seg)
	if err != nil {
		return nil, capnpod.Pod{}, err
	}
	pod.SetSpec(spec)

	// Status
	status, err := createCapnpPodStatus(seg)
	if err != nil {
		return nil, capnpod.Pod{}, err
	}
	pod.SetStatus(status)

	return msg, pod, nil
}

func createCapnpObjectMeta(seg *capnp.Segment) (capnpod.ObjectMeta, error) {
	om, err := capnpod.NewObjectMeta(seg)
	if err != nil {
		return capnpod.ObjectMeta{}, err
	}

	om.SetName("test-pod")
	om.SetGenerateName("test-pod-")
	om.SetNamespace("default")
	om.SetSelfLink("/api/v1/namespaces/default/pods/test-pod")
	om.SetUid("12345678-1234-1234-1234-123456789012")
	om.SetResourceVersion("12345")
	om.SetGeneration(1)

	ct, _ := capnpod.NewTime(seg)
	ct.SetSeconds(1703721600)
	ct.SetNanos(0)
	om.SetCreationTimestamp(ct)

	labels, _ := capnpod.NewKeyValue_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		kv := labels.At(i)
		kv.SetKey(fmt.Sprintf("label-key-%d", i))
		kv.SetValue(fmt.Sprintf("label-value-%d", i))
	}
	om.SetLabels(labels)

	annotations, _ := capnpod.NewKeyValue_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		kv := annotations.At(i)
		kv.SetKey(fmt.Sprintf("annotation-key-%d", i))
		kv.SetValue(fmt.Sprintf("annotation-value-%d", i))
	}
	om.SetAnnotations(annotations)

	ownerRefs, _ := capnpod.NewOwnerReference_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		or := ownerRefs.At(i)
		or.SetApiVersion("v1")
		or.SetKind("ReplicaSet")
		or.SetName(fmt.Sprintf("owner-%d", i))
		or.SetUid(fmt.Sprintf("uid-%d", i))
	}
	om.SetOwnerReferences(ownerRefs)

	finalizers, _ := capnp.NewTextList(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		finalizers.Set(i, fmt.Sprintf("finalizer-%d", i))
	}
	om.SetFinalizers(finalizers)

	return om, nil
}

func createCapnpPodSpec(seg *capnp.Segment) (capnpod.PodSpec, error) {
	ps, err := capnpod.NewPodSpec(seg)
	if err != nil {
		return capnpod.PodSpec{}, err
	}

	volumes, _ := capnpod.NewVolume_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		v := volumes.At(i)
		v.SetName(fmt.Sprintf("volume-%d", i))
		vs, _ := capnpod.NewVolumeSource(seg)
		ed, _ := capnpod.NewEmptyDirVolumeSource(seg)
		ed.SetMedium(capnpod.StorageMedium_storageMediumMemory)
		vs.SetEmptyDir(ed)
		v.SetVolumeSource(vs)
	}
	ps.SetVolumes(volumes)

	initContainers, _ := capnpod.NewContainer_List(seg, 2)
	for i := 0; i < 2; i++ {
		c := initContainers.At(i)
		populateCapnpContainer(seg, c, fmt.Sprintf("init-container-%d", i))
	}
	ps.SetInitContainers(initContainers)

	containers, _ := capnpod.NewContainer_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		c := containers.At(i)
		populateCapnpContainer(seg, c, fmt.Sprintf("container-%d", i))
	}
	ps.SetContainers(containers)

	ps.SetRestartPolicy(capnpod.RestartPolicy_restartPolicyAlways)
	ps.SetTerminationGracePeriodSeconds(30)
	ps.SetDnsPolicy(capnpod.DNSPolicy_dnsPolicyClusterFirst)

	nodeSelector, _ := capnpod.NewKeyValue_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		kv := nodeSelector.At(i)
		kv.SetKey(fmt.Sprintf("node-key-%d", i))
		kv.SetValue(fmt.Sprintf("node-value-%d", i))
	}
	ps.SetNodeSelector(nodeSelector)

	ps.SetServiceAccountName("default")
	ps.SetNodeName("node-1")
	ps.SetHostNetwork(false)
	ps.SetHostPid(false)
	ps.SetHostIpc(false)

	psc, _ := createCapnpPodSecurityContext(seg)
	ps.SetSecurityContext(psc)

	ps.SetHostname("test-pod")
	ps.SetSubdomain("test-subdomain")
	ps.SetSchedulerName("default-scheduler")

	tolerations, _ := capnpod.NewToleration_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		t := tolerations.At(i)
		t.SetKey(fmt.Sprintf("key-%d", i))
		t.SetOperator(capnpod.TolerationOperator_tolerationOperatorEqual)
		t.SetValue(fmt.Sprintf("value-%d", i))
		t.SetEffect(capnpod.TaintEffect_taintEffectNoSchedule)
	}
	ps.SetTolerations(tolerations)

	hostAliases, _ := capnpod.NewHostAlias_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		ha := hostAliases.At(i)
		ha.SetIp(fmt.Sprintf("10.0.0.%d", i))
		hostnames, _ := capnp.NewTextList(seg, 3)
		for j := 0; j < 3; j++ {
			hostnames.Set(j, fmt.Sprintf("host-%d-%d.example.com", i, j))
		}
		ha.SetHostnames(hostnames)
	}
	ps.SetHostAliases(hostAliases)

	ps.SetPriorityClassName("high-priority")
	ps.SetPriority(1000)

	dnsConfig, _ := capnpod.NewPodDNSConfig(seg)
	nameservers, _ := capnp.NewTextList(seg, 2)
	nameservers.Set(0, "8.8.8.8")
	nameservers.Set(1, "8.8.4.4")
	dnsConfig.SetNameservers(nameservers)
	searches, _ := capnp.NewTextList(seg, 2)
	searches.Set(0, "default.svc.cluster.local")
	searches.Set(1, "svc.cluster.local")
	dnsConfig.SetSearches(searches)
	ps.SetDnsConfig(dnsConfig)

	aff, _ := createCapnpAffinity(seg)
	ps.SetAffinity(aff)

	return ps, nil
}

func populateCapnpContainer(seg *capnp.Segment, c capnpod.Container, name string) {
	c.SetName(name)
	c.SetImage("nginx:latest")

	command, _ := capnp.NewTextList(seg, 3)
	for i := 0; i < 3; i++ {
		command.Set(i, fmt.Sprintf("/bin/cmd%d", i))
	}
	c.SetCommand(command)

	args, _ := capnp.NewTextList(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		args.Set(i, fmt.Sprintf("--arg%d=value%d", i, i))
	}
	c.SetArgs(args)

	c.SetWorkingDir("/app")

	ports, _ := capnpod.NewContainerPort_List(seg, 3)
	for i := 0; i < 3; i++ {
		p := ports.At(i)
		p.SetName(fmt.Sprintf("port-%d", i))
		p.SetContainerPort(int32(8080 + i))
		p.SetProtocol(capnpod.Protocol_protocolTcp)
	}
	c.SetPorts(ports)

	env, _ := capnpod.NewEnvVar_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		e := env.At(i)
		e.SetName(fmt.Sprintf("ENV_%d", i))
		e.SetValue(fmt.Sprintf("value-%d", i))
	}
	c.SetEnv(env)

	resources, _ := capnpod.NewResourceRequirements(seg)
	limits, _ := capnpod.NewKeyValue_List(seg, 2)
	limits.At(0).SetKey("cpu")
	limits.At(0).SetValue("1000m")
	limits.At(1).SetKey("memory")
	limits.At(1).SetValue("1Gi")
	resources.SetLimits(limits)
	requests, _ := capnpod.NewKeyValue_List(seg, 2)
	requests.At(0).SetKey("cpu")
	requests.At(0).SetValue("500m")
	requests.At(1).SetKey("memory")
	requests.At(1).SetValue("512Mi")
	resources.SetRequests(requests)
	c.SetResources(resources)

	volumeMounts, _ := capnpod.NewVolumeMount_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		vm := volumeMounts.At(i)
		vm.SetName(fmt.Sprintf("volume-%d", i))
		vm.SetMountPath(fmt.Sprintf("/mnt/volume-%d", i))
		vm.SetReadOnly(i%2 == 0)
	}
	c.SetVolumeMounts(volumeMounts)

	c.SetTerminationMessagePath("/dev/termination-log")
	c.SetImagePullPolicy(capnpod.PullPolicy_pullPolicyIfNotPresent)

	livenessProbe, _ := capnpod.NewProbe(seg)
	handler, _ := capnpod.NewProbeHandler(seg)
	httpGet, _ := capnpod.NewHTTPGetAction(seg)
	httpGet.SetPath("/healthz")
	port, _ := capnpod.NewIntOrString(seg)
	port.SetIntVal(8080)
	httpGet.SetPort(port)
	handler.SetHttpGet(httpGet)
	livenessProbe.SetHandler(handler)
	livenessProbe.SetInitialDelaySeconds(10)
	livenessProbe.SetPeriodSeconds(30)
	c.SetLivenessProbe(livenessProbe)

	readinessProbe, _ := capnpod.NewProbe(seg)
	handler2, _ := capnpod.NewProbeHandler(seg)
	httpGet2, _ := capnpod.NewHTTPGetAction(seg)
	httpGet2.SetPath("/ready")
	port2, _ := capnpod.NewIntOrString(seg)
	port2.SetIntVal(8080)
	httpGet2.SetPort(port2)
	handler2.SetHttpGet(httpGet2)
	readinessProbe.SetHandler(handler2)
	readinessProbe.SetInitialDelaySeconds(5)
	readinessProbe.SetPeriodSeconds(10)
	c.SetReadinessProbe(readinessProbe)
}

func createCapnpPodSecurityContext(seg *capnp.Segment) (capnpod.PodSecurityContext, error) {
	psc, err := capnpod.NewPodSecurityContext(seg)
	if err != nil {
		return capnpod.PodSecurityContext{}, err
	}

	psc.SetRunAsUser(1000)
	psc.SetRunAsGroup(1000)
	psc.SetRunAsNonRoot(true)

	supplementalGroups, _ := capnp.NewInt64List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		supplementalGroups.Set(i, int64(1000+i))
	}
	psc.SetSupplementalGroups(supplementalGroups)
	psc.SetFsGroup(2000)

	return psc, nil
}

func createCapnpAffinity(seg *capnp.Segment) (capnpod.Affinity, error) {
	aff, err := capnpod.NewAffinity(seg)
	if err != nil {
		return capnpod.Affinity{}, err
	}

	nodeAff, _ := capnpod.NewNodeAffinity(seg)
	nodeSelector, _ := capnpod.NewNodeSelector(seg)
	terms, _ := capnpod.NewNodeSelectorTerm_List(seg, 1)
	term := terms.At(0)
	reqs, _ := capnpod.NewNodeSelectorRequirement_List(seg, 1)
	req := reqs.At(0)
	req.SetKey("kubernetes.io/os")
	req.SetOperator(capnpod.NodeSelectorOperator_nodeSelectorOperatorIn)
	values, _ := capnp.NewTextList(seg, 1)
	values.Set(0, "linux")
	req.SetValues(values)
	term.SetMatchExpressions(reqs)
	nodeSelector.SetNodeSelectorTerms(terms)
	nodeAff.SetRequiredDuringSchedulingIgnoredDuringExecution(nodeSelector)
	aff.SetNodeAffinity(nodeAff)

	podAff, _ := capnpod.NewPodAffinity(seg)
	podTerms, _ := capnpod.NewPodAffinityTerm_List(seg, 1)
	podTerm := podTerms.At(0)
	podTerm.SetTopologyKey("kubernetes.io/hostname")
	ls, _ := capnpod.NewLabelSelector(seg)
	matchLabels, _ := capnpod.NewKeyValue_List(seg, 1)
	kv := matchLabels.At(0)
	kv.SetKey("app")
	kv.SetValue("test")
	ls.SetMatchLabels(matchLabels)
	podTerm.SetLabelSelector(ls)
	podAff.SetRequiredDuringSchedulingIgnoredDuringExecution(podTerms)
	aff.SetPodAffinity(podAff)

	return aff, nil
}

func createCapnpPodStatus(seg *capnp.Segment) (capnpod.PodStatus, error) {
	ps, err := capnpod.NewPodStatus(seg)
	if err != nil {
		return capnpod.PodStatus{}, err
	}

	ps.SetPhase(capnpod.PodPhase_podPhaseRunning)

	conditions, _ := capnpod.NewPodCondition_List(seg, 4)
	condTypes := []capnpod.PodConditionType{
		capnpod.PodConditionType_podConditionTypeInitialized,
		capnpod.PodConditionType_podConditionTypeReady,
		capnpod.PodConditionType_podConditionTypeContainersReady,
		capnpod.PodConditionType_podConditionTypePodScheduled,
	}
	for i := 0; i < 4; i++ {
		c := conditions.At(i)
		c.SetType(condTypes[i])
		c.SetStatus(capnpod.ConditionStatus_conditionStatusTrue)
		t, _ := capnpod.NewTime(seg)
		t.SetSeconds(1703721600)
		c.SetLastTransitionTime(t)
	}
	ps.SetConditions(conditions)

	ps.SetMessage_("Pod is running")
	ps.SetHostIp("192.168.1.100")
	ps.SetPodIp("10.244.0.5")

	podIPs, _ := capnpod.NewPodIP_List(seg, 2)
	podIPs.At(0).SetIp("10.244.0.5")
	podIPs.At(1).SetIp("fd00::5")
	ps.SetPodIps(podIPs)

	startTime, _ := capnpod.NewTime(seg)
	startTime.SetSeconds(1703721600)
	ps.SetStartTime(startTime)

	containerStatuses, _ := capnpod.NewContainerStatus_List(seg, int32(listSize))
	for i := 0; i < listSize; i++ {
		cs := containerStatuses.At(i)
		cs.SetName(fmt.Sprintf("container-%d", i))
		state, _ := capnpod.NewContainerState(seg)
		running, _ := capnpod.NewContainerStateRunning(seg)
		t, _ := capnpod.NewTime(seg)
		t.SetSeconds(1703721600)
		running.SetStartedAt(t)
		state.SetRunning(running)
		cs.SetState(state)
		cs.SetReady(true)
		cs.SetRestartCount(0)
		cs.SetImage("nginx:latest")
		cs.SetImageId("docker://sha256:abc123")
		cs.SetContainerId(fmt.Sprintf("docker://container-%d", i))
	}
	ps.SetContainerStatuses(containerStatuses)

	ps.SetQosClass(capnpod.PodQOSClass_podQosClassBurstable)

	return ps, nil
}

// Benchmarks for Protocol Buffers
func BenchmarkProtoMarshal(b *testing.B) {
	pod := createProtoPod()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(pod)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProtoUnmarshal(b *testing.B) {
	pod := createProtoPod()
	data, err := proto.Marshal(pod)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newPod := &protopod.Pod{}
		err := proto.Unmarshal(data, newPod)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for Cap'n Proto
func BenchmarkCapnpMarshal(b *testing.B) {
	msg, _, err := createCapnpPod()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := capnp.NewEncoder(&buf).Encode(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCapnpPooledMarshal benchmarks Cap'n Proto with arena pooling.
// This creates, marshals, and releases the message in each iteration to show
// the full creation cost with pooling benefits from arena reuse.
func BenchmarkCapnpPooledMarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		msg, _, err := createCapnpPod()
		if err != nil {
			b.Fatal(err)
		}
		var buf bytes.Buffer
		err = capnp.NewEncoder(&buf).Encode(msg)
		if err != nil {
			b.Fatal(err)
		}
		msg.Release()
	}
}

func BenchmarkCapnpUnmarshal(b *testing.B) {
	msg, _, err := createCapnpPod()
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	err = capnp.NewEncoder(&buf).Encode(msg)
	if err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := capnp.Unmarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCapnpPooledUnmarshal benchmarks Cap'n Proto unmarshal with arena pooling.
func BenchmarkCapnpPooledUnmarshal(b *testing.B) {
	msg, _, err := createCapnpPod()
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	err = capnp.NewEncoder(&buf).Encode(msg)
	if err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()
	msg.Release()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newMsg, err := capnp.Unmarshal(data)
		if err != nil {
			b.Fatal(err)
		}
		newMsg.Release()
	}
}

func BenchmarkClawMarshal(b *testing.B) {
	pod := createClawPod()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := pod.Marshal()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClawUnmarshal(b *testing.B) {
	pod := createClawPod()
	data, err := pod.Marshal()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newPod := clawpod.NewPod(context.Background())
		err := newPod.Unmarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClawUnmarshalPooled(b *testing.B) {
	data := [1000][]byte{}
	pod := createClawPod()

	// Because .Marshal() would just give us pod's internals and Unmarshal would use it
	// then Release() would cause it to go back in the pool, we'd end up corrupting the data.
	// So we prepare 1000 copies of the marshaled data to use in the benchmark.
	prepData := func() {
		b.StopTimer()
		for i := 0; i < 1000; i++ {
			data[i], _ = pod.MarshalSafe()
		}
		b.StartTimer()
	}
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newPod := clawpod.NewPod(ctx)
		dataIdx := i % 1000
		if dataIdx == 0 {
			prepData()
		}
		err := newPod.Unmarshal(data[dataIdx])
		if err != nil {
			b.Fatal(err)
		}
		newPod.Release(ctx)
	}
}

// Test to print sizes
func TestPrintSizes(t *testing.T) {
	// Proto
	protoPod := createProtoPod()
	protoData, err := proto.Marshal(protoPod)
	if err != nil {
		t.Fatal(err)
	}

	// Cap'n Proto
	capnpMsg, _, err := createCapnpPod()
	if err != nil {
		t.Fatal(err)
	}
	var capnpBuf bytes.Buffer
	err = capnp.NewEncoder(&capnpBuf).Encode(capnpMsg)
	if err != nil {
		t.Fatal(err)
	}
	capnpData := capnpBuf.Bytes()

	// Claw
	clawPod := createClawPod()
	clawData, err := clawPod.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	// JSON (go-json-experiment)
	jsonPod := createJSONPod()
	jsonData, err := jsonMarshal(jsonPod)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("\n=== Serialized Sizes ===\n")
	fmt.Printf("Protocol Buffers: %d bytes\n", len(protoData))
	fmt.Printf("Cap'n Proto:      %d bytes\n", len(capnpData))
	fmt.Printf("Claw:             %d bytes\n", len(clawData))
	fmt.Printf("JSON:             %d bytes\n", len(jsonData))
	fmt.Printf("\n")
}
