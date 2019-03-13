package main

import (
	"io/ioutil"
	"path"
	"testing"
	"time"

	clustercontroller "github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machine"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	maxRetryAttempts = 50
	retryWait        = 6 * time.Second
)

func setupCluster(ctx *TestContext, t *testing.T) {
	var err error

	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		if err = ctx.client.Create(ctx.ctx, ctx.cluster); err != nil {
			time.Sleep(retryWait)
			continue
		}

		t.Logf("Created the cluster '%s'", ctx.cluster.Name)
		return
	}

	t.Fatalf("failed to create the cluster object after %d attempts: %v", maxRetryAttempts, err)
}

func waitForControlPlane(ctx *TestContext, t *testing.T) {
	err := wait.PollImmediate(1*time.Second, 15*time.Minute, func() (bool, error) {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		cluster, err := ctx.clusterLister.Get(ctx.cluster.Name)
		if err != nil {
			return false, nil
		}
		ctx.cluster = cluster.DeepCopy()
		if ctx.cluster.Status.NamespaceName == "" {
			return false, nil
		}
		if !ctx.cluster.Status.Health.AllHealthy() {
			return false, nil
		}

		podList := &corev1.PodList{}
		if err := ctx.client.List(ctx.ctx, &ctrlruntimeclient.ListOptions{Namespace: ctx.cluster.Status.NamespaceName}, podList); err != nil {
			t.Logf("Failed to list pods from cluster: %v. Retrying...", err)
			return false, nil
		}

		for _, pod := range podList.Items {
			if !podIsReady(pod) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("failed waiting for control plane pods to become ready: %v", err)
	}
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

	config, err := clientcmd.BuildConfigFromFlags("", filename)
	if err != nil {
		t.Fatal(err)
	}

	dynamicClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("Failed to create dynamic client: %v", err)
	}

	ctx.clusterContext.client = dynamicClient
}

func setupNodes(ctx *TestContext, t *testing.T) {
	dc, found := ctx.dcs[ctx.cluster.Spec.Cloud.DatacenterName]
	if !found {
		t.Fatalf("Node datacenter '%s' not found in datacenters.yaml", ctx.cluster.Spec.Cloud.DatacenterName)
	}

	machineDeployment, err := machine.Deployment(ctx.cluster, ctx.nodeDeployment, dc, nil)
	if err != nil {
		t.Fatalf("failed to generate MachineDeployment object: %v", err)
	}
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		if err := ctx.clusterContext.client.Create(ctx.ctx, machineDeployment); err != nil {
			t.Logf("Failed to create MachineDeployment: %v. Retrying [%d/%d] ...", err, attempt, maxRetryAttempts)
			time.Sleep(retryWait)
			continue
		}
		return
	}
	t.Fatalf("failed to create MachineDeployment after %d attempts", maxRetryAttempts)
}

func waitForNodes(ctx *TestContext, t *testing.T) {
	err := wait.PollImmediate(1*time.Second, 30*time.Minute, func() (bool, error) {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		nodeList := &corev1.NodeList{}
		if err := ctx.clusterContext.client.List(ctx.ctx, &ctrlruntimeclient.ListOptions{}, nodeList); err != nil {
			t.Logf("Failed to list nodes from cluster: %v. Retrying...", err)
			return false, nil
		}
		if len(nodeList.Items) < int(ctx.nodeDeployment.Spec.Replicas) {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed waiting for nodes: %v", err)
	}
}

func setFinalizers(ctx *TestContext, t *testing.T) {
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		select {
		case <-ctx.ctx.Done():
			t.Fatal("Parent context is closed")
		default:
		}

		// Refresh the cluster to avoid errors
		cluster, err := ctx.clusterLister.Get(ctx.cluster.Name)
		if err != nil {
			t.Logf("Failed to get cluster from lister: %v. Retrying...", err)
			time.Sleep(retryWait)
			continue
		}

		finalizers := sets.NewString(cluster.Finalizers...)
		finalizers.Insert(clustercontroller.InClusterPVCleanupFinalizer)
		finalizers.Insert(clustercontroller.InClusterLBCleanupFinalizer)
		cluster.Finalizers = finalizers.List()

		if err = ctx.client.Update(ctx.ctx, cluster); err != nil {
			t.Logf("Failed to update cluster: %v. Retrying...", err)
			time.Sleep(retryWait)
			continue
		}

		ctx.cluster = cluster.DeepCopy()
		return
	}
	t.Fatalf("failed to add Finalizers to cluster after %d attempts", maxRetryAttempts)
}

func podIsReady(p corev1.Pod) bool {
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
		if err := ctx.clusterContext.client.List(ctx.ctx, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}, podList); err != nil {
			t.Logf("Failed to list pods from cluster: %v. Retrying...", err)
			return false, nil
		}

		for _, pod := range podList.Items {
			if !podIsReady(pod) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("failed waiting for pods in kube-system to become ready: %v", err)
	}
}
