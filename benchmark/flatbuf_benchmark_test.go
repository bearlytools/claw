package benchmark

import (
	"fmt"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"

	fb "github.com/bearlytools/claw/benchmark/msgs/flatbuf/flatbuf"
)

// Helper to create a populated FlatBuffers Pod
func createFlatbufPod() []byte {
	builder := flatbuffers.NewBuilder(32768)

	// Create TypeMeta
	typeMetaKind := builder.CreateString("Pod")
	typeMetaApiVersion := builder.CreateString("v1")
	fb.TypeMetaStart(builder)
	fb.TypeMetaAddKind(builder, typeMetaKind)
	fb.TypeMetaAddApiVersion(builder, typeMetaApiVersion)
	typeMeta := fb.TypeMetaEnd(builder)

	// Create ObjectMeta
	metadata := createFlatbufObjectMeta(builder)

	// Create PodSpec
	spec := createFlatbufPodSpec(builder)

	// Create PodStatus
	status := createFlatbufPodStatus(builder)

	// Create Pod
	fb.PodStart(builder)
	fb.PodAddTypeMeta(builder, typeMeta)
	fb.PodAddMetadata(builder, metadata)
	fb.PodAddSpec(builder, spec)
	fb.PodAddStatus(builder, status)
	pod := fb.PodEnd(builder)

	fb.FinishPodBuffer(builder, pod)
	return builder.FinishedBytes()
}

func createFlatbufObjectMeta(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	name := builder.CreateString("test-pod")
	generateName := builder.CreateString("test-pod-")
	namespace := builder.CreateString("default")
	selfLink := builder.CreateString("/api/v1/namespaces/default/pods/test-pod")
	uid := builder.CreateString("12345678-1234-1234-1234-123456789012")
	resourceVersion := builder.CreateString("12345")

	// Create creation timestamp
	fb.TimeStart(builder)
	fb.TimeAddSeconds(builder, 1703721600)
	fb.TimeAddNanos(builder, 0)
	creationTimestamp := fb.TimeEnd(builder)

	// Create labels
	labelOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		key := builder.CreateString(fmt.Sprintf("label-key-%d", i))
		value := builder.CreateString(fmt.Sprintf("label-value-%d", i))
		fb.KeyValueStart(builder)
		fb.KeyValueAddKey(builder, key)
		fb.KeyValueAddValue(builder, value)
		labelOffsets[i] = fb.KeyValueEnd(builder)
	}
	fb.ObjectMetaStartLabelsVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(labelOffsets[i])
	}
	labels := builder.EndVector(listSize)

	// Create annotations
	annotationOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		key := builder.CreateString(fmt.Sprintf("annotation-key-%d", i))
		value := builder.CreateString(fmt.Sprintf("annotation-value-%d", i))
		fb.KeyValueStart(builder)
		fb.KeyValueAddKey(builder, key)
		fb.KeyValueAddValue(builder, value)
		annotationOffsets[i] = fb.KeyValueEnd(builder)
	}
	fb.ObjectMetaStartAnnotationsVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(annotationOffsets[i])
	}
	annotations := builder.EndVector(listSize)

	// Create owner references
	ownerRefOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		apiVersion := builder.CreateString("v1")
		kind := builder.CreateString("ReplicaSet")
		ownerName := builder.CreateString(fmt.Sprintf("owner-%d", i))
		ownerUid := builder.CreateString(fmt.Sprintf("uid-%d", i))
		fb.OwnerReferenceStart(builder)
		fb.OwnerReferenceAddApiVersion(builder, apiVersion)
		fb.OwnerReferenceAddKind(builder, kind)
		fb.OwnerReferenceAddName(builder, ownerName)
		fb.OwnerReferenceAddUid(builder, ownerUid)
		ownerRefOffsets[i] = fb.OwnerReferenceEnd(builder)
	}
	fb.ObjectMetaStartOwnerReferencesVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(ownerRefOffsets[i])
	}
	ownerRefs := builder.EndVector(listSize)

	// Create finalizers
	finalizerStrings := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		finalizerStrings[i] = builder.CreateString(fmt.Sprintf("finalizer-%d", i))
	}
	fb.ObjectMetaStartFinalizersVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(finalizerStrings[i])
	}
	finalizers := builder.EndVector(listSize)

	fb.ObjectMetaStart(builder)
	fb.ObjectMetaAddName(builder, name)
	fb.ObjectMetaAddGenerateName(builder, generateName)
	fb.ObjectMetaAddNamespace_(builder, namespace)
	fb.ObjectMetaAddSelfLink(builder, selfLink)
	fb.ObjectMetaAddUid(builder, uid)
	fb.ObjectMetaAddResourceVersion(builder, resourceVersion)
	fb.ObjectMetaAddGeneration(builder, 1)
	fb.ObjectMetaAddCreationTimestamp(builder, creationTimestamp)
	fb.ObjectMetaAddLabels(builder, labels)
	fb.ObjectMetaAddAnnotations(builder, annotations)
	fb.ObjectMetaAddOwnerReferences(builder, ownerRefs)
	fb.ObjectMetaAddFinalizers(builder, finalizers)
	return fb.ObjectMetaEnd(builder)
}

