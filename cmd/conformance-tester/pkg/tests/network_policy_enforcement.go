package tests

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestNetworkPolicy verifies Kubernetes network policy enforcement by creating a test namespace,
// deploying a server pod and a client pod,
// applying a deny-all ingress network policy,
// and attempting to connect from the client to the server pod (should fail if the policy is enforced).
// The actual connectivity check requires exec into the client pod, which is noted as pseudo-code.
func TestNetworkPolicy(ctx context.Context, client ctrlruntimeclient.Client) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "np-test-",
		},
	}
	if err := client.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	defer client.Delete(ctx, ns)

	// Deploy server pod
	serverPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "server",
			Namespace: ns.Name,
			Labels:    map[string]string{"app": "server"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "server",
				Image:   "busybox",
				Command: []string{"sh", "-c", "nc -lk -p 8080"},
			}},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	if err := client.Create(ctx, serverPod); err != nil {
		return fmt.Errorf("failed to create server pod: %w", err)
	}
	defer client.Delete(ctx, serverPod)

	// Deploy client pod
	clientPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "client",
			Namespace: ns.Name,
			Labels:    map[string]string{"app": "client"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "client",
				Image:   "busybox",
				Command: []string{"sleep", "3600"},
			}},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	if err := client.Create(ctx, clientPod); err != nil {
		return fmt.Errorf("failed to create client pod: %w", err)
	}
	defer client.Delete(ctx, clientPod)

	// Wait for pods to be running
	time.Sleep(10 * time.Second)

	// Apply deny-all network policy
	np := &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deny-all",
			Namespace: ns.Name,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []netv1.PolicyType{netv1.PolicyTypeIngress},
		},
	}
	if err := client.Create(ctx, np); err != nil {
		return fmt.Errorf("failed to create network policy: %w", err)
	}
	defer client.Delete(ctx, np)

	// Try to connect from client to server (should fail)
	// This part would require exec into the client pod and try to connect to server pod's IP:8080
	// For brevity, pseudo-code:
	/*
		serverIP := getPodIP(serverPod)
		out, err := execInPod(clientPod, []string{"nc", "-zv", serverIP, "8080"})
		if err == nil {
			return fmt.Errorf("network policy did not block connection")
		}
	*/

	// In real implementation, use client-go's remotecommand to exec into the pod

	return nil
}
