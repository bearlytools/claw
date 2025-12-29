@0xb8e3a2f4c9d1e5a7;

using Go = import "/go.capnp";
$Go.package("capn");
$Go.import("github.com/bearlytools/claw/benchmark/msgs/capn");

# Time represents a timestamp.
struct Time {
  seconds @0 :Int64;
  nanos @1 :Int32;
}

# TypeMeta describes an individual object in an API response or request.
struct TypeMeta {
  kind @0 :Text;
  apiVersion @1 :Text;
}

# ObjectMeta is metadata that all persisted resources must have.
struct ObjectMeta {
  name @0 :Text;
  generateName @1 :Text;
  namespace @2 :Text;
  selfLink @3 :Text;
  uid @4 :Text;
  resourceVersion @5 :Text;
  generation @6 :Int64;
  creationTimestamp @7 :Time;
  deletionTimestamp @8 :Time;
  deletionGracePeriodSeconds @9 :Int64;
  labels @10 :List(KeyValue);
  annotations @11 :List(KeyValue);
  ownerReferences @12 :List(OwnerReference);
  finalizers @13 :List(Text);
  managedFields @14 :List(ManagedFieldsEntry);
}

# KeyValue represents a key-value pair.
struct KeyValue {
  key @0 :Text;
  value @1 :Text;
}

# OwnerReference contains enough information to let you identify an owning object.
struct OwnerReference {
  apiVersion @0 :Text;
  kind @1 :Text;
  name @2 :Text;
  uid @3 :Text;
  controller @4 :Bool;
  blockOwnerDeletion @5 :Bool;
}

# ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resource.
struct ManagedFieldsEntry {
  manager @0 :Text;
  operation @1 :Text;
  apiVersion @2 :Text;
  time @3 :Time;
  fieldsType @4 :Text;
  fieldsV1 @5 :Text;
  subresource @6 :Text;
}

# Pod is a collection of containers that can run on a host.
struct Pod {
  typeMeta @0 :TypeMeta;
  metadata @1 :ObjectMeta;
  spec @2 :PodSpec;
  status @3 :PodStatus;
}

# PodSpec is a description of a pod.
struct PodSpec {
  volumes @0 :List(Volume);
  initContainers @1 :List(Container);
  containers @2 :List(Container);
  ephemeralContainers @3 :List(EphemeralContainer);
  restartPolicy @4 :RestartPolicy;
  terminationGracePeriodSeconds @5 :Int64;
  activeDeadlineSeconds @6 :Int64;
  dnsPolicy @7 :DNSPolicy;
  nodeSelector @8 :List(KeyValue);
  serviceAccountName @9 :Text;
  automountServiceAccountToken @10 :Bool;
  nodeName @11 :Text;
  hostNetwork @12 :Bool;
  hostPid @13 :Bool;
  hostIpc @14 :Bool;
  shareProcessNamespace @15 :Bool;
  securityContext @16 :PodSecurityContext;
  imagePullSecrets @17 :List(LocalObjectReference);
  hostname @18 :Text;
  subdomain @19 :Text;
  affinity @20 :Affinity;
  schedulerName @21 :Text;
  tolerations @22 :List(Toleration);
  hostAliases @23 :List(HostAlias);
  priorityClassName @24 :Text;
  priority @25 :Int32;
  dnsConfig @26 :PodDNSConfig;
  readinessGates @27 :List(PodReadinessGate);
  runtimeClassName @28 :Text;
  enableServiceLinks @29 :Bool;
  preemptionPolicy @30 :PreemptionPolicy;
  overhead @31 :List(KeyValue);
  topologySpreadConstraints @32 :List(TopologySpreadConstraint);
  setHostnameAsFqdn @33 :Bool;
  os @34 :PodOS;
  hostUsers @35 :Bool;
  schedulingGates @36 :List(PodSchedulingGate);
  resourceClaims @37 :List(PodResourceClaim);
  resources @38 :ResourceRequirements;
}

# PodStatus represents information about the status of a pod.
struct PodStatus {
  phase @0 :PodPhase;
  conditions @1 :List(PodCondition);
  message @2 :Text;
  reason @3 :Text;
  nominatedNodeName @4 :Text;
  hostIp @5 :Text;
  hostIps @6 :List(HostIP);
  podIp @7 :Text;
  podIps @8 :List(PodIP);
  startTime @9 :Time;
  initContainerStatuses @10 :List(ContainerStatus);
  containerStatuses @11 :List(ContainerStatus);
  qosClass @12 :PodQOSClass;
  ephemeralContainerStatuses @13 :List(ContainerStatus);
  resize @14 :PodResizeStatus;
  resourceClaimStatuses @15 :List(PodResourceClaimStatus);
  observedGeneration @16 :Int64;
}

