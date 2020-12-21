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

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	scaleUpCount       = 5
	scaleDownCount     = 3
	healthCheckTimeout = 10
)

func (r *testRunner) testEtcdLauncher(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// So far, the etcdlauncher is experimental. We need to enable it first
	log.Info("Testing etcd-launcher experimental features...")
	log.Info("Enabling etcd-launcher...")
	if err := r.enableEtcdlauncherForCluster(ctx, cluster); err != nil {
		return fmt.Errorf("failed to enable etcd-launcher: %v", zap.Error(err))
	}

	if err := wait.Poll(30*time.Second, healthCheckTimeout*time.Minute, func() (bool, error) {
		active, err := r.isEtcdLauncherActive(ctx, cluster)
		if err != nil {
			log.Warnf("failed to check etcd-launcher status: %v", zap.Error(err))
			return false, nil
		}
		return active, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd-launcher enabled status: %v", zap.Error(err))
	}
	log.Info("etcd-launcher enabled successfully...")

	log.Info("waiting for etcd to regain quorum...")
	time.Sleep(2 * time.Minute)

	// scale up to 5 nodes
	log.Infof("Testing etcd cluster scale up: scaling etcd cluster to %d nodes...", scaleUpCount)
	if err := r.resizeEtcd(ctx, cluster, scaleUpCount); err != nil {
		return fmt.Errorf("failed while trying to scale up the etcd cluster: %v", zap.Error(err))
	}
	if err := wait.Poll(30*time.Second, healthCheckTimeout*time.Minute, func() (bool, error) {
		healthy, err := r.isClusterEtcdHealthy(ctx, cluster)
		if err != nil {
			log.Warnf("failed to check cluster etcd health status: %v", zap.Error(err))
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd cluster scale up status: %v", zap.Error(err))
	}
	// count the pods!
	readyPods, err := r.getStsReadyPodsCount(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to check ready pods count: %v", zap.Error(err))
	}
	if readyPods != scaleUpCount {
		return fmt.Errorf("failed to scale up etcd cluster: want [%d] nodes, got [%d]", scaleUpCount, readyPods)
	}
	log.Info("etcd cluster scaled up successfully...")

	log.Info("waiting for etcd to regain quorum...")
	time.Sleep(2 * time.Minute)

	// scale back to 3 nodes
	log.Infof("Testing etcd cluster scale down: scaling etcd cluster to %d nodes...", scaleDownCount)
	if err := r.resizeEtcd(ctx, cluster, scaleDownCount); err != nil {
		return fmt.Errorf("failed while trying to scale down the etcd cluster: %v", zap.Error(err))
	}

	if err := wait.Poll(30*time.Second, healthCheckTimeout*time.Minute, func() (bool, error) {
		healthy, err := r.isClusterEtcdHealthy(ctx, cluster)
		if err != nil {
			log.Warnf("failed to check cluster etcd health status: %v", zap.Error(err))
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd cluster scale down status: %v", zap.Error(err))
	}
	// count the pods!
	readyPods, err = r.getStsReadyPodsCount(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to check ready pods count: %v", zap.Error(err))
	}
	if readyPods != scaleDownCount {
		return fmt.Errorf("failed to scale down etcd cluster: want [%d] nodes, got [%d]", scaleDownCount, readyPods)
	}
	log.Info("etcd cluster scaled down successfully...")

	log.Info("waiting for etcd to regain quorum...")
	time.Sleep(2 * time.Minute)

	// delete one of the etcd node PVs
	log.Info("testing etcd node PV autotmatic recovery...")
	if err := r.forceDeleteEtcdPV(ctx, cluster); err != nil {
		return fmt.Errorf("failed to delete etcd node PV: %v", zap.Error(err))
	}

	// auto recovery should kick in. We need to wait for it
	if err := wait.Poll(30*time.Second, healthCheckTimeout*time.Minute, func() (bool, error) {
		healthy, err := r.isClusterEtcdHealthy(ctx, cluster)
		if err != nil {
			log.Warnf("failed to check cluster etcd health status: %v", zap.Error(err))
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd cluster health status: %v", zap.Error(err))
	}
	log.Info("etcd node PV recoverd successfully...")

	log.Info("waiting for etcd to regain quorum...")
	time.Sleep(2 * time.Minute)

	// check if we can disable it
	log.Info("Disabling etcd-launcher...")
	if err := r.disableEtcdlauncherForCluster(ctx, cluster); err != nil {
		return fmt.Errorf("failed to disable etcd-launcher: %v", zap.Error(err))
	}
	if err := wait.Poll(30*time.Second, 5*time.Minute, func() (bool, error) {
		healthy, err := r.isClusterEtcdHealthy(ctx, cluster)
		if err != nil {
			log.Warnf("failed to check cluster etcd health status: %v", zap.Error(err))
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd-launcher disabled status: %v", zap.Error(err))
	}
	log.Info("etcd-launcher disabled successfully...")
	log.Info(" Successfully tested etcd-launcher features.. ")
	return nil
}

// enable etcd launcher for the cluster
func (r *testRunner) enableEtcdlauncherForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return r.setClusterLauncherFeature(ctx, cluster, true)
}

func (r *testRunner) disableEtcdlauncherForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return r.setClusterLauncherFeature(ctx, cluster, false)
}

func (r *testRunner) setClusterLauncherFeature(ctx context.Context, cluster *kubermaticv1.Cluster, flag bool) error {
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %v", zap.Error(err))
	}

	if cluster.Spec.Features == nil {
		cluster.Spec.Features = map[string]bool{}
	}
	if cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] == flag {
		return nil
	}

	oldCluster := cluster.DeepCopy()

	cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = flag
	return r.seedClusterClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

// check etcd health
func (r *testRunner) isClusterEtcdHealthy(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return false, fmt.Errorf("failed to get cluster: %v", zap.Error(err))
	}
	sts := &appsv1.StatefulSet{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %v", zap.Error(err))
	}
	// we are healthy if the cluster controller is happy and the sts is ready
	return cluster.Status.ExtendedHealth.Etcd == kubermaticv1.HealthStatusUp &&
		*sts.Spec.Replicas == sts.Status.ReadyReplicas, nil
}

func (r *testRunner) isEtcdLauncherActive(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	etcdHealthy, err := r.isClusterEtcdHealthy(ctx, cluster)
	if err != nil {
		return false, fmt.Errorf("etcd health check failed: %v", zap.Error(err))
	}

	sts := &appsv1.StatefulSet{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %v", zap.Error(err))
	}

	return etcdHealthy && sts.Spec.Template.Spec.Containers[0].Command[0] == "/opt/bin/etcd-launcher", nil
}

// change cluster etcd size.
func (r *testRunner) resizeEtcd(ctx context.Context, cluster *kubermaticv1.Cluster, size int) error {
	if size > kubermaticv1.MaxEtcdClusterSize || size < kubermaticv1.DefaultEtcdClusterSize {
		return fmt.Errorf("Invalid etcd cluster size: %d", size)
	}

	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %v", zap.Error(err))
	}
	if cluster.Spec.ComponentsOverride.Etcd.ClusterSize == size {
		return nil
	}
	oldCluster := cluster.DeepCopy()
	cluster.Spec.ComponentsOverride.Etcd.ClusterSize = size
	return r.seedClusterClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *testRunner) forceDeleteEtcdPV(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := r.seedClusterClient.List(ctx, pvcList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Namespace}); err != nil || len(pvcList.Items) == 0 {
		return fmt.Errorf("failed to list PVCs or empty list in custer namespace: %v", zap.Error(err))
	}
	// pick a random PVC, get it's PV and delete it
	pvc := pvcList.Items[rand.Intn(len(pvcList.Items))]
	pvName := pvc.Spec.VolumeName
	pv := &corev1.PersistentVolume{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: pvName, Namespace: cluster.Namespace}, pv); err != nil {
		return fmt.Errorf("failed to get etcd node PV :%v", zap.Error(err))
	}
	oldPv := pv.DeepCopy()

	// first, we delete it
	if err := r.seedClusterClient.Delete(ctx, pv); err != nil {
		return fmt.Errorf("failed to delete etcd node PV: %v", zap.Error(err))
	}
	// now it will get stuck, we need to patch it to remove the pv finalizer
	pv.Finalizers = nil
	if err := r.seedClusterClient.Patch(ctx, pv, ctrlruntimeclient.MergeFrom(oldPv)); err != nil {
		return fmt.Errorf("failed to delete the PV finalizer: %v", zap.Error(err))
	}
	// make sure it's gone
	return wait.Poll(30*time.Second, healthCheckTimeout*time.Minute, func() (bool, error) {
		if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: pvName, Namespace: cluster.Namespace}, pv); kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

func (r *testRunner) getStsReadyPodsCount(ctx context.Context, cluster *kubermaticv1.Cluster) (int32, error) {
	sts := &appsv1.StatefulSet{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}, sts); err != nil {
		return 0, fmt.Errorf("failed to get StatefulSet: %v", zap.Error(err))
	}
	return sts.Status.ReadyReplicas, nil
}
