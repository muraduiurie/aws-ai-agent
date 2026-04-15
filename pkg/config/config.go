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
	AWS AWSConfig `mapstructure:"-"` // populated from environment variables
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

// AWSConfig holds AWS connection parameters resolved from environment variables.
type AWSConfig struct {
	Profile  string
	Region   string
	ReadOnly bool
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

	cfg.AWS = loadAWSFromEnv()

	return &cfg, nil
}

// loadAWSFromEnv resolves AWS configuration from environment variables,
// mirroring the behaviour of the AWS SDK's own credential chain.
func loadAWSFromEnv() AWSConfig {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "eu-west-3"
	}
	return AWSConfig{
		Profile:  os.Getenv("AWS_PROFILE"),
		Region:   region,
		ReadOnly: os.Getenv("READ_ONLY") == "true",
	}
}
