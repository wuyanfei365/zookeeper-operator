package v1

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	DefaultZkContainerRepository = "wuyanfei365/common"
	DefaultZkContainerVersion = "zk-3.5.5"
	DefaultZkContainerPolicy = "IfNotPresent"
	DefaultTerminationGracePeriod = 30
	DefaultZookeeperCacheVolumeSize = "20Gi"
	DefaultReadinessProbeInitialDelaySeconds = 10
	DefaultReadinessProbePeriodSeconds = 10
	DefaultReadinessProbeFailureThreshold = 3
	DefaultReadinessProbeSuccessThreshold = 1
	DefaultReadinessProbeTimeoutSeconds = 10
	DefaultLivenessProbeInitialDelaySeconds = 10
	DefaultLivenessProbePeriodSeconds = 10
	DefaultLivenessProbeFailureThreshold = 3
	DefaultLivenessProbeTimeoutSeconds = 10
)

type Probes struct {
	// +optional
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`
	// +optional
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`
}

func (s *ZookeeperClusterSpec) withDefaults(z *ZookeeperCluster) (changed bool) {
	changed = s.Image.withDefaults()
	if s.Conf.withDefaults() {
		changed = true
	}
	if s.Replicas == 0 {
		s.Replicas = 3
		changed = true
	}
	if s.Probes == nil {
		changed = true
		s.Probes = &Probes{}
	}
	if s.Probes.withDefaults() {
		changed = true
	}

	if s.Ports == nil {
		s.Ports = []v1.ContainerPort{
			{
				Name:          "client",
				ContainerPort: 2181,
			},
			{
				Name:          "quorum",
				ContainerPort: 2888,
			},
			{
				Name:          "leader-election",
				ContainerPort: 3888,
			},
			{
				Name:          "metrics",
				ContainerPort: 7000,
			},
			{
				Name:          "admin-server",
				ContainerPort: 8080,
			},
		}
		changed = true
	} else {
		var (
			foundClient, foundQuorum, foundLeader, foundMetrics, foundAdmin bool
		)
		for i := 0; i < len(s.Ports); i++ {
			if s.Ports[i].Name == "client" {
				foundClient = true
			} else if s.Ports[i].Name == "quorum" {
				foundQuorum = true
			} else if s.Ports[i].Name == "leader-election" {
				foundLeader = true
			} else if s.Ports[i].Name == "metrics" {
				foundMetrics = true
			} else if s.Ports[i].Name == "admin-server" {
				foundAdmin = true
			}
		}
		if !foundClient {
			ports := v1.ContainerPort{Name: "client", ContainerPort: 2181}
			s.Ports = append(s.Ports, ports)
			changed = true
		}
		if !foundQuorum {
			ports := v1.ContainerPort{Name: "quorum", ContainerPort: 2888}
			s.Ports = append(s.Ports, ports)
			changed = true
		}
		if !foundLeader {
			ports := v1.ContainerPort{Name: "leader-election", ContainerPort: 3888}
			s.Ports = append(s.Ports, ports)
			changed = true
		}
		if !foundMetrics {
			ports := v1.ContainerPort{Name: "metrics", ContainerPort: 7000}
			s.Ports = append(s.Ports, ports)
			changed = true
		}
		if !foundAdmin {
			ports := v1.ContainerPort{Name: "admin-server", ContainerPort: 8080}
			s.Ports = append(s.Ports, ports)
			changed = true
		}
	}

	if z.Spec.Labels == nil {
		z.Spec.Labels = map[string]string{}
		changed = true
	}
	if _, ok := z.Spec.Labels["app"]; !ok {
		z.Spec.Labels["app"] = z.GetName()
		changed = true
	}
	if _, ok := z.Spec.Labels["release"]; !ok {
		z.Spec.Labels["release"] = z.GetName()
		changed = true
	}
	if s.Pod.withDefaults(z) {
		changed = true
	}
	if strings.EqualFold(s.StorageType, "ephemeral") {
		if s.Ephemeral == nil {
			s.Ephemeral = &Ephemeral{}
			s.Ephemeral.EmptyDirVolumeSource = v1.EmptyDirVolumeSource{}
			changed = true
		}
	} else {
		if s.Persistence == nil {
			s.StorageType = "persistence"
			s.Persistence = &Persistence{}
			changed = true
		}
		if s.Persistence.withDefaults() {
			s.StorageType = "persistence"
			changed = true
		}
	}
	return changed
}

type Probe struct {
	// +kubebuilder:validation:Minimum=0
	// +optional
	InitialDelaySeconds int32 `json:"initialDelaySeconds"`
	// +kubebuilder:validation:Minimum=0
	// +optional
	PeriodSeconds int32 `json:"periodSeconds"`
	// +kubebuilder:validation:Minimum=0
	// +optional
	FailureThreshold int32 `json:"failureThreshold"`
	// +kubebuilder:validation:Minimum=0
	// +optional
	SuccessThreshold int32 `json:"successThreshold"`
	// +kubebuilder:validation:Minimum=0
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds"`
}

