package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/muraduiurie/aws-ai-agent/internal/aws"
)

func RegisterEC2(s *server.MCPServer, factory *aws.Factory) {
	s.AddTool(listEC2InstancesDefinition(), listEC2InstancesHandler(factory))
}

func listEC2InstancesDefinition() mcp.Tool {
	return mcp.NewTool(
		"list_ec2_instances",
		mcp.WithDescription("List EC2 instances in the configured AWS region. Optionally filter by state."),
		mcp.WithString("region",
			mcp.Description("AWS region to list instances in (e.g. us-east-1, eu-west-3)."),
			mcp.Required(),
		),
		mcp.WithString("state",
			mcp.Description("Filter instances by state (e.g. running, stopped, terminated). Returns all states if omitted."),
		),
	)
}

func listEC2InstancesHandler(factory *aws.Factory) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		region, err := req.RequireString("region")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		state := req.GetString("state", "")

		instances, err := factory.ListEC2Instances(ctx, region, state)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list EC2 instances", err), nil
		}

		result, err := mcp.NewToolResultJSON(instances)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize instances", err), nil
		}
		return result, nil
	}
}