# Container represents a single container that is expected to be run on the host.
struct Container {
  name @0 :Text;
  image @1 :Text;
  command @2 :List(Text);
  args @3 :List(Text);
  workingDir @4 :Text;
  ports @5 :List(ContainerPort);
  envFrom @6 :List(EnvFromSource);
  env @7 :List(EnvVar);
  resources @8 :ResourceRequirements;
  resizePolicy @9 :List(ContainerResizePolicy);
  restartPolicy @10 :ContainerRestartPolicy;
  volumeMounts @11 :List(VolumeMount);
  volumeDevices @12 :List(VolumeDevice);
  livenessProbe @13 :Probe;
  readinessProbe @14 :Probe;
  startupProbe @15 :Probe;
  lifecycle @16 :Lifecycle;
  terminationMessagePath @17 :Text;
  terminationMessagePolicy @18 :TerminationMessagePolicy;
  imagePullPolicy @19 :PullPolicy;
  securityContext @20 :SecurityContext;
  stdin @21 :Bool;
  stdinOnce @22 :Bool;
  tty @23 :Bool;
}

# EphemeralContainer is a temporary container that may be added to an existing pod.
struct EphemeralContainer {
  name @0 :Text;
  image @1 :Text;
  command @2 :List(Text);
  args @3 :List(Text);
  workingDir @4 :Text;
  ports @5 :List(ContainerPort);
  envFrom @6 :List(EnvFromSource);
  env @7 :List(EnvVar);
  resources @8 :ResourceRequirements;
  volumeMounts @9 :List(VolumeMount);
  volumeDevices @10 :List(VolumeDevice);
  livenessProbe @11 :Probe;
  readinessProbe @12 :Probe;
  startupProbe @13 :Probe;
  lifecycle @14 :Lifecycle;
  terminationMessagePath @15 :Text;
  terminationMessagePolicy @16 :TerminationMessagePolicy;
  imagePullPolicy @17 :PullPolicy;
  securityContext @18 :SecurityContext;
  stdin @19 :Bool;
  stdinOnce @20 :Bool;
  tty @21 :Bool;
  targetContainerName @22 :Text;
}

# ContainerPort represents a network port in a single container.
struct ContainerPort {
  name @0 :Text;
  hostPort @1 :Int32;
  containerPort @2 :Int32;
  protocol @3 :Protocol;
  hostIp @4 :Text;
}

# ContainerStatus contains the status of a container.
struct ContainerStatus {
  name @0 :Text;
  state @1 :ContainerState;
  lastTerminationState @2 :ContainerState;
  ready @3 :Bool;
  restartCount @4 :Int32;
  image @5 :Text;
  imageId @6 :Text;
  containerId @7 :Text;
  started @8 :Bool;
  allocatedResources @9 :List(KeyValue);
  resources @10 :ResourceRequirements;
  volumeMounts @11 :List(VolumeMountStatus);
}

# ContainerState holds a possible state of container.
struct ContainerState {
  waiting @0 :ContainerStateWaiting;
  running @1 :ContainerStateRunning;
  terminated @2 :ContainerStateTerminated;
}

# ContainerStateWaiting is a waiting state of a container.
struct ContainerStateWaiting {
  reason @0 :Text;
  message @1 :Text;
}

# ContainerStateRunning is a running state of a container.
struct ContainerStateRunning {
  startedAt @0 :Time;
}

# ContainerStateTerminated is a terminated state of a container.
struct ContainerStateTerminated {
  exitCode @0 :Int32;
  signal @1 :Int32;
  reason @2 :Text;
  message @3 :Text;
  startedAt @4 :Time;
  finishedAt @5 :Time;
  containerId @6 :Text;
}

# ContainerResizePolicy represents resource resize policy for a container.
struct ContainerResizePolicy {
  resourceName @0 :Text;
  restartPolicy @1 :ResourceResizeRestartPolicy;
}

# Volume represents a named volume in a pod.
struct Volume {
  name @0 :Text;
  volumeSource @1 :VolumeSource;
}

