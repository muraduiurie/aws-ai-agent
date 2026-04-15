package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/muraduiurie/aws-ai-agent/internal/kube"
)

// invoke calls a ToolHandlerFunc with a constructed request.
func invoke(h server.ToolHandlerFunc, args map[string]any) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return h(context.Background(), req)
}

func textOf(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func holderReady(h *kube.ClientHolder) bool {
	_, ok := h.Get()
	return ok
}

// minimalKubeconfig returns a minimal valid kubeconfig YAML string.
func minimalKubeconfig(clusterName, serverURL, token string) string {
	return `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: ` + serverURL + `
  name: ` + clusterName + `
contexts:
- context:
    cluster: ` + clusterName + `
    user: admin
  name: ` + clusterName + `-context
current-context: ` + clusterName + `-context
users:
- name: admin
  user:
    token: ` + token + `
`
}

// ── set_kubeconfig: argument validation ──────────────────────────────────────

func TestSetKubeconfig_MissingArgument(t *testing.T) {
	var h kube.ClientHolder
	result, err := invoke(setKubeconfigHandler(&h), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error when kubeconfig argument is missing")
	}
}

func TestSetKubeconfig_InvalidKubeconfig(t *testing.T) {
	var h kube.ClientHolder
	result, err := invoke(setKubeconfigHandler(&h), map[string]any{
		"kubeconfig": "not-valid-yaml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for invalid kubeconfig content")
	}
	if holderReady(&h) {
		t.Fatal("holder should remain uninitialized after failed set_kubeconfig")
	}
}

// ── set_kubeconfig: single cluster connection ─────────────────────────────────

func TestSetKubeconfig_ConnectToClusterA(t *testing.T) {
	var h kube.ClientHolder
	cfg := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")

	result, err := invoke(setKubeconfigHandler(&h), map[string]any{"kubeconfig": cfg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got: %s", textOf(result))
	}
	if msg := textOf(result); !strings.Contains(msg, "initialized") {
		t.Fatalf("unexpected success message: %q", msg)
	}
	if !holderReady(&h) {
		t.Fatal("holder should be initialized after successful set_kubeconfig")
	}
}

// ── set_kubeconfig: switching between clusters ────────────────────────────────

func TestSetKubeconfig_SwitchFromClusterAToClusterB(t *testing.T) {
	var h kube.ClientHolder

	cfgA := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")
	cfgB := minimalKubeconfig("cluster-b", "https://cluster-b.example.com:6443", "token-b")

	// Connect to cluster A.
	result, err := invoke(setKubeconfigHandler(&h), map[string]any{"kubeconfig": cfgA})
	if err != nil || result.IsError {
		t.Fatalf("failed to connect to cluster-a: %v / %s", err, textOf(result))
	}
	clientA, _ := h.Get()

	// Switch to cluster B.
	result, err = invoke(setKubeconfigHandler(&h), map[string]any{"kubeconfig": cfgB})
	if err != nil || result.IsError {
		t.Fatalf("failed to connect to cluster-b: %v / %s", err, textOf(result))
	}
	clientB, _ := h.Get()

	if clientA == clientB {
		t.Fatal("expected a new client instance after switching to cluster-b")
	}
}

func TestSetKubeconfig_SwitchAcrossMultipleClusters(t *testing.T) {
	var h kube.ClientHolder

	clusters := []struct{ name, server, token string }{
		{"cluster-a", "https://cluster-a.example.com:6443", "token-a"},
		{"cluster-b", "https://cluster-b.example.com:6443", "token-b"},
		{"cluster-a", "https://cluster-a.example.com:6443", "token-a"}, // reconnect to A
	}

	for _, c := range clusters {
		cfg := minimalKubeconfig(c.name, c.server, c.token)
		result, err := invoke(setKubeconfigHandler(&h), map[string]any{"kubeconfig": cfg})
		if err != nil || result.IsError {
			t.Fatalf("failed to connect to %s: %v / %s", c.name, err, textOf(result))
		}
		if !holderReady(&h) {
			t.Fatalf("holder should be initialized after connecting to %s", c.name)
		}
	}
}

func TestSetKubeconfig_InvalidKubeconfigPreservesExistingClient(t *testing.T) {
	var h kube.ClientHolder
	cfgA := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")

	// Connect to cluster A.
	result, err := invoke(setKubeconfigHandler(&h), map[string]any{"kubeconfig": cfgA})
	if err != nil || result.IsError {
		t.Fatalf("initial connect failed: %v / %s", err, textOf(result))
	}
	clientBefore, _ := h.Get()

	// Attempt a bad kubeconfig — should fail without touching the existing client.
	result, err = invoke(setKubeconfigHandler(&h), map[string]any{"kubeconfig": "garbage"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for invalid kubeconfig")
	}

	clientAfter, ok := h.Get()
	if !ok || clientAfter == nil {
		t.Fatal("existing client should be preserved after a failed set_kubeconfig")
	}
	if clientBefore != clientAfter {
		t.Fatal("client instance should not change after a failed set_kubeconfig")
	}
}

// ── kube tool guard: not initialized ─────────────────────────────────────────

func TestListPods_NotInitialized(t *testing.T) {
	var h kube.ClientHolder
	result, err := invoke(listPodsHandler(&h), map[string]any{"namespace": "default"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error when Kubernetes client is not initialized")
	}
	if msg := textOf(result); !strings.Contains(msg, "set_kubeconfig") {
		t.Fatalf("error message should mention set_kubeconfig, got: %q", msg)
	}
}
