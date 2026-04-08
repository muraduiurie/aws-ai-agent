package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	corev1 "k8s.io/api/core/v1"

	"github.com/muraduiurie/aws-ai-agent/internal/kube"
	"github.com/muraduiurie/aws-ai-agent/pkg/types"
)

func RegisterKube(s *server.MCPServer, client kube.Client) {
	s.AddTool(listPodsDefinition(), listPodsHandler(client))
	s.AddTool(getPodDefinition(), getPodHandler(client))
	s.AddTool(createPodDefinition(), createPodHandler(client))
	s.AddTool(updatePodDefinition(), updatePodHandler(client))
	s.AddTool(deletePodDefinition(), deletePodHandler(client))
	s.AddTool(getPodLogsDefinition(), getPodLogsHandler(client))
}

// ── list_pods ────────────────────────────────────────────────────────────────

func listPodsDefinition() mcp.Tool {
	return mcp.NewTool(
		"list_pods",
		mcp.WithDescription("List pods in a Kubernetes namespace, optionally filtered by label selector."),
		mcp.WithString("namespace",
			mcp.Description("Kubernetes namespace to list pods in."),
			mcp.Required(),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter pods (e.g. app=nginx). Returns all pods if omitted."),
		),
	)
}

func listPodsHandler(client kube.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		labelSelector := req.GetString("label_selector", "")

		pods, err := client.ListPods(ctx, namespace, labelSelector)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list pods", err), nil
		}

		summaries := make([]types.PodSummary, len(pods))
		for i, p := range pods {
			summaries[i] = toPodSummary(p)
		}

		result, err := mcp.NewToolResultJSON(summaries)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize pods", err), nil
		}
		return result, nil
	}
}

// ── get_pod ──────────────────────────────────────────────────────────────────

func getPodDefinition() mcp.Tool {
	return mcp.NewTool(
		"get_pod",
		mcp.WithDescription("Get details of a specific Kubernetes pod."),
		mcp.WithString("namespace",
			mcp.Description("Namespace the pod belongs to."),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("Name of the pod."),
			mcp.Required(),
		),
	)
}

func getPodHandler(client kube.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		pod, err := client.GetPod(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get pod", err), nil
		}

		result, err := mcp.NewToolResultJSON(toPodSummary(*pod))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize pod", err), nil
		}
		return result, nil
	}
}

// ── create_pod ───────────────────────────────────────────────────────────────

func createPodDefinition() mcp.Tool {
	return mcp.NewTool(
		"create_pod",
		mcp.WithDescription("Create a Kubernetes pod from a JSON manifest."),
		mcp.WithString("namespace",
			mcp.Description("Namespace to create the pod in."),
			mcp.Required(),
		),
		mcp.WithString("manifest",
			mcp.Description("Full pod manifest as a JSON string (v1.Pod)."),
			mcp.Required(),
		),
	)
}

func createPodHandler(client kube.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		manifest, err := req.RequireString("manifest")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var pod corev1.Pod
		if err := json.Unmarshal([]byte(manifest), &pod); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid pod manifest: %v", err)), nil
		}

		created, err := client.CreatePod(ctx, namespace, &pod)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create pod", err), nil
		}

		result, err := mcp.NewToolResultJSON(toPodSummary(*created))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize pod", err), nil
		}
		return result, nil
	}
}

// ── update_pod ───────────────────────────────────────────────────────────────

func updatePodDefinition() mcp.Tool {
	return mcp.NewTool(
		"update_pod",
		mcp.WithDescription("Update an existing Kubernetes pod from a JSON manifest."),
		mcp.WithString("namespace",
			mcp.Description("Namespace the pod belongs to."),
			mcp.Required(),
		),
		mcp.WithString("manifest",
			mcp.Description("Full updated pod manifest as a JSON string (v1.Pod)."),
			mcp.Required(),
		),
	)
}

func updatePodHandler(client kube.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		manifest, err := req.RequireString("manifest")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var pod corev1.Pod
		if err := json.Unmarshal([]byte(manifest), &pod); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid pod manifest: %v", err)), nil
		}

		updated, err := client.UpdatePod(ctx, namespace, &pod)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update pod", err), nil
		}

		result, err := mcp.NewToolResultJSON(toPodSummary(*updated))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to serialize pod", err), nil
		}
		return result, nil
	}
}

// ── delete_pod ───────────────────────────────────────────────────────────────

func deletePodDefinition() mcp.Tool {
	return mcp.NewTool(
		"delete_pod",
		mcp.WithDescription("Delete a Kubernetes pod by name."),
		mcp.WithString("namespace",
			mcp.Description("Namespace the pod belongs to."),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("Name of the pod to delete."),
			mcp.Required(),
		),
	)
}

func deletePodHandler(client kube.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if err := client.DeletePod(ctx, namespace, name); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to delete pod", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("pod %s/%s deleted", namespace, name)), nil
	}
}

// ── get_pod_logs ─────────────────────────────────────────────────────────────

func getPodLogsDefinition() mcp.Tool {
	return mcp.NewTool(
		"get_pod_logs",
		mcp.WithDescription("Retrieve logs from a container inside a Kubernetes pod."),
		mcp.WithString("namespace",
			mcp.Description("Namespace the pod belongs to."),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("Name of the pod."),
			mcp.Required(),
		),
		mcp.WithString("container",
			mcp.Description("Container name. Required when the pod has more than one container."),
		),
		mcp.WithNumber("tail_lines",
			mcp.Description("Number of lines to return from the end of the logs. Returns all lines if omitted."),
		),
	)
}

func getPodLogsHandler(client kube.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		container := req.GetString("container", "")

		var tailLines *int64
		if n := req.GetInt("tail_lines", 0); n > 0 {
			v := int64(n)
			tailLines = &v
		}

		logs, err := client.GetPodLogs(ctx, namespace, name, container, tailLines)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get pod logs", err), nil
		}

		return mcp.NewToolResultText(logs), nil
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toPodSummary(pod corev1.Pod) types.PodSummary {
	summary := types.PodSummary{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Phase:     string(pod.Status.Phase),
		NodeName:  pod.Spec.NodeName,
		PodIP:     pod.Status.PodIP,
	}
	if !pod.CreationTimestamp.IsZero() {
		summary.CreationTimestamp = pod.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z")
	}

	statusByName := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, cs := range pod.Status.ContainerStatuses {
		statusByName[cs.Name] = cs
	}
	for _, c := range pod.Spec.Containers {
		info := types.ContainerInfo{Name: c.Name, Image: c.Image}
		if cs, ok := statusByName[c.Name]; ok {
			info.Ready = cs.Ready
		}
		summary.Containers = append(summary.Containers, info)
	}

	return summary
}
