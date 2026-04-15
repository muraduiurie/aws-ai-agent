package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/muraduiurie/aws-ai-agent/pkg/config"
)

// Factory holds the resolved AWS config and vends service clients.
type Factory struct {
	cfg aws.Config
}

// NewFactory loads AWS credentials and region from the environment / shared
// config files, applying any overrides from the server config.
func NewFactory(ctx context.Context, serverCfg *config.AWSConfig) (*Factory, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(serverCfg.Region),
	}
	if serverCfg.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(serverCfg.Profile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &Factory{cfg: cfg}, nil
}
