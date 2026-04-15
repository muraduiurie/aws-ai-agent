package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
	"github.com/muraduiurie/aws-ai-agent/internal/kube"
	"github.com/muraduiurie/aws-ai-agent/internal/mcp"
	"github.com/muraduiurie/aws-ai-agent/pkg/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	awsFactory, err := aws.NewFactory(ctx, &cfg.AWS)
	if err != nil {
		log.Fatalf("failed to initialize AWS client factory: %v", err)
	}

	s := mcp.NewServer(awsFactory, &kube.ClientHolder{})

	log.Printf("starting %s (region=%s, read-only=%v)", "aws-mcp-server", cfg.AWS.Region, cfg.AWS.ReadOnly)

	if err := s.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
