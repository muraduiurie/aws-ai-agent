package mcp

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
	"github.com/muraduiurie/aws-ai-agent/internal/tools"
)

const (
	serverName    = "aws-mcp-server"
	serverVersion = "0.1.0"
)

// Server bundles the MCP server with the AWS client factory.
type Server struct {
	MCP     *server.MCPServer
	Factory *aws.Factory
}

// NewServer creates and configures the MCP server instance.
func NewServer(factory *aws.Factory) *Server {
	s := &Server{
		MCP: server.NewMCPServer(
			serverName,
			serverVersion,
			server.WithToolCapabilities(true),
		),
		Factory: factory,
	}
	s.registerTools()
	return s
}

// registerTools registers all tool groups with the MCP server.
func (s *Server) registerTools() {
	tools.RegisterEC2(s.MCP, s.Factory)
}

// Start begins listening for MCP messages over stdio.
func (s *Server) Start(ctx context.Context) error {
	stdioServer := server.NewStdioServer(s.MCP)
	stdioServer.SetErrorLogger(log.New(os.Stderr, "[mcp] ", log.LstdFlags))
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
