package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
)

func RegisterEKS(s *server.MCPServer, factory *aws.Factory) {
	s.AddTool(listEKSClustersDefinition(), listEKSClustersHandler(factory))
	s.AddTool(getEKSClusterDefinition(), getEKSClusterHandler(factory))
	s.AddTool(getEKSKubeconfigDefinition(), getEKSKubeconfigHandler(factory))
}

// ── list_eks_clusters ─────────────────────────────────────────────────────────

func listEKSClustersDefinition() mcp.Tool {
	return mcp.NewTool(
		"list_eks_clusters",
		mcp.WithDescription("List all EKS cluster names in an AWS region."),
		mcp.WithString("region",
			mcp.Description("AWS region to list clusters in (e.g. us-east-1, eu-west-3)."),
			mcp.Required(),
		),
	)
}

func listEKSClustersHandler(factory *aws.Factory) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		region, err := req.RequireString("region")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clusters, err := factory.ListEKSClusters(ctx, region)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list EKS clusters", err), nil
		}

		result, err := mcp.NewToolResultJSON(clusters)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize clusters", err), nil
		}
		return result, nil
	}
}

// ── get_eks_cluster ───────────────────────────────────────────────────────────

func getEKSClusterDefinition() mcp.Tool {
	return mcp.NewTool(
		"get_eks_cluster",
		mcp.WithDescription("Get details of a specific EKS cluster."),
		mcp.WithString("region",
			mcp.Description("AWS region the cluster resides in."),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("Name of the EKS cluster."),
			mcp.Required(),
		),
	)
}

func getEKSClusterHandler(factory *aws.Factory) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		region, err := req.RequireString("region")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		cluster, err := factory.GetEKSCluster(ctx, region, name)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get EKS cluster", err), nil
		}

		result, err := mcp.NewToolResultJSON(cluster)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize cluster", err), nil
		}
		return result, nil
	}
}

// ── get_eks_kubeconfig ────────────────────────────────────────────────────────

func getEKSKubeconfigDefinition() mcp.Tool {
	return mcp.NewTool(
		"get_eks_kubeconfig",
		mcp.WithDescription("Generate a kubeconfig for an EKS cluster. The returned string can be used directly to connect to the cluster."),
		mcp.WithString("region",
			mcp.Description("AWS region the cluster resides in."),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("Name of the EKS cluster."),
			mcp.Required(),
		),
	)
}

func getEKSKubeconfigHandler(factory *aws.Factory) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		region, err := req.RequireString("region")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		kubeconfig, err := factory.GetEKSKubeconfig(ctx, region, name)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get EKS kubeconfig", err), nil
		}

		return mcp.NewToolResultText(kubeconfig), nil
	}
}
