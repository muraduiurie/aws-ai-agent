package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
	"github.com/muraduiurie/aws-ai-agent/internal/config"
	"github.com/muraduiurie/aws-ai-agent/internal/mcp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	awsFactory, err := aws.NewFactory(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize AWS client factory: %v", err)
	}

	s := mcp.NewServer(awsFactory)

	log.Printf("starting %s (region=%s, read-only=%v)", "aws-mcp-server", cfg.AWSRegion, cfg.ReadOnly)

	if err := s.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