# VolumeSource represents the source location of a volume to mount.
struct VolumeSource {
  hostPath @0 :HostPathVolumeSource;
  emptyDir @1 :EmptyDirVolumeSource;
  gcePersistentDisk @2 :GCEPersistentDiskVolumeSource;
  awsElasticBlockStore @3 :AWSElasticBlockStoreVolumeSource;
  secret @4 :SecretVolumeSource;
  nfs @5 :NFSVolumeSource;
  persistentVolumeClaim @6 :PersistentVolumeClaimVolumeSource;
  downwardApi @7 :DownwardAPIVolumeSource;
  configMap @8 :ConfigMapVolumeSource;
  projected @9 :ProjectedVolumeSource;
  csi @10 :CSIVolumeSource;
  ephemeral @11 :EphemeralVolumeSource;
}

# HostPathVolumeSource represents a host path mapped into a pod.
struct HostPathVolumeSource {
  path @0 :Text;
  type @1 :HostPathType;
}

# EmptyDirVolumeSource represents an empty directory for a pod.
struct EmptyDirVolumeSource {
  medium @0 :StorageMedium;
  sizeLimit @1 :Text;
}

# GCEPersistentDiskVolumeSource represents a GCE persistent disk.
struct GCEPersistentDiskVolumeSource {
  pdName @0 :Text;
  fsType @1 :Text;
  partition @2 :Int32;
  readOnly @3 :Bool;
}

# AWSElasticBlockStoreVolumeSource represents an AWS EBS disk.
struct AWSElasticBlockStoreVolumeSource {
  volumeId @0 :Text;
  fsType @1 :Text;
  partition @2 :Int32;
  readOnly @3 :Bool;
}

# SecretVolumeSource adapts a Secret into a volume.
struct SecretVolumeSource {
  secretName @0 :Text;
  items @1 :List(KeyToPath);
  defaultMode @2 :Int32;
  optional @3 :Bool;
}

# NFSVolumeSource represents an NFS mount that lasts the lifetime of a pod.
struct NFSVolumeSource {
  server @0 :Text;
  path @1 :Text;
  readOnly @2 :Bool;
}

# PersistentVolumeClaimVolumeSource references a PersistentVolumeClaim.
struct PersistentVolumeClaimVolumeSource {
  claimName @0 :Text;
  readOnly @1 :Bool;
}

# DownwardAPIVolumeSource represents a volume with downward API info.
struct DownwardAPIVolumeSource {
  items @0 :List(DownwardAPIVolumeFile);
  defaultMode @1 :Int32;
}

# DownwardAPIVolumeFile represents info to project into a volume.
struct DownwardAPIVolumeFile {
  path @0 :Text;
  fieldRef @1 :ObjectFieldSelector;
  resourceFieldRef @2 :ResourceFieldSelector;
  mode @3 :Int32;
}

# ConfigMapVolumeSource adapts a ConfigMap into a volume.
struct ConfigMapVolumeSource {
  name @0 :Text;
  items @1 :List(KeyToPath);
  defaultMode @2 :Int32;
  optional @3 :Bool;
}

# KeyToPath maps a string key to a path within a volume.
struct KeyToPath {
  key @0 :Text;
  path @1 :Text;
  mode @2 :Int32;
}

# ProjectedVolumeSource represents a projected volume source.
struct ProjectedVolumeSource {
  sources @0 :List(VolumeProjection);
  defaultMode @1 :Int32;
}

# VolumeProjection contains the projected volume sources.
struct VolumeProjection {
  secret @0 :SecretProjection;
  downwardApi @1 :DownwardAPIProjection;
  configMap @2 :ConfigMapProjection;
  serviceAccountToken @3 :ServiceAccountTokenProjection;
  clusterTrustBundle @4 :ClusterTrustBundleProjection;
}

# SecretProjection adapts a secret into a projected volume.
struct SecretProjection {
  name @0 :Text;
  items @1 :List(KeyToPath);
  optional @2 :Bool;
}

# DownwardAPIProjection projects downward API info.
struct DownwardAPIProjection {
  items @0 :List(DownwardAPIVolumeFile);
}

# ConfigMapProjection adapts a ConfigMap into a projected volume.
struct ConfigMapProjection {
  name @0 :Text;
  items @1 :List(KeyToPath);
  optional @2 :Bool;
}

# ServiceAccountTokenProjection represents a projected service account token.
struct ServiceAccountTokenProjection {
  audience @0 :Text;
  expirationSeconds @1 :Int64;
  path @2 :Text;
}

