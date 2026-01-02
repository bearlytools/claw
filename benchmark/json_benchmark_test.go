package benchmark

import (
	"fmt"
	"testing"

	"github.com/go-json-experiment/json"
)

// NOTE: JSON size is printed in TestPrintSizes in benchmark_test.go along with other formats

// JSON-compatible Pod structures that mirror the proto/claw structures

type JSONTime struct {
	Seconds int64 `json:"seconds"`
	Nanos   int32 `json:"nanos"`
}

type JSONKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type JSONOwnerReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	UID        string `json:"uid"`
}

type JSONObjectMeta struct {
	Name              string               `json:"name"`
	GenerateName      string               `json:"generateName"`
	Namespace         string               `json:"namespace"`
	SelfLink          string               `json:"selfLink"`
	UID               string               `json:"uid"`
	ResourceVersion   string               `json:"resourceVersion"`
	Generation        int64                `json:"generation"`
	CreationTimestamp JSONTime             `json:"creationTimestamp"`
	Labels            []JSONKeyValue       `json:"labels"`
	Annotations       []JSONKeyValue       `json:"annotations"`
	OwnerReferences   []JSONOwnerReference `json:"ownerReferences"`
	Finalizers        []string             `json:"finalizers"`
}

type JSONTypeMeta struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

type JSONContainerPort struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

type JSONEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type JSONResourceRequirements struct {
	Limits   map[string]string `json:"limits"`
	Requests map[string]string `json:"requests"`
}

type JSONVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

type JSONIntOrString struct {
	IntVal    int32  `json:"intVal,omitempty"`
	StringVal string `json:"stringVal,omitempty"`
}

type JSONHTTPGetAction struct {
	Path string          `json:"path"`
	Port JSONIntOrString `json:"port"`
}

type JSONProbeHandler struct {
	HTTPGet *JSONHTTPGetAction `json:"httpGet,omitempty"`
}

type JSONProbe struct {
	Handler             JSONProbeHandler `json:"handler"`
	InitialDelaySeconds int32            `json:"initialDelaySeconds"`
	PeriodSeconds       int32            `json:"periodSeconds"`
}

type JSONContainer struct {
	Name                   string                   `json:"name"`
	Image                  string                   `json:"image"`
	Command                []string                 `json:"command"`
	Args                   []string                 `json:"args"`
	WorkingDir             string                   `json:"workingDir"`
	Ports                  []JSONContainerPort      `json:"ports"`
	Env                    []JSONEnvVar             `json:"env"`
	Resources              JSONResourceRequirements `json:"resources"`
	VolumeMounts           []JSONVolumeMount        `json:"volumeMounts"`
	TerminationMessagePath string                   `json:"terminationMessagePath"`
	ImagePullPolicy        string                   `json:"imagePullPolicy"`
	LivenessProbe          *JSONProbe               `json:"livenessProbe,omitempty"`
	ReadinessProbe         *JSONProbe               `json:"readinessProbe,omitempty"`
}

type JSONEmptyDirVolumeSource struct {
	Medium string `json:"medium"`
}

type JSONVolumeSource struct {
	EmptyDir *JSONEmptyDirVolumeSource `json:"emptyDir,omitempty"`
}

type JSONVolume struct {
	Name         string           `json:"name"`
	VolumeSource JSONVolumeSource `json:"volumeSource"`
}

type JSONPodSecurityContext struct {
	RunAsUser          int64   `json:"runAsUser"`
	RunAsGroup         int64   `json:"runAsGroup"`
	RunAsNonRoot       bool    `json:"runAsNonRoot"`
	SupplementalGroups []int64 `json:"supplementalGroups"`
	FSGroup            int64   `json:"fsGroup"`
}

type JSONToleration struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
}

type JSONHostAlias struct {
	IP        string   `json:"ip"`
	Hostnames []string `json:"hostnames"`
}

type JSONPodDNSConfig struct {
	Nameservers []string `json:"nameservers"`
	Searches    []string `json:"searches"`
}

type JSONNodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type JSONNodeSelectorTerm struct {
	MatchExpressions []JSONNodeSelectorRequirement `json:"matchExpressions"`
}

type JSONNodeSelector struct {
	NodeSelectorTerms []JSONNodeSelectorTerm `json:"nodeSelectorTerms"`
}

type JSONNodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution *JSONNodeSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

type JSONLabelSelector struct {
	MatchLabels []JSONKeyValue `json:"matchLabels"`
}

type JSONPodAffinityTerm struct {
	TopologyKey   string            `json:"topologyKey"`
	LabelSelector JSONLabelSelector `json:"labelSelector"`
}

type JSONPodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution []JSONPodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution"`
}

type JSONAffinity struct {
	NodeAffinity *JSONNodeAffinity `json:"nodeAffinity,omitempty"`
	PodAffinity  *JSONPodAffinity  `json:"podAffinity,omitempty"`
}

type JSONPodSpec struct {
	Volumes                       []JSONVolume            `json:"volumes"`
	InitContainers                []JSONContainer         `json:"initContainers"`
	Containers                    []JSONContainer         `json:"containers"`
	RestartPolicy                 string                  `json:"restartPolicy"`
	TerminationGracePeriodSeconds int64                   `json:"terminationGracePeriodSeconds"`
	DNSPolicy                     string                  `json:"dnsPolicy"`
	NodeSelector                  []JSONKeyValue          `json:"nodeSelector"`
	ServiceAccountName            string                  `json:"serviceAccountName"`
	NodeName                      string                  `json:"nodeName"`
	HostNetwork                   bool                    `json:"hostNetwork"`
	HostPID                       bool                    `json:"hostPID"`
	HostIPC                       bool                    `json:"hostIPC"`
	SecurityContext               *JSONPodSecurityContext `json:"securityContext,omitempty"`
	Hostname                      string                  `json:"hostname"`
	Subdomain                     string                  `json:"subdomain"`
	SchedulerName                 string                  `json:"schedulerName"`
	Tolerations                   []JSONToleration        `json:"tolerations"`
	HostAliases                   []JSONHostAlias         `json:"hostAliases"`
	PriorityClassName             string                  `json:"priorityClassName"`
	Priority                      int32                   `json:"priority"`
	DNSConfig                     *JSONPodDNSConfig       `json:"dnsConfig,omitempty"`
	Affinity                      *JSONAffinity           `json:"affinity,omitempty"`
}

type JSONContainerStateRunning struct {
	StartedAt JSONTime `json:"startedAt"`
}

type JSONContainerStateTerminated struct {
	ExitCode   int32    `json:"exitCode"`
	FinishedAt JSONTime `json:"finishedAt"`
}

type JSONContainerStateWaiting struct {
	Reason string `json:"reason"`
}

type JSONContainerState struct {
	Running    *JSONContainerStateRunning    `json:"running,omitempty"`
	Terminated *JSONContainerStateTerminated `json:"terminated,omitempty"`
	Waiting    *JSONContainerStateWaiting    `json:"waiting,omitempty"`
}

type JSONContainerStatus struct {
	Name         string             `json:"name"`
	State        JSONContainerState `json:"state"`
	Ready        bool               `json:"ready"`
	RestartCount int32              `json:"restartCount"`
	Image        string             `json:"image"`
	ImageID      string             `json:"imageID"`
	ContainerID  string             `json:"containerID"`
}

type JSONPodCondition struct {
	Type               string   `json:"type"`
	Status             string   `json:"status"`
	LastTransitionTime JSONTime `json:"lastTransitionTime"`
}

type JSONPodIP struct {
	IP string `json:"ip"`
}

type JSONPodStatus struct {
	Phase             string                `json:"phase"`
	Conditions        []JSONPodCondition    `json:"conditions"`
	Message           string                `json:"message"`
	HostIP            string                `json:"hostIP"`
	PodIP             string                `json:"podIP"`
	PodIPs            []JSONPodIP           `json:"podIPs"`
	StartTime         JSONTime              `json:"startTime"`
	ContainerStatuses []JSONContainerStatus `json:"containerStatuses"`
	QOSClass          string                `json:"qosClass"`
}

type JSONPod struct {
	TypeMeta JSONTypeMeta   `json:"typeMeta"`
	Metadata JSONObjectMeta `json:"metadata"`
	Spec     JSONPodSpec    `json:"spec"`
	Status   JSONPodStatus  `json:"status"`
}

func createJSONPod() *JSONPod {
	return &JSONPod{
		TypeMeta: JSONTypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Metadata: createJSONObjectMeta(),
		Spec:     createJSONPodSpec(),
		Status:   createJSONPodStatus(),
	}
}

