package agent

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/muraduiurie/aws-ai-agent/pkg/config"
)

type claudeAgent struct {
	client   anthropic.Client
	model    anthropic.Model
	maxToken int64
}

// NewAgent creates a new Claude-backed Agent from the Claude section of the
// application config.
func NewAgent(cfg *config.ClaudeConfig) (Agent, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("claude token must not be empty")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("claude model must not be empty")
	}
	return &claudeAgent{
		client:   anthropic.NewClient(option.WithAPIKey(cfg.Token)),
		model:    cfg.Model,
		maxToken: int64(cfg.MaxToken),
	}, nil
}

// Ping sends a minimal "ping" message to the model and returns its response
// text. Use it to verify that the API key is valid and the connection works.
func (a *claudeAgent) Ping(ctx context.Context) (string, error) {
	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: 16,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("ping")),
		},
	})
	if err != nil {
		return "", fmt.Errorf("ping: %w", err)
	}
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("ping: empty response from model")
	}
	return msg.Content[0].Text, nil
}