# ClusterTrustBundleProjection describes how to select a set of ClusterTrustBundle objects.
struct ClusterTrustBundleProjection {
  name @0 :Text;
  signerName @1 :Text;
  labelSelector @2 :LabelSelector;
  optional @3 :Bool;
  path @4 :Text;
}

# CSIVolumeSource represents a CSI volume.
struct CSIVolumeSource {
  driver @0 :Text;
  readOnly @1 :Bool;
  fsType @2 :Text;
  volumeAttributes @3 :List(KeyValue);
  nodePublishSecretRef @4 :LocalObjectReference;
}

# EphemeralVolumeSource represents an ephemeral volume.
struct EphemeralVolumeSource {
  volumeClaimTemplate @0 :PersistentVolumeClaimTemplate;
}

# PersistentVolumeClaimTemplate is used to produce PersistentVolumeClaim objects.
struct PersistentVolumeClaimTemplate {
  metadata @0 :ObjectMeta;
  spec @1 :PersistentVolumeClaimSpec;
}

# PersistentVolumeClaimSpec describes the common attributes of storage devices.
struct PersistentVolumeClaimSpec {
  accessModes @0 :List(PersistentVolumeAccessMode);
  selector @1 :LabelSelector;
  resources @2 :ResourceRequirements;
  volumeName @3 :Text;
  storageClassName @4 :Text;
  volumeMode @5 :PersistentVolumeMode;
  dataSource @6 :TypedLocalObjectReference;
  dataSourceRef @7 :TypedObjectReference;
  volumeAttributesClassName @8 :List(Text);
}

# VolumeMount describes a mounting of a Volume within a container.
struct VolumeMount {
  name @0 :Text;
  readOnly @1 :Bool;
  recursiveReadOnly @2 :RecursiveReadOnlyMode;
  mountPath @3 :Text;
  subPath @4 :Text;
  mountPropagation @5 :MountPropagationMode;
  subPathExpr @6 :Text;
}

# VolumeMountStatus shows status of a mount.
struct VolumeMountStatus {
  name @0 :Text;
  mountPath @1 :Text;
  readOnly @2 :Bool;
  recursiveReadOnly @3 :RecursiveReadOnlyMode;
}

# VolumeDevice describes a mapping of a raw block device within a container.
struct VolumeDevice {
  name @0 :Text;
  devicePath @1 :Text;
}

# EnvVar represents an environment variable.
struct EnvVar {
  name @0 :Text;
  value @1 :Text;
  valueFrom @2 :EnvVarSource;
}

# EnvVarSource represents a source for the value of an EnvVar.
struct EnvVarSource {
  fieldRef @0 :ObjectFieldSelector;
  resourceFieldRef @1 :ResourceFieldSelector;
  configMapKeyRef @2 :ConfigMapKeySelector;
  secretKeyRef @3 :SecretKeySelector;
}

# EnvFromSource represents the source of a set of ConfigMaps.
struct EnvFromSource {
  prefix @0 :Text;
  configMapRef @1 :ConfigMapEnvSource;
  secretRef @2 :SecretEnvSource;
}

# ConfigMapEnvSource selects a ConfigMap to populate the environment.
struct ConfigMapEnvSource {
  name @0 :Text;
  optional @1 :Bool;
}

# SecretEnvSource selects a Secret to populate the environment.
struct SecretEnvSource {
  name @0 :Text;
  optional @1 :Bool;
}

# ObjectFieldSelector selects an APIVersioned field of an object.
struct ObjectFieldSelector {
  apiVersion @0 :Text;
  fieldPath @1 :Text;
}

# ResourceFieldSelector represents container resources (cpu, memory) and their output format.
struct ResourceFieldSelector {
  containerName @0 :Text;
  resource @1 :Text;
  divisor @2 :Text;
}

# ConfigMapKeySelector selects a key of a ConfigMap.
struct ConfigMapKeySelector {
  name @0 :Text;
  key @1 :Text;
  optional @2 :Bool;
}

# SecretKeySelector selects a key of a Secret.
struct SecretKeySelector {
  name @0 :Text;
  key @1 :Text;
  optional @2 :Bool;
}

# ResourceRequirements describes the compute resource requirements.
struct ResourceRequirements {
  limits @0 :List(KeyValue);
  requests @1 :List(KeyValue);
  claims @2 :List(ResourceClaim);
}

# ResourceClaim references one entry in PodSpec.ResourceClaims.
struct ResourceClaim {
  name @0 :Text;
  request @1 :Text;
}