func createFlatbufPodSpec(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// Create volumes
	volumeOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		volumeName := builder.CreateString(fmt.Sprintf("volume-%d", i))
		fb.EmptyDirVolumeSourceStart(builder)
		fb.EmptyDirVolumeSourceAddMedium(builder, fb.StorageMediumMemory)
		emptyDir := fb.EmptyDirVolumeSourceEnd(builder)
		fb.VolumeSourceStart(builder)
		fb.VolumeSourceAddEmptyDir(builder, emptyDir)
		volumeSource := fb.VolumeSourceEnd(builder)
		fb.VolumeStart(builder)
		fb.VolumeAddName(builder, volumeName)
		fb.VolumeAddVolumeSource(builder, volumeSource)
		volumeOffsets[i] = fb.VolumeEnd(builder)
	}
	fb.PodSpecStartVolumesVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(volumeOffsets[i])
	}
	volumes := builder.EndVector(listSize)

	// Create init containers
	initContainerOffsets := make([]flatbuffers.UOffsetT, 2)
	for i := 1; i >= 0; i-- {
		initContainerOffsets[i] = createFlatbufContainer(builder, fmt.Sprintf("init-container-%d", i))
	}
	fb.PodSpecStartInitContainersVector(builder, 2)
	for i := 1; i >= 0; i-- {
		builder.PrependUOffsetT(initContainerOffsets[i])
	}
	initContainers := builder.EndVector(2)

	// Create containers
	containerOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		containerOffsets[i] = createFlatbufContainer(builder, fmt.Sprintf("container-%d", i))
	}
	fb.PodSpecStartContainersVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(containerOffsets[i])
	}
	containers := builder.EndVector(listSize)

	// Create node selector
	nodeSelectorOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		key := builder.CreateString(fmt.Sprintf("node-key-%d", i))
		value := builder.CreateString(fmt.Sprintf("node-value-%d", i))
		fb.KeyValueStart(builder)
		fb.KeyValueAddKey(builder, key)
		fb.KeyValueAddValue(builder, value)
		nodeSelectorOffsets[i] = fb.KeyValueEnd(builder)
	}
	fb.PodSpecStartNodeSelectorVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(nodeSelectorOffsets[i])
	}
	nodeSelector := builder.EndVector(listSize)

	// Create tolerations
	tolerationOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		key := builder.CreateString(fmt.Sprintf("key-%d", i))
		value := builder.CreateString(fmt.Sprintf("value-%d", i))
		fb.TolerationStart(builder)
		fb.TolerationAddKey(builder, key)
		fb.TolerationAddOperator(builder, fb.TolerationOperatorEqual)
		fb.TolerationAddValue(builder, value)
		fb.TolerationAddEffect(builder, fb.TaintEffectNoSchedule)
		tolerationOffsets[i] = fb.TolerationEnd(builder)
	}
	fb.PodSpecStartTolerationsVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(tolerationOffsets[i])
	}
	tolerations := builder.EndVector(listSize)

	// Create host aliases
	hostAliasOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		ip := builder.CreateString(fmt.Sprintf("10.0.0.%d", i))
		hostnameStrs := make([]flatbuffers.UOffsetT, 3)
		for j := 2; j >= 0; j-- {
			hostnameStrs[j] = builder.CreateString(fmt.Sprintf("host-%d-%d.example.com", i, j))
		}
		fb.HostAliasStartHostnamesVector(builder, 3)
		for j := 2; j >= 0; j-- {
			builder.PrependUOffsetT(hostnameStrs[j])
		}
		hostnames := builder.EndVector(3)
		fb.HostAliasStart(builder)
		fb.HostAliasAddIp(builder, ip)
		fb.HostAliasAddHostnames(builder, hostnames)
		hostAliasOffsets[i] = fb.HostAliasEnd(builder)
	}
	fb.PodSpecStartHostAliasesVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(hostAliasOffsets[i])
	}
	hostAliases := builder.EndVector(listSize)

	// Create security context
	fb.PodSecurityContextStartSupplementalGroupsVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependInt64(int64(1000 + i))
	}
	supplementalGroups := builder.EndVector(listSize)
	fb.PodSecurityContextStart(builder)
	fb.PodSecurityContextAddRunAsUser(builder, 1000)
	fb.PodSecurityContextAddRunAsGroup(builder, 1000)
	fb.PodSecurityContextAddRunAsNonRoot(builder, true)
	fb.PodSecurityContextAddSupplementalGroups(builder, supplementalGroups)
	fb.PodSecurityContextAddFsGroup(builder, 2000)
	securityContext := fb.PodSecurityContextEnd(builder)

	// Create DNS config
	ns1 := builder.CreateString("8.8.8.8")
	ns2 := builder.CreateString("8.8.4.4")
	fb.PodDNSConfigStartNameserversVector(builder, 2)
	builder.PrependUOffsetT(ns2)
	builder.PrependUOffsetT(ns1)
	nameservers := builder.EndVector(2)

	s1 := builder.CreateString("default.svc.cluster.local")
	s2 := builder.CreateString("svc.cluster.local")
	fb.PodDNSConfigStartSearchesVector(builder, 2)
	builder.PrependUOffsetT(s2)
	builder.PrependUOffsetT(s1)
	searches := builder.EndVector(2)

	fb.PodDNSConfigStart(builder)
	fb.PodDNSConfigAddNameservers(builder, nameservers)
	fb.PodDNSConfigAddSearches(builder, searches)
	dnsConfig := fb.PodDNSConfigEnd(builder)

	// Create affinity
	affinity := createFlatbufAffinity(builder)

	// Create strings for PodSpec
	serviceAccountName := builder.CreateString("default")
	nodeName := builder.CreateString("node-1")
	hostname := builder.CreateString("test-pod")
	subdomain := builder.CreateString("test-subdomain")
	schedulerName := builder.CreateString("default-scheduler")
	priorityClassName := builder.CreateString("high-priority")

	fb.PodSpecStart(builder)
	fb.PodSpecAddVolumes(builder, volumes)
	fb.PodSpecAddInitContainers(builder, initContainers)
	fb.PodSpecAddContainers(builder, containers)
	fb.PodSpecAddRestartPolicy(builder, fb.RestartPolicyAlways)
	fb.PodSpecAddTerminationGracePeriodSeconds(builder, 30)
	fb.PodSpecAddDnsPolicy(builder, fb.DNSPolicyClusterFirst)
	fb.PodSpecAddNodeSelector(builder, nodeSelector)
	fb.PodSpecAddServiceAccountName(builder, serviceAccountName)
	fb.PodSpecAddNodeName(builder, nodeName)
	fb.PodSpecAddHostNetwork(builder, false)
	fb.PodSpecAddHostPid(builder, false)
	fb.PodSpecAddHostIpc(builder, false)
	fb.PodSpecAddSecurityContext(builder, securityContext)
	fb.PodSpecAddHostname(builder, hostname)
	fb.PodSpecAddSubdomain(builder, subdomain)
	fb.PodSpecAddSchedulerName(builder, schedulerName)
	fb.PodSpecAddTolerations(builder, tolerations)
	fb.PodSpecAddHostAliases(builder, hostAliases)
	fb.PodSpecAddPriorityClassName(builder, priorityClassName)
	fb.PodSpecAddPriority(builder, 1000)
	fb.PodSpecAddDnsConfig(builder, dnsConfig)
	fb.PodSpecAddAffinity(builder, affinity)
	return fb.PodSpecEnd(builder)
}

