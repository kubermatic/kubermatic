// +build e2e

/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcdlauncher

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	datacenter = "kubermatic"
	location   = "do-fra1"
	version    = utils.KubernetesVersion()
	credential = "e2e-digitalocean"
)

const (
	scaleUpCount   = 5
	scaleDownCount = 3
)

func TestScaling(t *testing.T) {
	ctx := context.Background()

	client, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// login
	masterToken, err := utils.RetrieveMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}
	testClient := utils.NewTestClient(masterToken, t)

	// create dummy project
	t.Log("creating project...")
	project, err := testClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	defer cleanupProject(t, project.ID)

	// create dummy cluster (NB: If these tests fail, the etcd ring can be
	// _so_ dead that any cleanup attempt is futile; make sure to not create
	// any cloud resources, as they might be orphaned)

	t.Log("creating cluster...")
	apiCluster, err := testClient.CreateDOCluster(project.ID, datacenter, rand.String(10), credential, version, location, 0)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	// wait for the cluster to become healthy
	if err := testClient.WaitForClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	// get the cluster object (the CRD, not the API's representation)
	cluster := &kubermaticv1.Cluster{}
	if err := client.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	// we run all these tests in the same cluster to speed up the e2e test
	if err := enableLauncher(ctx, t, client, cluster); err != nil {
		t.Fatalf("failed to enable etcd-launcher: %v", err)
	}
	waitForQuorum(t)

	if err := scaleUp(ctx, t, client, cluster); err != nil {
		t.Fatalf("failed to scale up: %v", err)
	}
	waitForQuorum(t)

	if err := scaleDown(ctx, t, client, cluster); err != nil {
		t.Fatalf("failed to scale down: %v", err)
	}
	waitForQuorum(t)

	if err := breakAndRecover(ctx, t, client, cluster); err != nil {
		t.Fatalf("failed to test volume recovery: %v", err)
	}
	waitForQuorum(t)

	if err := disableLauncher(ctx, t, client, cluster); err != nil {
		t.Fatalf("failed to disable etcd-launcher: %v", err)
	}

	t.Log("tests succeeded")
}

func enableLauncher(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	t.Log("enabling etcd-launcher...")
	if err := enableLauncherForCluster(ctx, client, cluster); err != nil {
		return fmt.Errorf("failed to enable etcd-launcher: %v", err)
	}

	if err := waitForClusterHealthy(ctx, t, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %v", err)
	}

	active, err := isEtcdLauncherActive(ctx, client, cluster)
	if err != nil {
		return fmt.Errorf("failed to check StatefulSet command: %v", err)
	}

	if !active {
		return errors.New("feature flag had no effect on the StatefulSet")
	}

	return nil
}

func disableLauncher(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	t.Log("disabling etcd-launcher...")
	if err := disableEtcdlauncherForCluster(ctx, client, cluster); err != nil {
		return fmt.Errorf("failed to disable etcd-launcher: %v", err)
	}

	if err := waitForClusterHealthy(ctx, t, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %v", err)
	}

	active, err := isEtcdLauncherActive(ctx, client, cluster)
	if err != nil {
		return fmt.Errorf("failed to check StatefulSet command: %v", err)
	}

	if active {
		return errors.New("removing the feature flag had no effect on the StatefulSet")
	}

	return nil
}

func scaleUp(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	t.Logf("scaling etcd cluster up to %d nodes...", scaleUpCount)
	if err := resizeEtcd(ctx, client, cluster, scaleUpCount); err != nil {
		return fmt.Errorf("failed while trying to scale up the etcd cluster: %v", err)
	}

	if err := waitForRollout(ctx, t, client, cluster, scaleUpCount); err != nil {
		return fmt.Errorf("rollout got stuck: %v", err)
	}
	t.Log("etcd cluster scaled up successfully.")

	return nil
}

func scaleDown(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	t.Logf("scaling etcd cluster down to %d nodes...", scaleDownCount)
	if err := resizeEtcd(ctx, client, cluster, scaleDownCount); err != nil {
		return fmt.Errorf("failed while trying to scale down the etcd cluster: %v", err)
	}

	if err := waitForRollout(ctx, t, client, cluster, scaleDownCount); err != nil {
		return fmt.Errorf("rollout got stuck: %v", err)
	}
	t.Log("etcd cluster scaled down successfully.")

	return nil
}

func breakAndRecover(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	// delete one of the etcd node PVs
	t.Log("testing etcd node PV automatic recovery...")
	if err := forceDeleteEtcdPV(ctx, client, cluster); err != nil {
		return fmt.Errorf("failed to delete etcd node PV: %v", err)
	}

	// auto recovery should kick in. We need to wait for it
	if err := waitForClusterHealthy(ctx, t, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %v", err)
	}
	t.Log("etcd node PV recovered successfully.")

	return nil
}

// enable etcd launcher for the cluster
func enableLauncherForCluster(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	return setClusterLauncherFeature(ctx, client, cluster, true)
}

func disableEtcdlauncherForCluster(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	return setClusterLauncherFeature(ctx, client, cluster, false)
}

