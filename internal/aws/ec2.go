package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/muraduiurie/aws-ai-agent/pkg/types"
)

// ListEC2Instances returns all EC2 instances in the given region, optionally
// filtered by instance state. It pages through all results automatically.
func (f *Factory) ListEC2Instances(ctx context.Context, region, state string) ([]types.EC2Instance, error) {
	client := ec2.NewFromConfig(f.cfg, func(o *ec2.Options) {
		o.Region = region
	})

	input := &ec2.DescribeInstancesInput{}
	if state != "" {
		input.Filters = []ec2types.Filter{
			{Name: awssdk.String("instance-state-name"), Values: []string{state}},
		}
	}

	paginator := ec2.NewDescribeInstancesPaginator(client, input)

	var instances []types.EC2Instance
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe instances: %w", err)
		}
		for _, reservation := range page.Reservations {
			for _, inst := range reservation.Instances {
				instances = append(instances, toEC2Instance(inst))
			}
		}
	}

	return instances, nil
}

func toEC2Instance(inst ec2types.Instance) types.EC2Instance {
	out := types.EC2Instance{
		InstanceID:   awssdk.ToString(inst.InstanceId),
		InstanceType: string(inst.InstanceType),
		State:        string(inst.State.Name),
		PublicIP:     awssdk.ToString(inst.PublicIpAddress),
		PrivateIP:    awssdk.ToString(inst.PrivateIpAddress),
	}
	if inst.LaunchTime != nil {
		out.LaunchTime = inst.LaunchTime.UTC().Format(time.RFC3339)
	}
	for _, tag := range inst.Tags {
		if awssdk.ToString(tag.Key) == "Name" {
			out.Name = awssdk.ToString(tag.Value)
			break
		}
	}
	return out
}
