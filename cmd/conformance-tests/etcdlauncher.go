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
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"time"

	"go.etcd.io/etcd/v3/pkg/transport"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"

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
	healthCheckTimeout = 10 * time.Minute
)

func (r *testRunner) testEtcdLauncher(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// So far, the etcd-launcher is experimental. We need to enable it first
	log.Info("Testing etcd-launcher experimental features.")

	log.Info("Enabling etcd-launcher...")
	if err := r.setClusterLauncherFeature(ctx, cluster, true); err != nil {
		return fmt.Errorf("failed to enable etcd-launcher: %v", err)
	}

	if err := wait.Poll(2*time.Second, healthCheckTimeout, func() (bool, error) {
		active, err := r.isEtcdLauncherActive(ctx, cluster)
		if err != nil {
			log.Warnw("Failed to check etcd-launcher status", zap.Error(err))
			return false, nil
		}
		return active, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd-launcher enabled status: %v", err)
	}
	log.Info("etcd-launcher enabled successfully.")

	// let etcd settle down
	if err := r.waitForHealthyEtcd(ctx, log, cluster); err != nil {
		return fmt.Errorf("etcd did not become healthy: %v", err)
	}

	// scale up to 5 nodes
	if err := r.etcdScaleTest(ctx, log, cluster, scaleUpCount); err != nil {
		return fmt.Errorf("failed to scale up etcd cluster: %v", err)
	}

	// scale down to 3 nodes
	if err := r.etcdScaleTest(ctx, log, cluster, scaleDownCount); err != nil {
		return fmt.Errorf("failed to scale down etcd cluster: %v", err)
	}

	// poke the anthill: delete one of the etcd node PVs
	log.Info("Testing automatic node PV recovery...")
	if err := r.forceDeleteEtcdPV(ctx, cluster); err != nil {
		return fmt.Errorf("failed to delete etcd node PV: %v", err)
	}

	// make sure etcd notices the missing volume and does not answer requests based on caches
	time.Sleep(5 * time.Second)

	// wait for the cluster to recover automatically
	if err := r.waitForHealthyEtcd(ctx, log, cluster); err != nil {
		return fmt.Errorf("etcd did not become healthy: %v", err)
	}
	log.Info("Node PV recovered successfully.")

	// disable etcd-launcher again
	log.Info("Disabling etcd-launcher...")
	if err := r.setClusterLauncherFeature(ctx, cluster, false); err != nil {
		return fmt.Errorf("failed to disable etcd-launcher: %v", err)
	}
	if err := r.waitForHealthyEtcd(ctx, log, cluster); err != nil {
		return fmt.Errorf("etcd did not become healthy: %v", err)
	}
	log.Info("etcd-launcher disabled successfully...")

	// woohoo, success!
	log.Info("Successfully tested etcd-launcher features.")

	return nil
}

func (r *testRunner) waitForHealthyEtcd(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	log.Info("Waiting for etcd to become fully healthy...")

	if err := wait.Poll(2*time.Second, healthCheckTimeout, func() (bool, error) {
		return r.isEtcdHealthy(ctx, log, cluster)
	}); err != nil {
		return fmt.Errorf("failed to check etcd health: %v", err)
	}

	log.Info("Cluster is fully healthy.")

	return nil
}

// etcdScaleTest rescales the etcd ring to a given size, waits for it to become
// healthy and then checks if there are the desired number of pods.
func (r *testRunner) etcdScaleTest(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, targetSize int) error {
	// scale up to 5 nodes
	log.Infof("Testing etcd cluster scaling: scaling etcd cluster to %d nodes...", targetSize)
	if err := r.resizeEtcd(ctx, cluster, targetSize); err != nil {
		return fmt.Errorf("failed to scale up the etcd cluster: %v", err)
	}

	if err := r.waitForHealthyEtcd(ctx, log, cluster); err != nil {
		return fmt.Errorf("etcd did not become healthy: %v", err)
	}

	// count the pods, in case the etcd cluster is healthy but too small or too large
	readyPods, err := r.getStsReadyPodsCount(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to check ready pods count: %v", err)
	}
	if readyPods != targetSize {
		return fmt.Errorf("failed to scale etcd cluster: want [%d] nodes, got [%d]", targetSize, readyPods)
	}

	return nil
}

func (r *testRunner) setClusterLauncherFeature(ctx context.Context, cluster *kubermaticv1.Cluster, flag bool) error {
	return r.patchCluster(ctx, cluster, func(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
		if cluster.Spec.Features == nil {
			cluster.Spec.Features = map[string]bool{}
		}

		cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = flag

		return cluster, nil
	})
}

func (r *testRunner) isEtcdLauncherActive(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	sts, err := r.getEtcdStatefulSet(ctx, cluster)
	if err != nil {
		return false, err
	}

	return sts.Spec.Template.Spec.Containers[0].Command[0] == "/opt/bin/etcd-launcher", nil
}

// change cluster etcd size.
func (r *testRunner) resizeEtcd(ctx context.Context, cluster *kubermaticv1.Cluster, size int) error {
	return r.patchCluster(ctx, cluster, func(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
		if size > kubermaticv1.MaxEtcdClusterSize || size < kubermaticv1.DefaultEtcdClusterSize {
			return nil, fmt.Errorf("Invalid etcd cluster size: %d", size)
		}

		cluster.Spec.ComponentsOverride.Etcd.ClusterSize = size

		return cluster, nil
	})
}

func (r *testRunner) forceDeleteEtcdPV(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := r.seedClusterClient.List(ctx, pvcList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Namespace}); err != nil || len(pvcList.Items) == 0 {
		return fmt.Errorf("failed to list PVCs or empty list in cluster namespace: %v", err)
	}

	// pick a random PVC, get its PV and delete it
	pvc := pvcList.Items[rand.Intn(len(pvcList.Items))]
	pvName := pvc.Spec.VolumeName
	pv := &corev1.PersistentVolume{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: pvName, Namespace: cluster.Namespace}, pv); err != nil {
		return fmt.Errorf("failed to get etcd node PV: %v", err)
	}
	oldPv := pv.DeepCopy()

	// first, we delete it
	if err := r.seedClusterClient.Delete(ctx, pv); err != nil {
		return fmt.Errorf("failed to delete etcd node PV: %v", err)
	}

	// now it will get stuck, we need to patch it to remove the pv finalizer
	pv.Finalizers = nil
	if err := r.seedClusterClient.Patch(ctx, pv, ctrlruntimeclient.MergeFrom(oldPv)); err != nil {
		return fmt.Errorf("failed to delete the PV finalizer: %v", err)
	}

	// make sure it's gone
	return wait.Poll(2*time.Second, healthCheckTimeout, func() (bool, error) {
		if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: pvName, Namespace: cluster.Namespace}, pv); kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