func setClusterLauncherFeature(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, flag bool) error {
	return patchCluster(ctx, client, cluster, func(c *kubermaticv1.Cluster) error {
		if cluster.Spec.Features == nil {
			cluster.Spec.Features = map[string]bool{}
		}

		cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = flag
		return nil
	})
}

// isClusterEtcdHealthy checks whether the etcd status on the Cluster object
// is Healthy and the StatefulSet is fully rolled out.
func isClusterEtcdHealthy(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	// refresh cluster status
	if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return false, fmt.Errorf("failed to get cluster: %v", err)
	}

	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %v", err)
	}

	// we are healthy if the cluster controller is happy and the sts is ready
	return cluster.Status.ExtendedHealth.Etcd == kubermaticv1.HealthStatusUp &&
		*sts.Spec.Replicas == sts.Status.ReadyReplicas, nil
}

// isEtcdLauncherActive deduces from the StatefulSet's current spec whether or
// or not the etcd-launcher is enabled (and reconciled).
func isEtcdLauncherActive(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	etcdHealthy, err := isClusterEtcdHealthy(ctx, client, cluster)
	if err != nil {
		return false, fmt.Errorf("etcd health check failed: %v", err)
	}

	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %v", err)
	}

	return etcdHealthy && sts.Spec.Template.Spec.Containers[0].Command[0] == "/opt/bin/etcd-launcher", nil
}

// resizeEtcd changes the etcd cluster size.
func resizeEtcd(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, size int) error {
	if size > kubermaticv1.MaxEtcdClusterSize || size < kubermaticv1.MinEtcdClusterSize {
		return fmt.Errorf("Invalid etcd cluster size: %d", size)
	}

	return patchCluster(ctx, client, cluster, func(c *kubermaticv1.Cluster) error {
		n := int32(size)
		cluster.Spec.ComponentsOverride.Etcd.ClusterSize = &n
		return nil
	})
}

func waitForClusterHealthy(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	before := time.Now()

	if err := wait.PollImmediate(3*time.Second, 10*time.Minute, func() (bool, error) {
		healthy, err := isClusterEtcdHealthy(ctx, client, cluster)
		if err != nil {
			t.Logf("failed to check cluster etcd health status: %v", err)
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd health status: %v", err)
	}

	t.Logf("cluster became healthy after %v.", time.Since(before))

	return nil
}

func waitForRollout(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, targetSize int) error {
	t.Log("waiting for rollout...")

	if err := waitForClusterHealthy(ctx, t, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %v", err)
	}

	// count the pods
	readyPods, err := getStsReadyPodsCount(ctx, client, cluster)
	if err != nil {
		return fmt.Errorf("failed to check ready pods count: %v", err)
	}
	if int(readyPods) != targetSize {
		return fmt.Errorf("failed to scale etcd cluster: want [%d] nodes, got [%d]", targetSize, readyPods)
	}

	return nil
}

// TODO: Make this much smarter.
func waitForQuorum(t *testing.T) {
	t.Log("waiting for etcd to regain quorum...")
	time.Sleep(2 * time.Minute)
}

func forceDeleteEtcdPV(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	ns := clusterNamespace(cluster)

	selector, err := labels.Parse("app=etcd")
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %v", err)
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	opt := &ctrlruntimeclient.ListOptions{
		LabelSelector: selector,
		Namespace:     ns,
	}
	if err := client.List(ctx, pvcList, opt); err != nil || len(pvcList.Items) == 0 {
		return fmt.Errorf("failed to list PVCs or empty list in cluster namespace: %v", err)
	}

	// pick a random PVC, get its PV and delete it
	pvc := pvcList.Items[rand.Intn(len(pvcList.Items))]
	pvName := pvc.Spec.VolumeName
	typedName := types.NamespacedName{Name: pvName, Namespace: ns}

	pv := &corev1.PersistentVolume{}
	if err := client.Get(ctx, typedName, pv); err != nil {
		return fmt.Errorf("failed to get etcd node PV %s: %v", pvName, err)
	}
	oldPv := pv.DeepCopy()

	// first, we delete it
	if err := client.Delete(ctx, pv); err != nil {
		return fmt.Errorf("failed to delete etcd node PV %s: %v", pvName, err)
	}

	// now it will get stuck, we need to patch it to remove the pv finalizer
	pv.Finalizers = nil
	if err := client.Patch(ctx, pv, ctrlruntimeclient.MergeFrom(oldPv)); err != nil {
		return fmt.Errorf("failed to delete the PV %s finalizer: %v", pvName, err)
	}

	// make sure it's gone
	return wait.PollImmediate(2*time.Second, 3*time.Minute, func() (bool, error) {
		if err := client.Get(ctx, typedName, pv); kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

func getStsReadyPodsCount(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (int32, error) {
	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return 0, fmt.Errorf("failed to get StatefulSet: %v", err)
	}
	return sts.Status.ReadyReplicas, nil
}

func clusterNamespace(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s", cluster.Name)
}

type patchFunc func(cluster *kubermaticv1.Cluster) error

func patchCluster(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, patch patchFunc) error {
	if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %v", err)
	}

	oldCluster := cluster.DeepCopy()
	if err := patch(cluster); err != nil {
		return err
	}

	if err := client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch cluster: %v", err)
	}

	// give KKP some time to reconcile
	time.Sleep(10 * time.Second)

	return nil
}