func createJSONObjectMeta() JSONObjectMeta {
	labels := make([]JSONKeyValue, listSize)
	annotations := make([]JSONKeyValue, listSize)
	for i := 0; i < listSize; i++ {
		labels[i] = JSONKeyValue{
			Key:   fmt.Sprintf("label-key-%d", i),
			Value: fmt.Sprintf("label-value-%d", i),
		}
		annotations[i] = JSONKeyValue{
			Key:   fmt.Sprintf("annotation-key-%d", i),
			Value: fmt.Sprintf("annotation-value-%d", i),
		}
	}

	ownerRefs := make([]JSONOwnerReference, listSize)
	for i := 0; i < listSize; i++ {
		ownerRefs[i] = JSONOwnerReference{
			APIVersion: "v1",
			Kind:       "ReplicaSet",
			Name:       fmt.Sprintf("owner-%d", i),
			UID:        fmt.Sprintf("uid-%d", i),
		}
	}

	finalizers := make([]string, listSize)
	for i := 0; i < listSize; i++ {
		finalizers[i] = fmt.Sprintf("finalizer-%d", i)
	}

	return JSONObjectMeta{
		Name:            "test-pod",
		GenerateName:    "test-pod-",
		Namespace:       "default",
		SelfLink:        "/api/v1/namespaces/default/pods/test-pod",
		UID:             "12345678-1234-1234-1234-123456789012",
		ResourceVersion: "12345",
		Generation:      1,
		CreationTimestamp: JSONTime{
			Seconds: 1703721600,
			Nanos:   0,
		},
		Labels:          labels,
		Annotations:     annotations,
		OwnerReferences: ownerRefs,
		Finalizers:      finalizers,
	}
}

func createJSONPodSpec() JSONPodSpec {
	containers := make([]JSONContainer, listSize)
	for i := 0; i < listSize; i++ {
		containers[i] = createJSONContainer(fmt.Sprintf("container-%d", i))
	}

	initContainers := make([]JSONContainer, 2)
	for i := 0; i < 2; i++ {
		initContainers[i] = createJSONContainer(fmt.Sprintf("init-container-%d", i))
	}

	volumes := make([]JSONVolume, listSize)
	for i := 0; i < listSize; i++ {
		volumes[i] = JSONVolume{
			Name: fmt.Sprintf("volume-%d", i),
			VolumeSource: JSONVolumeSource{
				EmptyDir: &JSONEmptyDirVolumeSource{
					Medium: "Memory",
				},
			},
		}
	}

	tolerations := make([]JSONToleration, listSize)
	for i := 0; i < listSize; i++ {
		tolerations[i] = JSONToleration{
			Key:      fmt.Sprintf("key-%d", i),
			Operator: "Equal",
			Value:    fmt.Sprintf("value-%d", i),
			Effect:   "NoSchedule",
		}
	}

	hostAliases := make([]JSONHostAlias, listSize)
	for i := 0; i < listSize; i++ {
		hostnames := make([]string, 3)
		for j := 0; j < 3; j++ {
			hostnames[j] = fmt.Sprintf("host-%d-%d.example.com", i, j)
		}
		hostAliases[i] = JSONHostAlias{
			IP:        fmt.Sprintf("10.0.0.%d", i),
			Hostnames: hostnames,
		}
	}

	nodeSelector := make([]JSONKeyValue, listSize)
	for i := 0; i < listSize; i++ {
		nodeSelector[i] = JSONKeyValue{
			Key:   fmt.Sprintf("node-key-%d", i),
			Value: fmt.Sprintf("node-value-%d", i),
		}
	}

	return JSONPodSpec{
		Volumes:                       volumes,
		InitContainers:                initContainers,
		Containers:                    containers,
		RestartPolicy:                 "Always",
		TerminationGracePeriodSeconds: 30,
		DNSPolicy:                     "ClusterFirst",
		NodeSelector:                  nodeSelector,
		ServiceAccountName:            "default",
		NodeName:                      "node-1",
		HostNetwork:                   false,
		HostPID:                       false,
		HostIPC:                       false,
		SecurityContext:               createJSONPodSecurityContext(),
		Hostname:                      "test-pod",
		Subdomain:                     "test-subdomain",
		SchedulerName:                 "default-scheduler",
		Tolerations:                   tolerations,
		HostAliases:                   hostAliases,
		PriorityClassName:             "high-priority",
		Priority:                      1000,
		DNSConfig: &JSONPodDNSConfig{
			Nameservers: []string{"8.8.8.8", "8.8.4.4"},
			Searches:    []string{"default.svc.cluster.local", "svc.cluster.local"},
		},
		Affinity: createJSONAffinity(),
	}
}