func (r *testRunner) getStsReadyPodsCount(ctx context.Context, cluster *kubermaticv1.Cluster) (int, error) {
	sts, err := r.getEtcdStatefulSet(ctx, cluster)
	if err != nil {
		return 0, err
	}
	return int(sts.Status.ReadyReplicas), nil
}

func (r *testRunner) getCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster: %v", err)
	}

	return cluster, nil
}

func (r *testRunner) getEtcdStatefulSet(ctx context.Context, cluster *kubermaticv1.Cluster) (*appsv1.StatefulSet, error) {
	cluster, err := r.getCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}, sts); err != nil {
		return nil, fmt.Errorf("failed to get StatefulSet: %v", err)
	}

	return sts, nil
}

type clusterPatch func(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

func (r *testRunner) patchCluster(ctx context.Context, cluster *kubermaticv1.Cluster, patch clusterPatch) error {
	cluster, err := r.getCluster(ctx, cluster)
	if err != nil {
		return err
	}

	oldCluster := cluster.DeepCopy()

	cluster, err = patch(cluster)
	if err != nil {
		return err
	}

	return r.seedClusterClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

// getEtcdClient returns an etcd.Client setup to communicate with a usercluster's etcd ring.
func (r *testRunner) getEtcdClient(ctx context.Context, cluster *kubermaticv1.Cluster) (*etcd.Client, error) {
	// get TLS certificate
	etcdSecret := &corev1.Secret{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      resources.EtcdTLSCertificateSecretName,
	}, etcdSecret); err != nil {
		return nil, fmt.Errorf("failed to find etcd TLS certificate: %v", err)
	}

	caSecret := &corev1.Secret{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      resources.CASecretName,
	}, caSecret); err != nil {
		return nil, fmt.Errorf("failed to find etcd CA certificate: %v", err)
	}

	// dump secrets to temporary files
	tempDir, err := ioutil.TempDir("", "etcd.*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %v", err)
	}

	certFile := filepath.Join(tempDir, "etcd.crt")
	keyFile := filepath.Join(tempDir, "etcd.key")
	caFile := filepath.Join(tempDir, "ca.crt")

	if err := ioutil.WriteFile(certFile, caSecret.Data[resources.EtcdTLSCertSecretKey], 0600); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	if err := ioutil.WriteFile(keyFile, caSecret.Data[resources.EtcdTLSKeySecretKey], 0600); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	if err := ioutil.WriteFile(caFile, caSecret.Data[resources.CACertSecretKey], 0600); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	clusterSize := cluster.Spec.ComponentsOverride.Etcd.ClusterSize
	if clusterSize <= 0 {
		clusterSize = kubermaticv1.DefaultEtcdClusterSize
	}

	endpoints := etcd.ClientEndpoints(clusterSize, cluster.Status.NamespaceName)

	return etcd.NewClient(endpoints, &transport.TLSInfo{
		CertFile:       certFile,
		KeyFile:        keyFile,
		TrustedCAFile:  caFile,
		ClientCertAuth: true,
	})
}

// isEtcdHealthy combines creating a client and checking the health of an etcd cluster.
// This is useful for the various wait loops in the tests.
func (r *testRunner) isEtcdHealthy(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (bool, error) {
	client, err := r.getEtcdClient(ctx, cluster)
	if err != nil {
		log.Warnw("Failed to create etcd client", zap.Error(err))
		return false, nil
	}
	defer client.Close()

	healthy, err := client.Healthy(ctx, nil)
	if err != nil {
		log.Warnw("Failed to check health status", zap.Error(err))
		return false, nil
	}

	return healthy, nil
}