# Probe describes a health check to be performed against a container.
struct Probe {
  handler @0 :ProbeHandler;
  initialDelaySeconds @1 :Int32;
  timeoutSeconds @2 :Int32;
  periodSeconds @3 :Int32;
  successThreshold @4 :Int32;
  failureThreshold @5 :Int32;
  terminationGracePeriodSeconds @6 :Int64;
}

# ProbeHandler defines a specific action that should be taken.
struct ProbeHandler {
  exec @0 :ExecAction;
  httpGet @1 :HTTPGetAction;
  tcpSocket @2 :TCPSocketAction;
  grpc @3 :GRPCAction;
}

# ExecAction describes a "run in container" action.
struct ExecAction {
  command @0 :List(Text);
}

# HTTPGetAction describes an action based on HTTP Get requests.
struct HTTPGetAction {
  path @0 :Text;
  port @1 :IntOrString;
  host @2 :Text;
  scheme @3 :URIScheme;
  httpHeaders @4 :List(HTTPHeader);
}

# HTTPHeader describes a custom header to be used in HTTP probes.
struct HTTPHeader {
  name @0 :Text;
  value @1 :Text;
}

# TCPSocketAction describes an action based on opening a socket.
struct TCPSocketAction {
  port @0 :IntOrString;
  host @1 :Text;
}

# GRPCAction describes an action involving a GRPC port.
struct GRPCAction {
  port @0 :Int32;
  service @1 :Text;
}

# IntOrString is a type that can hold an int32 or a string.
struct IntOrString {
  union {
    intVal @0 :Int32;
    strVal @1 :Text;
  }
}

# Lifecycle describes actions that the management system should take.
struct Lifecycle {
  postStart @0 :LifecycleHandler;
  preStop @1 :LifecycleHandler;
}

# LifecycleHandler defines a specific action that should be taken.
struct LifecycleHandler {
  exec @0 :ExecAction;
  httpGet @1 :HTTPGetAction;
  tcpSocket @2 :TCPSocketAction;
  sleep @3 :SleepAction;
}

# SleepAction is for sleeping in lifecycle hooks.
struct SleepAction {
  seconds @0 :Int64;
}

# SecurityContext holds security configuration for a container.
struct SecurityContext {
  capabilities @0 :Capabilities;
  privileged @1 :Bool;
  seLinuxOptions @2 :SELinuxOptions;
  windowsOptions @3 :WindowsSecurityContextOptions;
  runAsUser @4 :Int64;
  runAsGroup @5 :Int64;
  runAsNonRoot @6 :Bool;
  readOnlyRootFilesystem @7 :Bool;
  allowPrivilegeEscalation @8 :Bool;
  procMount @9 :ProcMountType;
  seccompProfile @10 :SeccompProfile;
  appArmorProfile @11 :AppArmorProfile;
}

# PodSecurityContext holds pod-level security attributes.
struct PodSecurityContext {
  seLinuxOptions @0 :SELinuxOptions;
  windowsOptions @1 :WindowsSecurityContextOptions;
  runAsUser @2 :Int64;
  runAsGroup @3 :Int64;
  runAsNonRoot @4 :Bool;
  supplementalGroups @5 :List(Int64);
  supplementalGroupsPolicy @6 :SupplementalGroupsPolicy;
  fsGroup @7 :Int64;
  sysctls @8 :List(Sysctl);
  fsGroupChangePolicy @9 :PodFSGroupChangePolicy;
  seccompProfile @10 :SeccompProfile;
  appArmorProfile @11 :AppArmorProfile;
  seLinuxChangePolicy @12 :PodSELinuxChangePolicy;
}

# Capabilities represent POSIX capabilities to add/drop.
struct Capabilities {
  add @0 :List(Text);
  drop @1 :List(Text);
}

# SELinuxOptions are the labels to be applied to the container.
struct SELinuxOptions {
  user @0 :Text;
  role @1 :Text;
  type @2 :Text;
  level @3 :Text;
}

# WindowsSecurityContextOptions contain Windows-specific options.
struct WindowsSecurityContextOptions {
  gmsaCredentialSpecName @0 :Text;
  gmsaCredentialSpec @1 :Text;
  runAsUserName @2 :Text;
  hostProcess @3 :Bool;
}

# SeccompProfile defines a pod/container's seccomp profile settings.
struct SeccompProfile {
  type @0 :SeccompProfileType;
  localhostProfile @1 :Text;
}

# AppArmorProfile defines a pod/container's AppArmor profile settings.
struct AppArmorProfile {
  type @0 :AppArmorProfileType;
  localhostProfile @1 :Text;
}