func (z *ZookeeperCluster) WithDefaults() bool {
	return z.Spec.withDefaults(z)
}

func (z *ZookeeperCluster) ConfigMapName() string {
	return fmt.Sprintf("%s-configmap", z.GetName())
}

func (z *ZookeeperCluster) GetKubernetesClusterDomain() string {
	if z.Spec.KubernetesClusterDomain == "" {
		return "cluster.local"
	}
	return z.Spec.KubernetesClusterDomain
}

func (z *ZookeeperCluster) ZookeeperPorts() Ports {
	ports := Ports{}
	for _, p := range z.Spec.Ports {
		if p.Name == "client" {
			ports.Client = p.ContainerPort
		} else if p.Name == "quorum" {
			ports.Quorum = p.ContainerPort
		} else if p.Name == "leader-election" {
			ports.Leader = p.ContainerPort
		} else if p.Name == "metrics" {
			ports.Metrics = p.ContainerPort
		} else if p.Name == "admin-server" {
			ports.AdminServer = p.ContainerPort
		}
	}
	return ports
}

func (z *ZookeeperCluster) GetClientServiceName() string {
	return fmt.Sprintf("%s-client", z.GetName())
}

func (z *ZookeeperCluster) GetAdminServerServiceName() string {
	return fmt.Sprintf("%s-admin-server", z.GetName())
}

func (z *ZookeeperCluster) GetTriggerRollingRestart() bool {
	return z.Spec.TriggerRollingRestart
}

func (z *ZookeeperCluster) SetTriggerRollingRestart(val bool) {
	z.Spec.TriggerRollingRestart = val
}

type Ports struct {
	Client      int32
	Quorum      int32
	Leader      int32
	Metrics     int32
	AdminServer int32
}

type ContainerImage struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	// +kubebuilder:validation:Enum="Always";"Never";"IfNotPresent"
	PullPolicy v1.PullPolicy `json:"pullPolicy,omitempty"`
}

func (c *ContainerImage) withDefaults() (changed bool) {
	if c.Repository == "" {
		changed = true
		c.Repository = DefaultZkContainerRepository
	}
	if c.Tag == "" {
		changed = true
		c.Tag = DefaultZkContainerVersion
	}
	if c.PullPolicy == "" {
		changed = true
		c.PullPolicy = DefaultZkContainerPolicy
	}
	return changed
}

func (c *ContainerImage) ToString() string {
	return fmt.Sprintf("%s:%s", c.Repository, c.Tag)
}

