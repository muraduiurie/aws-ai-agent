package kube

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPod returns a single pod by namespace and name.
func (c *client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// ListPods returns all pods in the given namespace, optionally filtered by a
// label selector (e.g. "app=nginx"). Pass an empty string to list all pods.
func (c *client) ListPods(ctx context.Context, namespace, labelSelector string) ([]corev1.Pod, error) {
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	list, err := c.clientset.CoreV1().Pods(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("list pods in %s: %w", namespace, err)
	}
	return list.Items, nil
}

// CreatePod creates a pod in the given namespace and returns the created object.
func (c *client) CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	created, err := c.clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create pod in %s: %w", namespace, err)
	}
	return created, nil
}

// UpdatePod performs a full update of an existing pod and returns the updated object.
func (c *client) UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	updated, err := c.clientset.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("update pod %s/%s: %w", namespace, pod.Name, err)
	}
	return updated, nil
}

// DeletePod deletes the pod with the given name from the given namespace.
func (c *client) DeletePod(ctx context.Context, namespace, name string) error {
	if err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetPodLogs returns the logs of a container inside a pod as a string.
// Specify container name when the pod has more than one container.
// Pass a non-nil tailLines to limit the number of lines returned.
func (c *client) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines *int64) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: container,
		TailLines: tailLines,
	}

	stream, err := c.clientset.CoreV1().Pods(namespace).GetLogs(name, opts).Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("stream logs for pod %s/%s: %w", namespace, name, err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("read logs for pod %s/%s: %w", namespace, name, err)
	}

	return string(data), nil
}