# Sysctl defines a kernel parameter to be set.
struct Sysctl {
  name @0 :Text;
  value @1 :Text;
}

# Affinity is a group of affinity scheduling rules.
struct Affinity {
  nodeAffinity @0 :NodeAffinity;
  podAffinity @1 :PodAffinity;
  podAntiAffinity @2 :PodAntiAffinity;
}

# NodeAffinity is a group of node affinity scheduling rules.
struct NodeAffinity {
  requiredDuringSchedulingIgnoredDuringExecution @0 :NodeSelector;
  preferredDuringSchedulingIgnoredDuringExecution @1 :List(PreferredSchedulingTerm);
}

# NodeSelector represents the union of the results of one or more label queries.
struct NodeSelector {
  nodeSelectorTerms @0 :List(NodeSelectorTerm);
}

# NodeSelectorTerm represents expressions and fields required to select node(s).
struct NodeSelectorTerm {
  matchExpressions @0 :List(NodeSelectorRequirement);
  matchFields @1 :List(NodeSelectorRequirement);
}

# NodeSelectorRequirement is a selector that contains values and an operator.
struct NodeSelectorRequirement {
  key @0 :Text;
  operator @1 :NodeSelectorOperator;
  values @2 :List(Text);
}

# PreferredSchedulingTerm is a scheduling term with weight.
struct PreferredSchedulingTerm {
  weight @0 :Int32;
  preference @1 :NodeSelectorTerm;
}

# PodAffinity is a group of inter pod affinity scheduling rules.
struct PodAffinity {
  requiredDuringSchedulingIgnoredDuringExecution @0 :List(PodAffinityTerm);
  preferredDuringSchedulingIgnoredDuringExecution @1 :List(WeightedPodAffinityTerm);
}

# PodAntiAffinity is a group of inter pod anti affinity scheduling rules.
struct PodAntiAffinity {
  requiredDuringSchedulingIgnoredDuringExecution @0 :List(PodAffinityTerm);
  preferredDuringSchedulingIgnoredDuringExecution @1 :List(WeightedPodAffinityTerm);
}

# PodAffinityTerm defines a set of pods for this pod to co-located/anti-located with.
struct PodAffinityTerm {
  labelSelector @0 :LabelSelector;
  namespaces @1 :List(Text);
  topologyKey @2 :Text;
  namespaceSelector @3 :LabelSelector;
  matchLabelKeys @4 :List(Text);
  mismatchLabelKeys @5 :List(Text);
}

# WeightedPodAffinityTerm represents a pod affinity term with a weight.
struct WeightedPodAffinityTerm {
  weight @0 :Int32;
  podAffinityTerm @1 :PodAffinityTerm;
}

# LabelSelector is a label query over a set of resources.
struct LabelSelector {
  matchLabels @0 :List(KeyValue);
  matchExpressions @1 :List(LabelSelectorRequirement);
}

# LabelSelectorRequirement is a selector that contains values and an operator.
struct LabelSelectorRequirement {
  key @0 :Text;
  operator @1 :LabelSelectorOperator;
  values @2 :List(Text);
}

# Toleration defines that a pod tolerates a taint.
struct Toleration {
  key @0 :Text;
  operator @1 :TolerationOperator;
  value @2 :Text;
  effect @3 :TaintEffect;
  tolerationSeconds @4 :Int64;
}

# TopologySpreadConstraint specifies how to spread matching pods.
struct TopologySpreadConstraint {
  maxSkew @0 :Int32;
  topologyKey @1 :Text;
  whenUnsatisfiable @2 :UnsatisfiableConstraintAction;
  labelSelector @3 :LabelSelector;
  minDomains @4 :Int32;
  nodeAffinityPolicy @5 :NodeInclusionPolicy;
  nodeTaintsPolicy @6 :NodeInclusionPolicy;
  matchLabelKeys @7 :List(Text);
}

# HostAlias holds the mapping between IP and hostnames.
struct HostAlias {
  ip @0 :Text;
  hostnames @1 :List(Text);
}

# HostIP represents a single IP address allocated to the host.
struct HostIP {
  ip @0 :Text;
}

# PodIP represents a single IP address allocated to the pod.
struct PodIP {
  ip @0 :Text;
}

# PodCondition contains details for the current condition of this pod.
struct PodCondition {
  type @0 :PodConditionType;
  status @1 :ConditionStatus;
  lastProbeTime @2 :Time;
  lastTransitionTime @3 :Time;
  reason @4 :Text;
  message @5 :Text;
}

