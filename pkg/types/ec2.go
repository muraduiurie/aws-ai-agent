package types

// EC2Instance is a summary of an EC2 instance returned by list_ec2_instances.
type EC2Instance struct {
	InstanceID   string `json:"instance_id"`
	InstanceType string `json:"instance_type"`
	State        string `json:"state"`
	Name         string `json:"name,omitempty"`
	PublicIP     string `json:"public_ip,omitempty"`
	PrivateIP    string `json:"private_ip,omitempty"`
	LaunchTime   string `json:"launch_time,omitempty"`
}