func createFlatbufContainer(builder *flatbuffers.Builder, name string) flatbuffers.UOffsetT {
	containerName := builder.CreateString(name)
	image := builder.CreateString("nginx:latest")
	workingDir := builder.CreateString("/app")
	terminationMessagePath := builder.CreateString("/dev/termination-log")

	// Create command
	cmdStrs := make([]flatbuffers.UOffsetT, 3)
	for i := 2; i >= 0; i-- {
		cmdStrs[i] = builder.CreateString(fmt.Sprintf("/bin/cmd%d", i))
	}
	fb.ContainerStartCommandVector(builder, 3)
	for i := 2; i >= 0; i-- {
		builder.PrependUOffsetT(cmdStrs[i])
	}
	command := builder.EndVector(3)

	// Create args
	argStrs := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		argStrs[i] = builder.CreateString(fmt.Sprintf("--arg%d=value%d", i, i))
	}
	fb.ContainerStartArgsVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(argStrs[i])
	}
	args := builder.EndVector(listSize)

	// Create ports
	portOffsets := make([]flatbuffers.UOffsetT, 3)
	for i := 2; i >= 0; i-- {
		portName := builder.CreateString(fmt.Sprintf("port-%d", i))
		fb.ContainerPortStart(builder)
		fb.ContainerPortAddName(builder, portName)
		fb.ContainerPortAddContainerPort(builder, int32(8080+i))
		fb.ContainerPortAddProtocol(builder, fb.ProtocolTCP)
		portOffsets[i] = fb.ContainerPortEnd(builder)
	}
	fb.ContainerStartPortsVector(builder, 3)
	for i := 2; i >= 0; i-- {
		builder.PrependUOffsetT(portOffsets[i])
	}
	ports := builder.EndVector(3)

	// Create env
	envOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		envName := builder.CreateString(fmt.Sprintf("ENV_%d", i))
		envValue := builder.CreateString(fmt.Sprintf("value-%d", i))
		fb.EnvVarStart(builder)
		fb.EnvVarAddName(builder, envName)
		fb.EnvVarAddValue(builder, envValue)
		envOffsets[i] = fb.EnvVarEnd(builder)
	}
	fb.ContainerStartEnvVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(envOffsets[i])
	}
	env := builder.EndVector(listSize)

	// Create resources
	cpuKey := builder.CreateString("cpu")
	cpuLimitVal := builder.CreateString("1000m")
	cpuReqVal := builder.CreateString("500m")
	memKey := builder.CreateString("memory")
	memLimitVal := builder.CreateString("1Gi")
	memReqVal := builder.CreateString("512Mi")

	fb.KeyValueStart(builder)
	fb.KeyValueAddKey(builder, cpuKey)
	fb.KeyValueAddValue(builder, cpuLimitVal)
	cpuLimit := fb.KeyValueEnd(builder)

	fb.KeyValueStart(builder)
	fb.KeyValueAddKey(builder, memKey)
	fb.KeyValueAddValue(builder, memLimitVal)
	memLimit := fb.KeyValueEnd(builder)

	fb.ResourceRequirementsStartLimitsVector(builder, 2)
	builder.PrependUOffsetT(memLimit)
	builder.PrependUOffsetT(cpuLimit)
	limits := builder.EndVector(2)

	cpuKey2 := builder.CreateString("cpu")
	memKey2 := builder.CreateString("memory")
	fb.KeyValueStart(builder)
	fb.KeyValueAddKey(builder, cpuKey2)
	fb.KeyValueAddValue(builder, cpuReqVal)
	cpuReq := fb.KeyValueEnd(builder)

	fb.KeyValueStart(builder)
	fb.KeyValueAddKey(builder, memKey2)
	fb.KeyValueAddValue(builder, memReqVal)
	memReq := fb.KeyValueEnd(builder)

	fb.ResourceRequirementsStartRequestsVector(builder, 2)
	builder.PrependUOffsetT(memReq)
	builder.PrependUOffsetT(cpuReq)
	requests := builder.EndVector(2)

	fb.ResourceRequirementsStart(builder)
	fb.ResourceRequirementsAddLimits(builder, limits)
	fb.ResourceRequirementsAddRequests(builder, requests)
	resources := fb.ResourceRequirementsEnd(builder)

	// Create volume mounts
	vmOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		vmName := builder.CreateString(fmt.Sprintf("volume-%d", i))
		mountPath := builder.CreateString(fmt.Sprintf("/mnt/volume-%d", i))
		fb.VolumeMountStart(builder)
		fb.VolumeMountAddName(builder, vmName)
		fb.VolumeMountAddMountPath(builder, mountPath)
		fb.VolumeMountAddReadOnly(builder, i%2 == 0)
		vmOffsets[i] = fb.VolumeMountEnd(builder)
	}
	fb.ContainerStartVolumeMountsVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(vmOffsets[i])
	}
	volumeMounts := builder.EndVector(listSize)

	// Create liveness probe
	livenessPath := builder.CreateString("/healthz")
	fb.IntOrStringStart(builder)
	fb.IntOrStringAddIntVal(builder, 8080)
	livenessPort := fb.IntOrStringEnd(builder)
	fb.HTTPGetActionStart(builder)
	fb.HTTPGetActionAddPath(builder, livenessPath)
	fb.HTTPGetActionAddPort(builder, livenessPort)
	livenessHttpGet := fb.HTTPGetActionEnd(builder)
	fb.ProbeHandlerStart(builder)
	fb.ProbeHandlerAddHttpGet(builder, livenessHttpGet)
	livenessHandler := fb.ProbeHandlerEnd(builder)
	fb.ProbeStart(builder)
	fb.ProbeAddHandler(builder, livenessHandler)
	fb.ProbeAddInitialDelaySeconds(builder, 10)
	fb.ProbeAddPeriodSeconds(builder, 30)
	livenessProbe := fb.ProbeEnd(builder)

	// Create readiness probe
	readinessPath := builder.CreateString("/ready")
	fb.IntOrStringStart(builder)
	fb.IntOrStringAddIntVal(builder, 8080)
	readinessPort := fb.IntOrStringEnd(builder)
	fb.HTTPGetActionStart(builder)
	fb.HTTPGetActionAddPath(builder, readinessPath)
	fb.HTTPGetActionAddPort(builder, readinessPort)
	readinessHttpGet := fb.HTTPGetActionEnd(builder)
	fb.ProbeHandlerStart(builder)
	fb.ProbeHandlerAddHttpGet(builder, readinessHttpGet)
	readinessHandler := fb.ProbeHandlerEnd(builder)
	fb.ProbeStart(builder)
	fb.ProbeAddHandler(builder, readinessHandler)
	fb.ProbeAddInitialDelaySeconds(builder, 5)
	fb.ProbeAddPeriodSeconds(builder, 10)
	readinessProbe := fb.ProbeEnd(builder)

	fb.ContainerStart(builder)
	fb.ContainerAddName(builder, containerName)
	fb.ContainerAddImage(builder, image)
	fb.ContainerAddCommand(builder, command)
	fb.ContainerAddArgs(builder, args)
	fb.ContainerAddWorkingDir(builder, workingDir)
	fb.ContainerAddPorts(builder, ports)
	fb.ContainerAddEnv(builder, env)
	fb.ContainerAddResources(builder, resources)
	fb.ContainerAddVolumeMounts(builder, volumeMounts)
	fb.ContainerAddTerminationMessagePath(builder, terminationMessagePath)
	fb.ContainerAddImagePullPolicy(builder, fb.PullPolicyIfNotPresent)
	fb.ContainerAddLivenessProbe(builder, livenessProbe)
	fb.ContainerAddReadinessProbe(builder, readinessProbe)
	return fb.ContainerEnd(builder)
}

