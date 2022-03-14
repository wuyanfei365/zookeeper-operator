package v1

// ZookeeperClusterStatus defines the observed state of ZookeeperCluster
type ZookeeperClusterStatus struct {
	Phase string `json:"phase,omitempty"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Members MembersStatus `json:"members,omitempty"`

	Replicas int32 `json:"replicas,omitempty"`

	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	InternalClientEndpoint string `json:"internalClientEndpoint,omitempty"`

	ExternalClientEndpoint string `json:"externalClientEndpoint,omitempty"`

	MetaRootCreated bool `json:"metaRootCreated,omitempty"`

	CurrentVersion string `json:"currentVersion,omitempty"`

	TargetVersion string `json:"targetVersion,omitempty"`

	Conditions []ClusterCondition `json:"conditions,omitempty"`
}