func createJSONContainer(name string) JSONContainer {
	command := make([]string, 3)
	args := make([]string, listSize)
	for i := 0; i < 3; i++ {
		command[i] = fmt.Sprintf("/bin/cmd%d", i)
	}
	for i := 0; i < listSize; i++ {
		args[i] = fmt.Sprintf("--arg%d=value%d", i, i)
	}

	ports := make([]JSONContainerPort, 3)
	for i := 0; i < 3; i++ {
		ports[i] = JSONContainerPort{
			Name:          fmt.Sprintf("port-%d", i),
			ContainerPort: int32(8080 + i),
			Protocol:      "TCP",
		}
	}

	env := make([]JSONEnvVar, listSize)
	for i := 0; i < listSize; i++ {
		env[i] = JSONEnvVar{
			Name:  fmt.Sprintf("ENV_%d", i),
			Value: fmt.Sprintf("value-%d", i),
		}
	}

	volumeMounts := make([]JSONVolumeMount, listSize)
	for i := 0; i < listSize; i++ {
		volumeMounts[i] = JSONVolumeMount{
			Name:      fmt.Sprintf("volume-%d", i),
			MountPath: fmt.Sprintf("/mnt/volume-%d", i),
			ReadOnly:  i%2 == 0,
		}
	}

	return JSONContainer{
		Name:       name,
		Image:      "nginx:latest",
		Command:    command,
		Args:       args,
		WorkingDir: "/app",
		Ports:      ports,
		Env:        env,
		Resources: JSONResourceRequirements{
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
		ImagePullPolicy:        "IfNotPresent",
		LivenessProbe: &JSONProbe{
			Handler: JSONProbeHandler{
				HTTPGet: &JSONHTTPGetAction{
					Path: "/healthz",
					Port: JSONIntOrString{IntVal: 8080},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
		},
		ReadinessProbe: &JSONProbe{
			Handler: JSONProbeHandler{
				HTTPGet: &JSONHTTPGetAction{
					Path: "/ready",
					Port: JSONIntOrString{IntVal: 8080},
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}
}

func createJSONPodSecurityContext() *JSONPodSecurityContext {
	supplementalGroups := make([]int64, listSize)
	for i := 0; i < listSize; i++ {
		supplementalGroups[i] = int64(1000 + i)
	}

	return &JSONPodSecurityContext{
		RunAsUser:          1000,
		RunAsGroup:         1000,
		RunAsNonRoot:       true,
		SupplementalGroups: supplementalGroups,
		FSGroup:            2000,
	}
}

func createJSONAffinity() *JSONAffinity {
	return &JSONAffinity{
		NodeAffinity: &JSONNodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &JSONNodeSelector{
				NodeSelectorTerms: []JSONNodeSelectorTerm{
					{
						MatchExpressions: []JSONNodeSelectorRequirement{
							{
								Key:      "kubernetes.io/os",
								Operator: "In",
								Values:   []string{"linux"},
							},
						},
					},
				},
			},
		},
		PodAffinity: &JSONPodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []JSONPodAffinityTerm{
				{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: JSONLabelSelector{
						MatchLabels: []JSONKeyValue{{Key: "app", Value: "test"}},
					},
				},
			},
		},
	}
}

func createJSONPodStatus() JSONPodStatus {
	conditions := make([]JSONPodCondition, 4)
	condTypes := []string{"Initialized", "Ready", "ContainersReady", "PodScheduled"}
	for i := 0; i < 4; i++ {
		conditions[i] = JSONPodCondition{
			Type:   condTypes[i],
			Status: "True",
			LastTransitionTime: JSONTime{
				Seconds: 1703721600,
				Nanos:   0,
			},
		}
	}

	containerStatuses := make([]JSONContainerStatus, listSize)
	for i := 0; i < listSize; i++ {
		containerStatuses[i] = JSONContainerStatus{
			Name: fmt.Sprintf("container-%d", i),
			State: JSONContainerState{
				Running: &JSONContainerStateRunning{
					StartedAt: JSONTime{Seconds: 1703721600},
				},
			},
			Ready:        true,
			RestartCount: 0,
			Image:        "nginx:latest",
			ImageID:      "docker://sha256:abc123",
			ContainerID:  fmt.Sprintf("docker://container-%d", i),
		}
	}

	podIPs := []JSONPodIP{
		{IP: "10.244.0.5"},
		{IP: "fd00::5"},
	}

	return JSONPodStatus{
		Phase:      "Running",
		Conditions: conditions,
		Message:    "Pod is running",
		HostIP:     "192.168.1.100",
		PodIP:      "10.244.0.5",
		PodIPs:     podIPs,
		StartTime: JSONTime{
			Seconds: 1703721600,
			Nanos:   0,
		},
		ContainerStatuses: containerStatuses,
		QOSClass:          "Burstable",
	}
}

// Benchmarks for JSON (go-json-experiment)
func BenchmarkJSONMarshal(b *testing.B) {
	pod := createJSONPod()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(pod)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	pod := createJSONPod()
	data, err := json.Marshal(pod)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newPod := &JSONPod{}
		err := json.Unmarshal(data, newPod)
		if err != nil {
			b.Fatal(err)
		}
	}
}
