package kube

import "sync"

// ClientHolder holds a lazily-initialized Kubernetes client.
// It is safe for concurrent use.
type ClientHolder struct {
	mu     sync.RWMutex
	client Client
}

// Set initializes the Kubernetes client from the provided kubeconfig content string.
func (h *ClientHolder) Set(kubeconfig string) error {
	c, err := NewClient(&kubeconfig)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.client = c
	h.mu.Unlock()
	return nil
}

// Get returns the current client and whether it has been initialized.
func (h *ClientHolder) Get() (Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client, h.client != nil
}
