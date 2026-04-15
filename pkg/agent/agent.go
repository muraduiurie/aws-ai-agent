package agent

import "context"

// Agent is the interface for AI agent operations.
type Agent interface {
	// Ping sends a minimal request to the model and returns the response text.
	// Use it to verify that the API key is valid and the connection is working.
	Ping(ctx context.Context) (string, error)
}
