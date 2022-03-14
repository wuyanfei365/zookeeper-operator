package v1

import v1 "k8s.io/api/core/v1"

// ZookeeperClusterSpec defines the desired state of ZookeeperCluster
type ZookeeperClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image ContainerImage `json:"image,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Replicas int32 `json:"replicas,omitempty"`

	Ports []v1.ContainerPort `json:"ports,omitempty"`

	Pod PodPolicy `json:"pod,omitempty"`

	AdminServerService AdminServerServicePolicy `json:"adminServerService,omitempty"`

	ClientService ClientServicePolicy `json:"clientService,omitempty"`

	TriggerRollingRestart bool `json:"triggerRollingRestart,omitempty"`

	HeadlessService HeadlessServicePolicy `json:"headlessService,omitempty"`

	StorageType string `json:"storageType,omitempty"`

	Persistence *Persistence `json:"persistence,omitempty"`

	Ephemeral *Ephemeral `json:"ephemeral,omitempty"`

	Conf ZookeeperConfig `json:"config,omitempty"`

	DomainName string `json:"domainName,omitempty"`

	KubernetesClusterDomain string `json:"kubernetesClusterDomain,omitempty"`

	Containers []v1.Container `json:"containers,omitempty"`

	InitContainers []v1.Container `json:"initContainers,omitempty"`

	Volumes []v1.Volume `json:"volumes,omitempty"`

	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`

	Probes *Probes `json:"probes,omitempty"`
}
