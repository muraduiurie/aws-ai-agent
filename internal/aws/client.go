package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/muraduiurie/aws-ai-agent/internal/config"
)

// Factory holds the resolved AWS config and vends service clients.
type Factory struct {
	cfg aws.Config
}

// NewFactory loads AWS credentials and region from the environment / shared
// config files, applying any overrides from the server config.
func NewFactory(ctx context.Context, serverCfg *config.Config) (*Factory, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(serverCfg.AWSRegion),
	}
	if serverCfg.AWSProfile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(serverCfg.AWSProfile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &Factory{cfg: cfg}, nil
}