func createFlatbufAffinity(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	// Node affinity
	osKey := builder.CreateString("kubernetes.io/os")
	linuxVal := builder.CreateString("linux")
	fb.NodeSelectorRequirementStartValuesVector(builder, 1)
	builder.PrependUOffsetT(linuxVal)
	values := builder.EndVector(1)
	fb.NodeSelectorRequirementStart(builder)
	fb.NodeSelectorRequirementAddKey(builder, osKey)
	fb.NodeSelectorRequirementAddOperator(builder, fb.NodeSelectorOperatorIn)
	fb.NodeSelectorRequirementAddValues(builder, values)
	req := fb.NodeSelectorRequirementEnd(builder)

	fb.NodeSelectorTermStartMatchExpressionsVector(builder, 1)
	builder.PrependUOffsetT(req)
	matchExpressions := builder.EndVector(1)
	fb.NodeSelectorTermStart(builder)
	fb.NodeSelectorTermAddMatchExpressions(builder, matchExpressions)
	term := fb.NodeSelectorTermEnd(builder)

	fb.NodeSelectorStartNodeSelectorTermsVector(builder, 1)
	builder.PrependUOffsetT(term)
	terms := builder.EndVector(1)
	fb.NodeSelectorStart(builder)
	fb.NodeSelectorAddNodeSelectorTerms(builder, terms)
	nodeSelector := fb.NodeSelectorEnd(builder)

	fb.NodeAffinityStart(builder)
	fb.NodeAffinityAddRequiredDuringSchedulingIgnoredDuringExecution(builder, nodeSelector)
	nodeAffinity := fb.NodeAffinityEnd(builder)

	// Pod affinity
	topologyKey := builder.CreateString("kubernetes.io/hostname")
	appKey := builder.CreateString("app")
	testVal := builder.CreateString("test")
	fb.KeyValueStart(builder)
	fb.KeyValueAddKey(builder, appKey)
	fb.KeyValueAddValue(builder, testVal)
	kv := fb.KeyValueEnd(builder)
	fb.LabelSelectorStartMatchLabelsVector(builder, 1)
	builder.PrependUOffsetT(kv)
	matchLabels := builder.EndVector(1)
	fb.LabelSelectorStart(builder)
	fb.LabelSelectorAddMatchLabels(builder, matchLabels)
	labelSelector := fb.LabelSelectorEnd(builder)

	fb.PodAffinityTermStart(builder)
	fb.PodAffinityTermAddTopologyKey(builder, topologyKey)
	fb.PodAffinityTermAddLabelSelector(builder, labelSelector)
	podTerm := fb.PodAffinityTermEnd(builder)

	fb.PodAffinityStartRequiredDuringSchedulingIgnoredDuringExecutionVector(builder, 1)
	builder.PrependUOffsetT(podTerm)
	podTerms := builder.EndVector(1)
	fb.PodAffinityStart(builder)
	fb.PodAffinityAddRequiredDuringSchedulingIgnoredDuringExecution(builder, podTerms)
	podAffinity := fb.PodAffinityEnd(builder)

	fb.AffinityStart(builder)
	fb.AffinityAddNodeAffinity(builder, nodeAffinity)
	fb.AffinityAddPodAffinity(builder, podAffinity)
	return fb.AffinityEnd(builder)
}

