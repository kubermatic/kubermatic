package tests

import (
	"context"
	"fmt"
	"k8c.io/kubermatic/v2/pkg/resources"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestPodDisruptionBudget verifies PDB enforcement in a Kubernetes cluster.
func TestPodDisruptionBudget(ctx context.Context, userClusterClient ctrlruntimeclient.Client) error {
	const (
		namespace  = "pdb-test"
		replicas   = 3
		pauseImage = resources.RegistryK8S + "/pause:3.9"
	)

	// 1. Ensure namespace exists
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	err := userClusterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: namespace}, ns)
	if errors.IsNotFound(err) {
		if err := userClusterClient.Create(ctx, ns); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// 2. Create deployment
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pdb-demo",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "pdb-demo"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "pdb-demo"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "pause",
						Image: pauseImage,
					}},
				},
			},
		},
	}
	if err := userClusterClient.Create(ctx, deploy); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// 3. Create PDB with minAvailable = replicas - 1
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pdb-demo",
			Namespace: namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(replicas - 1),
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "pdb-demo"},
			},
		},
	}
	if err := userClusterClient.Create(ctx, pdb); err != nil {
		return fmt.Errorf("failed to create PDB: %w", err)
	}

	// 4. Wait for pods to be ready
	if err := waitForPodsReady(ctx, userClusterClient, namespace, "app=pdb-demo", replicas, 2*time.Minute); err != nil {
		return fmt.Errorf("pods not ready: %w", err)
	}

	// 5. Try to evict more pods than allowed by PDB
	podList := &corev1.PodList{}
	if err := userClusterClient.List(ctx, podList, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabels{"app": "pdb-demo"}); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	evictCount := 2 // Try to evict 2 pods, should fail due to minAvailable
	evicted := 0
	for i := 0; i < evictCount; i++ {
		pod := podList.Items[i]
		eviction := &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: namespace,
			},
		}
		err := userClusterClient.SubResource("eviction").Create(ctx, &pod, eviction)
		if err != nil {
			// Expect error on second eviction due to PDB
			if evicted == 1 {
				return nil // Success: PDB prevented disruption
			}
			return fmt.Errorf("unexpected eviction error: %w", err)
		}
		evicted++
	}

	return fmt.Errorf("PDB did not prevent excessive pod disruption")
}

func int32Ptr(i int) *int32 { v := int32(i); return &v }

// Waits for the specified number of pods to be ready in the namespace.
func waitForPodsReady(ctx context.Context, client ctrlruntimeclient.Client, namespace, label string, count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		podList := &corev1.PodList{}
		err := client.List(ctx, podList, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabels{"app": "pdb-demo"})
		if err != nil {
			return err
		}
		ready := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodRunning {
				for _, cs := range pod.Status.ContainerStatuses {
					if cs.Ready {
						ready++
						break
					}
				}
			}
		}
		if ready >= count {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for pods to be ready")
}
