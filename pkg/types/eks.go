package types

// EKSCluster holds the details of an EKS cluster.
type EKSCluster struct {
	Name      string `json:"name"`
	ARN       string `json:"arn,omitempty"`
	Status    string `json:"status,omitempty"`
	Version   string `json:"version,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	RoleARN   string `json:"role_arn,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