type PodPolicy struct {

	Labels map[string]string `json:"labels,omitempty"`

	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	Affinity *v1.Affinity `json:"affinity,omitempty"`

	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	Env []v1.EnvVar `json:"env,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	SecurityContext *v1.PodSecurityContext `json:"securityContext,omitempty"`

	// +kubebuilder:validation:Minimum=0
	TerminationGracePeriodSeconds int64 `json:"terminationGracePeriodSeconds,omitempty"`

	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

func (p *PodPolicy) withDefaults(z *ZookeeperCluster) (changed bool) {
	if p.Labels == nil {
		p.Labels = map[string]string{}
		changed = true
	}
	if p.TerminationGracePeriodSeconds == 0 {
		p.TerminationGracePeriodSeconds = DefaultTerminationGracePeriod
		changed = true
	}
	if p.ServiceAccountName == "" {
		p.ServiceAccountName = "default"
		changed = true
	}
	if z.Spec.Pod.Labels == nil {
		p.Labels = map[string]string{}
		changed = true
	}
	if _, ok := p.Labels["app"]; !ok {
		p.Labels["app"] = z.GetName()
		changed = true
	}
	if _, ok := p.Labels["release"]; !ok {
		p.Labels["release"] = z.GetName()
		changed = true
	}
	if p.Affinity == nil {
		p.Affinity = &v1.Affinity{
			PodAntiAffinity: &v1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
					{
						Weight: 20,
						PodAffinityTerm: v1.PodAffinityTerm{
							TopologyKey: "kubernetes.io/hostname",
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "app",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{z.GetName()},
									},
								},
							},
						},
					},
				},
			},
		}
		changed = true
	}
	return changed
}

type AdminServerServicePolicy struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	External bool `json:"external,omitempty"`
}

type ClientServicePolicy struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

type HeadlessServicePolicy struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

func (s *Probes) withDefaults() (changed bool) {
	if s.ReadinessProbe == nil {
		changed = true
		s.ReadinessProbe = &Probe{}
		s.ReadinessProbe.InitialDelaySeconds = DefaultReadinessProbeInitialDelaySeconds
		s.ReadinessProbe.PeriodSeconds = DefaultReadinessProbePeriodSeconds
		s.ReadinessProbe.FailureThreshold = DefaultReadinessProbeFailureThreshold
		s.ReadinessProbe.SuccessThreshold = DefaultReadinessProbeSuccessThreshold
		s.ReadinessProbe.TimeoutSeconds = DefaultReadinessProbeTimeoutSeconds
	}

	if s.LivenessProbe == nil {
		changed = true
		s.LivenessProbe = &Probe{}
		s.LivenessProbe.InitialDelaySeconds = DefaultLivenessProbeInitialDelaySeconds
		s.LivenessProbe.PeriodSeconds = DefaultLivenessProbePeriodSeconds
		s.LivenessProbe.FailureThreshold = DefaultLivenessProbeFailureThreshold
		s.LivenessProbe.TimeoutSeconds = DefaultLivenessProbeTimeoutSeconds
	}

	return changed
}

type ZookeeperConfig struct {

	InitLimit int `json:"initLimit,omitempty"`

	TickTime int `json:"tickTime,omitempty"`

	SyncLimit int `json:"syncLimit,omitempty"`

	GlobalOutstandingLimit int `json:"globalOutstandingLimit,omitempty"`

	PreAllocSize int `json:"preAllocSize,omitempty"`

	SnapCount int `json:"snapCount,omitempty"`

	CommitLogCount int `json:"commitLogCount,omitempty"`

	SnapSizeLimitInKb int `json:"snapSizeLimitInKb,omitempty"`

	MaxCnxns int `json:"maxCnxns,omitempty"`

	MaxClientCnxns int `json:"maxClientCnxns,omitempty"`

	MinSessionTimeout int `json:"minSessionTimeout,omitempty"`

	MaxSessionTimeout int `json:"maxSessionTimeout,omitempty"`

	AutoPurgeSnapRetainCount int `json:"autoPurgeSnapRetainCount,omitempty"`

	AutoPurgePurgeInterval int `json:"autoPurgePurgeInterval,omitempty"`

	QuorumListenOnAllIPs bool `json:"quorumListenOnAllIPs,omitempty"`

	AdditionalConfig map[string]string `json:"additionalConfig,omitempty"`
}

func (c *ZookeeperConfig) withDefaults() (changed bool) {
	if c.InitLimit == 0 {
		changed = true
		c.InitLimit = 10
	}
	if c.TickTime == 0 {
		changed = true
		c.TickTime = 2000
	}
	if c.SyncLimit == 0 {
		changed = true
		c.SyncLimit = 2
	}
	if c.GlobalOutstandingLimit == 0 {
		changed = true
		c.GlobalOutstandingLimit = 1000
	}
	if c.PreAllocSize == 0 {
		changed = true
		c.PreAllocSize = 65536
	}
	if c.SnapCount == 0 {
		changed = true
		c.SnapCount = 10000
	}
	if c.CommitLogCount == 0 {
		changed = true
		c.CommitLogCount = 500
	}
	if c.SnapSizeLimitInKb == 0 {
		changed = true
		c.SnapSizeLimitInKb = 4194304
	}
	if c.MaxClientCnxns == 0 {
		changed = true
		c.MaxClientCnxns = 60
	}
	if c.MinSessionTimeout == 0 {
		changed = true
		c.MinSessionTimeout = 2 * c.TickTime
	}
	if c.MaxSessionTimeout == 0 {
		changed = true
		c.MaxSessionTimeout = 20 * c.TickTime
	}
	if c.AutoPurgeSnapRetainCount == 0 {
		changed = true
		c.AutoPurgeSnapRetainCount = 3
	}
	if c.AutoPurgePurgeInterval == 0 {
		changed = true
		c.AutoPurgePurgeInterval = 1
	}

	return changed
}

type Persistence struct {
	// +kubebuilder:validation:Enum="Delete";"Retain"
	VolumeReclaimPolicy VolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`

	PersistentVolumeClaimSpec v1.PersistentVolumeClaimSpec `json:"spec,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
}

type Ephemeral struct {
	EmptyDirVolumeSource v1.EmptyDirVolumeSource `json:"emptydirvolumesource,omitempty"`
}

func (p *Persistence) withDefaults() (changed bool) {
	if !p.VolumeReclaimPolicy.isValid() {
		changed = true
		p.VolumeReclaimPolicy = VolumeReclaimPolicyRetain
	}
	p.PersistentVolumeClaimSpec.AccessModes = []v1.PersistentVolumeAccessMode{
		v1.ReadWriteOnce,
	}

	storage, _ := p.PersistentVolumeClaimSpec.Resources.Requests["storage"]
	if storage.IsZero() {
		p.PersistentVolumeClaimSpec.Resources.Requests = v1.ResourceList{
			v1.ResourceStorage: resource.MustParse(DefaultZookeeperCacheVolumeSize),
		}
		changed = true
	}
	return changed
}

func (v VolumeReclaimPolicy) isValid() bool {
	if v != VolumeReclaimPolicyDelete && v != VolumeReclaimPolicyRetain {
		return false
	}
	return true
}

type VolumeReclaimPolicy string

const (
	VolumeReclaimPolicyRetain VolumeReclaimPolicy = "Retain"
	VolumeReclaimPolicyDelete VolumeReclaimPolicy = "Delete"
)