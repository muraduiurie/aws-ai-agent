package types

// PodSummary is a condensed view of a Kubernetes Pod returned by pod tools.
type PodSummary struct {
	Name              string          `json:"name"`
	Namespace         string          `json:"namespace"`
	Phase             string          `json:"phase"`
	NodeName          string          `json:"node_name,omitempty"`
	PodIP             string          `json:"pod_ip,omitempty"`
	CreationTimestamp string          `json:"creation_timestamp,omitempty"`
	Containers        []ContainerInfo `json:"containers,omitempty"`
}

// ContainerInfo holds the status of a single container within a pod.
type ContainerInfo struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Ready bool   `json:"ready"`
}