# PodDNSConfig defines the DNS parameters of a pod.
struct PodDNSConfig {
  nameservers @0 :List(Text);
  searches @1 :List(Text);
  options @2 :List(PodDNSConfigOption);
}

# PodDNSConfigOption defines DNS resolver options.
struct PodDNSConfigOption {
  name @0 :Text;
  value @1 :Text;
}

# PodReadinessGate contains the reference to a pod condition.
struct PodReadinessGate {
  conditionType @0 :PodConditionType;
}

# PodOS defines the OS parameters of a pod.
struct PodOS {
  name @0 :Text;
}

# PodSchedulingGate is associated to a Pod to guard its scheduling.
struct PodSchedulingGate {
  name @0 :Text;
}

# PodResourceClaim references a ResourceClaim.
struct PodResourceClaim {
  name @0 :Text;
  resourceClaimName @1 :Text;
  resourceClaimTemplateName @2 :Text;
}

# PodResourceClaimStatus is stored in the PodStatus for each PodResourceClaim.
struct PodResourceClaimStatus {
  name @0 :Text;
  resourceClaimName @1 :Text;
}

# LocalObjectReference contains enough information to let you locate the referenced object.
struct LocalObjectReference {
  name @0 :Text;
}

# TypedLocalObjectReference contains enough information to locate the typed referenced object.
struct TypedLocalObjectReference {
  apiGroup @0 :Text;
  kind @1 :Text;
  name @2 :Text;
}

# TypedObjectReference contains enough information to locate the typed referenced object.
struct TypedObjectReference {
  apiGroup @0 :Text;
  kind @1 :Text;
  name @2 :Text;
  namespace @3 :Text;
}

# Enumerations

enum RestartPolicy {
  restartPolicyUnspecified @0;
  restartPolicyAlways @1;
  restartPolicyOnFailure @2;
  restartPolicyNever @3;
}

enum PullPolicy {
  pullPolicyUnspecified @0;
  pullPolicyAlways @1;
  pullPolicyNever @2;
  pullPolicyIfNotPresent @3;
}

enum Protocol {
  protocolUnspecified @0;
  protocolTcp @1;
  protocolUdp @2;
  protocolSctp @3;
}

enum DNSPolicy {
  dnsPolicyUnspecified @0;
  dnsPolicyClusterFirstWithHostNet @1;
  dnsPolicyClusterFirst @2;
  dnsPolicyDefault @3;
  dnsPolicyNone @4;
}

enum PodPhase {
  podPhaseUnspecified @0;
  podPhasePending @1;
  podPhaseRunning @2;
  podPhaseSucceeded @3;
  podPhaseFailed @4;
  podPhaseUnknown @5;
}

enum PodQOSClass {
  podQosClassUnspecified @0;
  podQosClassGuaranteed @1;
  podQosClassBurstable @2;
  podQosClassBestEffort @3;
}

enum ContainerRestartPolicy {
  containerRestartPolicyUnspecified @0;
  containerRestartPolicyAlways @1;
}

enum TerminationMessagePolicy {
  terminationMessagePolicyUnspecified @0;
  terminationMessagePolicyFile @1;
  terminationMessagePolicyFallbackToLogsOnError @2;
}

enum URIScheme {
  uriSchemeUnspecified @0;
  uriSchemeHttp @1;
  uriSchemeHttps @2;
}

enum HostPathType {
  hostPathTypeUnspecified @0;
  hostPathTypeDirectoryOrCreate @1;
  hostPathTypeDirectory @2;
  hostPathTypeFileOrCreate @3;
  hostPathTypeFile @4;
  hostPathTypeSocket @5;
  hostPathTypeCharDevice @6;
  hostPathTypeBlockDevice @7;
}

enum StorageMedium {
  storageMediumUnspecified @0;
  storageMediumDefault @1;
  storageMediumMemory @2;
  storageMediumHugePages @3;
  storageMediumHugePages2Mi @4;
  storageMediumHugePages1Gi @5;
}

enum MountPropagationMode {
  mountPropagationModeUnspecified @0;
  mountPropagationModeNone @1;
  mountPropagationModeHostToContainer @2;
  mountPropagationModeBidirectional @3;
}

enum RecursiveReadOnlyMode {
  recursiveReadOnlyModeUnspecified @0;
  recursiveReadOnlyModeDisabled @1;
  recursiveReadOnlyModeIfPossible @2;
  recursiveReadOnlyModeEnabled @3;
}

