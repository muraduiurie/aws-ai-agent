package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
	"github.com/muraduiurie/aws-ai-agent/internal/config"
	"github.com/muraduiurie/aws-ai-agent/internal/kube"
	"github.com/muraduiurie/aws-ai-agent/internal/mcp"
)

const kubeconfigPathEnv = "KUBECONFIG_PATH"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	awsFactory, err := aws.NewFactory(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize AWS client factory: %v", err)
	}

	var kubeconfig *string
	if path := os.Getenv(kubeconfigPathEnv); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("failed to read kubeconfig from %s: %v", path, err)
		}
		content := string(data)
		kubeconfig = &content
	}

	kubeClient, err := kube.NewClient(kubeconfig)
	if err != nil {
		log.Fatalf("failed to initialize Kubernetes client: %v", err)
	}

	s := mcp.NewServer(awsFactory, kubeClient)

	log.Printf("starting %s (region=%s, read-only=%v)", "aws-mcp-server", cfg.AWSRegion, cfg.ReadOnly)

	if err := s.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