func createFlatbufPodStatus(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	message := builder.CreateString("Pod is running")
	hostIP := builder.CreateString("192.168.1.100")
	podIP := builder.CreateString("10.244.0.5")

	// Create conditions
	condTypes := []fb.PodConditionType{
		fb.PodConditionTypeInitialized,
		fb.PodConditionTypeReady,
		fb.PodConditionTypeContainersReady,
		fb.PodConditionTypePodScheduled,
	}
	condOffsets := make([]flatbuffers.UOffsetT, 4)
	for i := 3; i >= 0; i-- {
		fb.TimeStart(builder)
		fb.TimeAddSeconds(builder, 1703721600)
		lastTransitionTime := fb.TimeEnd(builder)
		fb.PodConditionStart(builder)
		fb.PodConditionAddType(builder, condTypes[i])
		fb.PodConditionAddStatus(builder, fb.ConditionStatusTrue)
		fb.PodConditionAddLastTransitionTime(builder, lastTransitionTime)
		condOffsets[i] = fb.PodConditionEnd(builder)
	}
	fb.PodStatusStartConditionsVector(builder, 4)
	for i := 3; i >= 0; i-- {
		builder.PrependUOffsetT(condOffsets[i])
	}
	conditions := builder.EndVector(4)

	// Create pod IPs
	ip1 := builder.CreateString("10.244.0.5")
	ip2 := builder.CreateString("fd00::5")
	fb.PodIPStart(builder)
	fb.PodIPAddIp(builder, ip1)
	podIP1 := fb.PodIPEnd(builder)
	fb.PodIPStart(builder)
	fb.PodIPAddIp(builder, ip2)
	podIP2 := fb.PodIPEnd(builder)
	fb.PodStatusStartPodIpsVector(builder, 2)
	builder.PrependUOffsetT(podIP2)
	builder.PrependUOffsetT(podIP1)
	podIPs := builder.EndVector(2)

	// Create start time
	fb.TimeStart(builder)
	fb.TimeAddSeconds(builder, 1703721600)
	startTime := fb.TimeEnd(builder)

	// Create container statuses
	csOffsets := make([]flatbuffers.UOffsetT, listSize)
	for i := listSize - 1; i >= 0; i-- {
		csName := builder.CreateString(fmt.Sprintf("container-%d", i))
		csImage := builder.CreateString("nginx:latest")
		csImageID := builder.CreateString("docker://sha256:abc123")
		csContainerID := builder.CreateString(fmt.Sprintf("docker://container-%d", i))

		fb.TimeStart(builder)
		fb.TimeAddSeconds(builder, 1703721600)
		startedAt := fb.TimeEnd(builder)
		fb.ContainerStateRunningStart(builder)
		fb.ContainerStateRunningAddStartedAt(builder, startedAt)
		running := fb.ContainerStateRunningEnd(builder)
		fb.ContainerStateStart(builder)
		fb.ContainerStateAddRunning(builder, running)
		state := fb.ContainerStateEnd(builder)

		fb.ContainerStatusStart(builder)
		fb.ContainerStatusAddName(builder, csName)
		fb.ContainerStatusAddState(builder, state)
		fb.ContainerStatusAddReady(builder, true)
		fb.ContainerStatusAddRestartCount(builder, 0)
		fb.ContainerStatusAddImage(builder, csImage)
		fb.ContainerStatusAddImageId(builder, csImageID)
		fb.ContainerStatusAddContainerId(builder, csContainerID)
		csOffsets[i] = fb.ContainerStatusEnd(builder)
	}
	fb.PodStatusStartContainerStatusesVector(builder, listSize)
	for i := listSize - 1; i >= 0; i-- {
		builder.PrependUOffsetT(csOffsets[i])
	}
	containerStatuses := builder.EndVector(listSize)

	fb.PodStatusStart(builder)
	fb.PodStatusAddPhase(builder, fb.PodPhaseRunning)
	fb.PodStatusAddConditions(builder, conditions)
	fb.PodStatusAddMessage(builder, message)
	fb.PodStatusAddHostIp(builder, hostIP)
	fb.PodStatusAddPodIp(builder, podIP)
	fb.PodStatusAddPodIps(builder, podIPs)
	fb.PodStatusAddStartTime(builder, startTime)
	fb.PodStatusAddContainerStatuses(builder, containerStatuses)
	fb.PodStatusAddQosClass(builder, fb.PodQOSClassBurstable)
	return fb.PodStatusEnd(builder)
}

