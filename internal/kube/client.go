package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is the interface for all Kubernetes operations exposed by this package.
type Client interface {
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	ListPods(ctx context.Context, namespace, labelSelector string) ([]corev1.Pod, error)
	CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error
	GetPodLogs(ctx context.Context, namespace, name, container string, tailLines *int64) (string, error)
}

// client is the concrete implementation of Client.
type client struct {
	clientset kubernetes.Interface
}

// NewClient builds a Kubernetes client.
// If kubeconfig is nil, in-cluster config is used (running inside a Pod).
// If kubeconfig is non-nil, the provided kubeconfig content is validated and
// used to connect — intended for kubeconfigs sourced from a Helm-managed Secret.
func NewClient(kubeconfig *string) (Client, error) {
	var cfg *rest.Config

	if kubeconfig == nil {
		var err error
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config: %w", err)
		}
	} else {
		var err error
		cfg, err = clientcmd.RESTConfigFromKubeConfig([]byte(*kubeconfig))
		if err != nil {
			return nil, fmt.Errorf("invalid kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	return &client{clientset: clientset}, nil
}
