package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func printSeedPods(ctx *TestContext, t *testing.T) {
	if !t.Failed() {
		return
	}

	ns := ctx.cluster.Status.NamespaceName
	podList := &corev1.PodList{}
	if err := ctx.client.List(ctx.ctx, &ctrlruntimeclient.ListOptions{Namespace: ns}, podList); err != nil {
		t.Logf("failed to list pods in namespace '%s': %v", ns, err)
		return
	}

	t.Logf("===== Control plane pods =====")
	for _, pod := range podList.Items {
		t.Logf("[%s/%s] Phase: %s", ns, pod.Name, pod.Status.Phase)
		for _, containerStatus := range pod.Status.ContainerStatuses {
			t.Logf("[%s/%s/%s] Restarts: %d", ns, pod.Name, containerStatus.Name, containerStatus.RestartCount)
		}
	}
}
