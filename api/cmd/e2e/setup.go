package main

import (
	"context"
	"io/ioutil"
	"path"
	"testing"
	"time"

	clustercontroller "github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machine"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

func setupCluster(ctx *TestContext, t *testing.T) {
	var (
		cluster *kubermaticv1.Cluster
		err     error
	)

	for attempt := 1; attempt <= 100; attempt++ {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		cluster, err = ctx.kubermaticClient.KubermaticV1().Clusters().Create(ctx.cluster)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		t.Logf("Created the cluster '%s'", cluster.Name)
		ctx.cluster = cluster.DeepCopy()
		return
	}

	t.Fatalf("failed to create the cluster object after 100 attempts: %v", err)
}

func waitForControlPlane(ctx *TestContext, t *testing.T) {
	err := wait.PollImmediate(1*time.Second, ctx.controlPlaneWaitTimeout, func() (done bool, err error) {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		cluster, err := ctx.clusterLister.Get(ctx.cluster.Name)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, nil
		}
		return cluster.Status.Health.AllHealthy(), nil
	})
	// Timeout or other error
	if err != nil {
		t.Fatalf("failed waiting for the control plane to become ready: %v", err)
	}

	cluster, err := ctx.clusterLister.Get(ctx.cluster.Name)
	if err != nil {
		t.Fatalf("failed to get cluster from context: %v", err)
	}

	ctx.cluster = cluster.DeepCopy()
}

func setupClusterContext(ctx *TestContext, t *testing.T) {
	kubeconfig, err := ctx.clusterClientProvider.GetAdminKubeconfig(ctx.cluster)
	if err != nil {
		t.Fatalf("failed to load kubeconfig from cluster client provider: %v", err)
	}

	filename := path.Join(ctx.workingDir, "kubeconfig")
	if err := ioutil.WriteFile(filename, kubeconfig, 0644); err != nil {
		t.Fatalf("failed to create a temporary file to store the kubeconfig")
	}

	ctx.clusterContext.kubeconfig = filename
	t.Logf("Wrote kubeconfig to %s", filename)

	client, err := ctx.clusterClientProvider.GetDynamicClient(ctx.cluster)
	if err != nil {
		t.Fatal(err)
	}
	ctx.clusterContext.client = client
}

func setupNodes(ctx *TestContext, t *testing.T) {
	dc, found := ctx.dcs[ctx.cluster.Spec.Cloud.DatacenterName]
	if !found {
		t.Fatalf("Node datacenter '%s' not found in datacenters.yaml", ctx.cluster.Spec.Cloud.DatacenterName)
	}

	machineDeployment, err := machine.Deployment(ctx.cluster, ctx.nodeDeployment, dc, nil)
	if err != nil {
		t.Fatalf("failed to generate Machine object: %v", err)
	}
	for attempt := 1; attempt <= 100; attempt++ {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		if err := ctx.clusterContext.client.Create(context.Background(), machineDeployment); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		return
	}
	t.Fatalf("failed to create Machine after 100 attempts")
}

func waitForNodes(ctx *TestContext, t *testing.T) {
	err := wait.PollImmediate(1*time.Second, 15*time.Minute, func() (bool, error) {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		nodeList := &corev1.NodeList{}
		if err := ctx.clusterContext.client.List(context.Background(), nil, nodeList); err != nil {
			return false, nil
		}
		if len(nodeList.Items) < ctx.nodeCount {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed waiting for nodes: %v", err)
	}
}

func setFinalizers(ctx *TestContext, t *testing.T) {
	for attempt := 1; attempt <= 100; attempt++ {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		// Refresh the cluster to avoid errors
		cluster, err := ctx.clusterLister.Get(ctx.cluster.Name)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		finalizers := sets.NewString(cluster.Finalizers...)
		finalizers.Insert(clustercontroller.InClusterPVCleanupFinalizer)
		finalizers.Insert(clustercontroller.InClusterLBCleanupFinalizer)
		cluster.Finalizers = finalizers.List()

		cluster, err = ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		ctx.cluster = cluster.DeepCopy()
		return
	}
	t.Fatalf("failed to create Machine after 100 attempts")
}

func podIsReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func waitForAllSystemPods(ctx *TestContext, t *testing.T) {
	err := wait.PollImmediate(1*time.Second, 15*time.Minute, func() (bool, error) {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		podList := &corev1.PodList{}
		if err := ctx.clusterContext.client.List(context.Background(), nil, podList); err != nil {
			return false, nil
		}

		for _, pod := range podList.Items {
			if !podIsReady(&pod) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("failed waiting for pods in kube-system to become ready: %v", err)
	}
}
