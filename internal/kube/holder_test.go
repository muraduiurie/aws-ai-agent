package kube_test

import (
	"sync"
	"testing"

	"github.com/muraduiurie/aws-ai-agent/internal/kube"
)

// minimalKubeconfig returns a minimal valid kubeconfig YAML for a named cluster/server.
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

func TestClientHolder_GetBeforeSet(t *testing.T) {
	var h kube.ClientHolder
	client, ok := h.Get()
	if ok {
		t.Fatal("expected ok=false on uninitialized holder")
	}
	if client != nil {
		t.Fatal("expected nil client on uninitialized holder")
	}
}

func TestClientHolder_SetInvalidKubeconfig(t *testing.T) {
	var h kube.ClientHolder
	err := h.Set("not-valid-kubeconfig-content")
	if err == nil {
		t.Fatal("expected error for invalid kubeconfig")
	}
	_, ok := h.Get()
	if ok {
		t.Fatal("holder should remain uninitialized after failed Set")
	}
}

func TestClientHolder_SetValidKubeconfig_ClusterA(t *testing.T) {
	var h kube.ClientHolder
	cfg := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")

	if err := h.Set(cfg); err != nil {
		t.Fatalf("Set cluster-a: %v", err)
	}

	client, ok := h.Get()
	if !ok {
		t.Fatal("expected ok=true after Set")
	}
	if client == nil {
		t.Fatal("expected non-nil client after Set")
	}
}

func TestClientHolder_SwitchBetweenClusters(t *testing.T) {
	var h kube.ClientHolder

	cfgA := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")
	cfgB := minimalKubeconfig("cluster-b", "https://cluster-b.example.com:6443", "token-b")

	// Connect to cluster A.
	if err := h.Set(cfgA); err != nil {
		t.Fatalf("Set cluster-a: %v", err)
	}
	clientA, ok := h.Get()
	if !ok || clientA == nil {
		t.Fatal("expected initialized client after Set cluster-a")
	}

	// Switch to cluster B.
	if err := h.Set(cfgB); err != nil {
		t.Fatalf("Set cluster-b: %v", err)
	}
	clientB, ok := h.Get()
	if !ok || clientB == nil {
		t.Fatal("expected initialized client after Set cluster-b")
	}

	// The holder should now point to cluster B (different client instance).
	if clientA == clientB {
		t.Fatal("expected different client instances after switching clusters")
	}
}

func TestClientHolder_SwitchBackToClusterA(t *testing.T) {
	var h kube.ClientHolder

	cfgA := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")
	cfgB := minimalKubeconfig("cluster-b", "https://cluster-b.example.com:6443", "token-b")

	// A → B → A
	for _, cfg := range []string{cfgA, cfgB, cfgA} {
		if err := h.Set(cfg); err != nil {
			t.Fatalf("Set: %v", err)
		}
		_, ok := h.Get()
		if !ok {
			t.Fatal("holder should be initialized after each Set")
		}
	}
}

func TestClientHolder_InvalidSetDoesNotClearExistingClient(t *testing.T) {
	var h kube.ClientHolder

	cfgA := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")
	if err := h.Set(cfgA); err != nil {
		t.Fatalf("Set cluster-a: %v", err)
	}

	// Attempt to set an invalid kubeconfig.
	if err := h.Set("garbage"); err == nil {
		t.Fatal("expected error for invalid kubeconfig")
	}

	// Original client must still be present.
	client, ok := h.Get()
	if !ok || client == nil {
		t.Fatal("existing client should be preserved after a failed Set")
	}
}

func TestClientHolder_ConcurrentAccess(t *testing.T) {
	var h kube.ClientHolder
	cfg := minimalKubeconfig("cluster-a", "https://cluster-a.example.com:6443", "token-a")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.Set(cfg)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Get()
		}()
	}
	wg.Wait()
}
