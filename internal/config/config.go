package config

import "os"

// Config holds server-level configuration resolved at startup.
type Config struct {
	AWSProfile string
	AWSRegion  string
	ReadOnly   bool
}

// Load builds a Config from environment variables with sensible defaults.
func Load() *Config {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "eu-west-3"
	}
	return &Config{
		AWSProfile: os.Getenv("AWS_PROFILE"),
		AWSRegion:  region,
		ReadOnly:   os.Getenv("READ_ONLY") == "true",
	}
}