enum PreemptionPolicy {
  preemptionPolicyUnspecified @0;
  preemptionPolicyPreemptLowerPriority @1;
  preemptionPolicyNever @2;
}

enum ProcMountType {
  procMountTypeUnspecified @0;
  procMountTypeDefault @1;
  procMountTypeUnmasked @2;
}

enum SeccompProfileType {
  seccompProfileTypeUnspecified @0;
  seccompProfileTypeUnconfined @1;
  seccompProfileTypeRuntimeDefault @2;
  seccompProfileTypeLocalhost @3;
}

enum AppArmorProfileType {
  appArmorProfileTypeUnspecified @0;
  appArmorProfileTypeUnconfined @1;
  appArmorProfileTypeRuntimeDefault @2;
  appArmorProfileTypeLocalhost @3;
}

enum SupplementalGroupsPolicy {
  supplementalGroupsPolicyUnspecified @0;
  supplementalGroupsPolicyMerge @1;
  supplementalGroupsPolicyStrict @2;
}

enum PodFSGroupChangePolicy {
  podFsGroupChangePolicyUnspecified @0;
  podFsGroupChangePolicyOnRootMismatch @1;
  podFsGroupChangePolicyAlways @2;
}

enum PodSELinuxChangePolicy {
  podSeLinuxChangePolicyUnspecified @0;
  podSeLinuxChangePolicyRecursive @1;
  podSeLinuxChangePolicyMountOption @2;
}

enum NodeSelectorOperator {
  nodeSelectorOperatorUnspecified @0;
  nodeSelectorOperatorIn @1;
  nodeSelectorOperatorNotIn @2;
  nodeSelectorOperatorExists @3;
  nodeSelectorOperatorDoesNotExist @4;
  nodeSelectorOperatorGt @5;
  nodeSelectorOperatorLt @6;
}

enum LabelSelectorOperator {
  labelSelectorOperatorUnspecified @0;
  labelSelectorOperatorIn @1;
  labelSelectorOperatorNotIn @2;
  labelSelectorOperatorExists @3;
  labelSelectorOperatorDoesNotExist @4;
}

enum TolerationOperator {
  tolerationOperatorUnspecified @0;
  tolerationOperatorExists @1;
  tolerationOperatorEqual @2;
}

enum TaintEffect {
  taintEffectUnspecified @0;
  taintEffectNoSchedule @1;
  taintEffectPreferNoSchedule @2;
  taintEffectNoExecute @3;
}

enum UnsatisfiableConstraintAction {
  unsatisfiableConstraintActionUnspecified @0;
  unsatisfiableConstraintActionDoNotSchedule @1;
  unsatisfiableConstraintActionScheduleAnyway @2;
}

enum NodeInclusionPolicy {
  nodeInclusionPolicyUnspecified @0;
  nodeInclusionPolicyIgnore @1;
  nodeInclusionPolicyHonor @2;
}

enum PodConditionType {
  podConditionTypeUnspecified @0;
  podConditionTypeContainersReady @1;
  podConditionTypeInitialized @2;
  podConditionTypeReady @3;
  podConditionTypePodScheduled @4;
  podConditionTypeDisruptionTarget @5;
  podConditionTypePodReadyToStartContainers @6;
}

enum ConditionStatus {
  conditionStatusUnspecified @0;
  conditionStatusTrue @1;
  conditionStatusFalse @2;
  conditionStatusUnknown @3;
}

enum PodResizeStatus {
  podResizeStatusUnspecified @0;
  podResizeStatusProposed @1;
  podResizeStatusInProgress @2;
  podResizeStatusDeferred @3;
  podResizeStatusInfeasible @4;
}

enum PersistentVolumeAccessMode {
  persistentVolumeAccessModeUnspecified @0;
  persistentVolumeAccessModeReadWriteOnce @1;
  persistentVolumeAccessModeReadOnlyMany @2;
  persistentVolumeAccessModeReadWriteMany @3;
  persistentVolumeAccessModeReadWriteOncePod @4;
}

enum PersistentVolumeMode {
  persistentVolumeModeUnspecified @0;
  persistentVolumeModeBlock @1;
  persistentVolumeModeFilesystem @2;
}

enum ResourceResizeRestartPolicy {
  resourceResizeRestartPolicyUnspecified @0;
  resourceResizeRestartPolicyNotRequired @1;
  resourceResizeRestartPolicyRestartContainer @2;
}
