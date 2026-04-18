package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	configPathEnv     = "CONFIG_PATH"
	defaultConfigPath = "/etc/aws-ai-agent/config.yaml"
)

// Config is the top-level configuration structure.
type Config struct {
	AI  AIConfig  `mapstructure:"ai"`
	AWS AWSConfig `mapstructure:"aws"` // base from YAML; env vars override
}

// AIConfig groups all AI provider configurations.
type AIConfig struct {
	Claude ClaudeConfig `mapstructure:"claude"`
}

// ClaudeConfig holds the Anthropic Claude connection parameters.
type ClaudeConfig struct {
	Token    string `mapstructure:"token"`
	MaxToken int    `mapstructure:"maxToken"`
	Model    string `mapstructure:"model"`
}

// AWSConfig holds AWS connection parameters.
// Values are read from the YAML file first; environment variables override them.
type AWSConfig struct {
	Profile string `mapstructure:"profile"`
	Region  string `mapstructure:"region"`

	// ReadOnly, when true, is intended to prevent any tool that mutates AWS or
	// Kubernetes resources from executing. It is loaded and logged at startup
	// but is NOT YET ENFORCED — no handler checks it today.
	//
	// Before adding any mutating tool (e.g. start/stop EC2 instances,
	// terminate nodes, create/update/delete Kubernetes resources), enforce this
	// flag in registerTools() in internal/mcp/server.go or at the individual
	// handler level in internal/tools/.
	ReadOnly bool `mapstructure:"readOnly"`
}

// Load reads the YAML configuration file from the path specified by
// CONFIG_PATH (defaults to /etc/aws-ai-agent/config.yaml), unmarshals it,
// and populates the AWS section from environment variables.
func Load() (*Config, error) {
	path := os.Getenv(configPathEnv)
	if path == "" {
		path = defaultConfigPath
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.AWS = mergeAWSEnvVars(cfg.AWS)

	return &cfg, nil
}

// mergeAWSEnvVars overrides YAML-sourced AWS values with environment variables
// where the variables are explicitly set. The precedence is: env var > YAML > default.
func mergeAWSEnvVars(base AWSConfig) AWSConfig {
	if v := os.Getenv("AWS_REGION"); v != "" {
		base.Region = v
	} else if v := os.Getenv("AWS_DEFAULT_REGION"); v != "" {
		base.Region = v
	}
	if base.Region == "" {
		base.Region = "eu-west-3"
	}
	if v := os.Getenv("AWS_PROFILE"); v != "" {
		base.Profile = v
	}
	if os.Getenv("READ_ONLY") == "true" {
		base.ReadOnly = true
	}
	return base
}
