package mcp

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
	"github.com/muraduiurie/aws-ai-agent/internal/kube"
	"github.com/muraduiurie/aws-ai-agent/internal/tools"
)

const (
	serverName    = "aws-mcp-server"
	serverVersion = "0.1.0"
)

// Server bundles the MCP server with the AWS client factory and Kubernetes client holder.
type Server struct {
	MCP        *server.MCPServer
	AWSFactory *aws.Factory
	KubeHolder *kube.ClientHolder
}

// NewServer creates and configures the MCP server instance.
func NewServer(awsFactory *aws.Factory, kubeHolder *kube.ClientHolder) *Server {
	s := &Server{
		MCP: server.NewMCPServer(
			serverName,
			serverVersion,
			server.WithToolCapabilities(true),
		),
		AWSFactory: awsFactory,
		KubeHolder: kubeHolder,
	}
	s.registerTools()
	return s
}

// registerTools registers all tool groups with the MCP server.
func (s *Server) registerTools() {
	tools.RegisterEC2(s.MCP, s.AWSFactory)
	tools.RegisterEKS(s.MCP, s.AWSFactory)
	tools.RegisterKube(s.MCP, s.KubeHolder)
}

// Start begins listening for MCP messages over stdio.
func (s *Server) Start(ctx context.Context) error {
	stdioServer := server.NewStdioServer(s.MCP)
	stdioServer.SetErrorLogger(log.New(os.Stderr, "[mcp] ", log.LstdFlags))
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