// Benchmarks for FlatBuffers
func BenchmarkFlatbufMarshal(b *testing.B) {
	// Pre-warm to get a stable builder
	_ = createFlatbufPod()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = createFlatbufPod()
	}
}

func BenchmarkFlatbufUnmarshal(b *testing.B) {
	data := createFlatbufPod()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fb.GetRootAsPod(data, 0)
	}
}

// BenchmarkFlatbufPooledMarshal benchmarks FlatBuffers with builder reuse.
func BenchmarkFlatbufPooledMarshal(b *testing.B) {
	builder := flatbuffers.NewBuilder(32768)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder.Reset()
		createFlatbufPodWithBuilder(builder)
	}
}

// createFlatbufPodWithBuilder creates a Pod using an existing builder for reuse.
func createFlatbufPodWithBuilder(builder *flatbuffers.Builder) []byte {
	// Create TypeMeta
	typeMetaKind := builder.CreateString("Pod")
	typeMetaApiVersion := builder.CreateString("v1")
	fb.TypeMetaStart(builder)
	fb.TypeMetaAddKind(builder, typeMetaKind)
	fb.TypeMetaAddApiVersion(builder, typeMetaApiVersion)
	typeMeta := fb.TypeMetaEnd(builder)

	// Create ObjectMeta
	metadata := createFlatbufObjectMeta(builder)

	// Create PodSpec
	spec := createFlatbufPodSpec(builder)

	// Create PodStatus
	status := createFlatbufPodStatus(builder)

	// Create Pod
	fb.PodStart(builder)
	fb.PodAddTypeMeta(builder, typeMeta)
	fb.PodAddMetadata(builder, metadata)
	fb.PodAddSpec(builder, spec)
	fb.PodAddStatus(builder, status)
	pod := fb.PodEnd(builder)

	fb.FinishPodBuffer(builder, pod)
	return builder.FinishedBytes()
}

func init() {
	// Print FlatBuffers size in TestPrintSizes
	data := createFlatbufPod()
	fmt.Printf("FlatBuffers:      %d bytes\n", len(data))
}
