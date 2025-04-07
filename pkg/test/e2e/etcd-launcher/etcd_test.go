//go:build e2e

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
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	logOptions  = utils.DefaultLogOptions
	credentials jig.BYOCredentials
)

const (
	scaleUpCount           = 5
	scaleDownCount         = 3
	minioBackupDestination = "minio"
	namespaceName          = "backup-test"
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestBackup(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	client, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewBYOCluster(client, logger, credentials)
	testJig.ClusterJig.WithTestName("etcd-backup").WithFeatures(map[string]bool{
		// make sure etcd-launcher is initially not enabled.
		kubermaticv1.ClusterFeatureEtcdLauncher: false,
	})

	_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
	defer testJig.Cleanup(ctx, t, false)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		t.Fatalf("failed to create user cluster client: %v", err)
	}

	// Create a resource on the cluster so that we can see that the backup works
	logger.Info("Creating test namespace...")
	testNamespace := &corev1.Namespace{}
	testNamespace.Name = namespaceName
	err = userClient.Create(ctx, testNamespace)
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}
	logger.Info("created test namespace")

	// create etcd backup that will be restored later
	backup, err := createBackup(ctx, logger, client, cluster)
	if err != nil {
		t.Fatalf("failed to create etcd backup: %v", err)
	}
	logger.Info("created etcd backup")

	// delete the test resource
	err = userClient.Delete(ctx, testNamespace)
	if err != nil {
		t.Fatalf("failed to delete test namespace: %v", err)
	}
	logger.Info("deleted test namespace")

	// enable etcd-launcher feature after creating a backup
	if err := enableLauncher(ctx, logger, client, cluster); err != nil {
		t.Fatalf("failed to enable etcd-launcher: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	// restore from backup
	if err := restoreBackup(ctx, logger, client, cluster, backup); err != nil {
		t.Fatalf("failed to restore etcd backup: %v", err)
	}
	logger.Info("restored etcd backup")

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	// check if resource was restored
	restoredNamespace := &corev1.Namespace{}
	err = userClient.Get(ctx, types.NamespacedName{Name: namespaceName}, restoredNamespace)
	if err != nil {
		t.Fatalf("failed to get restored test namespace: %v", err)
	}
	logger.Info("deleted namespace was restored by backup")
}

func TestScaling(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(logOptions).Sugar()

	client, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewBYOCluster(client, logger, credentials)
	testJig.ClusterJig.WithTestName("etcd-scaling").WithFeatures(map[string]bool{
		kubermaticv1.ClusterFeatureEtcdLauncher: true,
	})

	_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
	defer testJig.Cleanup(ctx, t, false)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := scaleUp(ctx, logger, client, cluster); err != nil {
		t.Fatalf("failed to scale up: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := scaleDown(ctx, logger, client, cluster); err != nil {
		t.Fatalf("failed to scale down: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := disableLauncher(ctx, logger, client, cluster); err != nil {
		t.Fatalf("succeeded in disabling immutable feature etcd-launcher: %v", err)
	}
}

func TestRecovery(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(logOptions).Sugar()

	client, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewBYOCluster(client, logger, credentials)
	testJig.ClusterJig.WithTestName("etcd-recovery").WithFeatures(map[string]bool{
		kubermaticv1.ClusterFeatureEtcdLauncher: true,
	})

	_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
	defer testJig.Cleanup(ctx, t, false)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := breakAndRecoverPV(ctx, logger, client, cluster); err != nil {
		t.Fatalf("failed to test volume recovery: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := breakAndRecoverPVC(ctx, logger, client, cluster); err != nil {
		t.Fatalf("failed to recover from PVC deletion: %v", err)
	}

	if err := waitForClusterHealthy(ctx, logger, client, cluster); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}
}

func createBackup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfig, error) {
	log.Info("creating backup of etcd data...")
	backup := &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-e2e-backup",
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.EtcdBackupConfigSpec{
			Cluster: corev1.ObjectReference{
				Kind:            cluster.Kind,
				Name:            cluster.Name,
				Namespace:       cluster.Namespace,
				UID:             cluster.UID,
				APIVersion:      cluster.APIVersion,
				ResourceVersion: cluster.ResourceVersion,
			},
			Destination: minioBackupDestination,
		},
	}

	if err := client.Create(ctx, backup); err != nil {
		return nil, fmt.Errorf("failed to create EtcdBackupConfig: %w", err)
	}

	if err := waitForEtcdBackup(ctx, log, client, backup); err != nil {
		return nil, fmt.Errorf("failed to wait for etcd backup finishing: %w (%v)", err, backup.Status)
	}

	return backup, nil
}

func restoreBackup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, backup *kubermaticv1.EtcdBackupConfig) error {
	log.Info("restoring etcd cluster from backup...")
	restore := &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-e2e-restore",
			Namespace: backup.Namespace,
		},
		Spec: kubermaticv1.EtcdRestoreSpec{
			Cluster: corev1.ObjectReference{
				Kind:            cluster.Kind,
				Name:            cluster.Name,
				Namespace:       cluster.Namespace,
				UID:             cluster.UID,
				APIVersion:      cluster.APIVersion,
				ResourceVersion: cluster.ResourceVersion,
			},
			BackupName:  backup.Status.CurrentBackups[0].BackupName,
			Destination: minioBackupDestination,
		},
	}

	if err := client.Create(ctx, restore); err != nil {
		return fmt.Errorf("failed to create EtcdRestore: %w", err)
	}

	if err := waitForEtcdRestore(ctx, log, client, restore); err != nil {
		return fmt.Errorf("failed to wait for etcd restore: %w", err)
	}

	if err := waitForClusterHealthy(ctx, log, client, cluster); err != nil {
		return fmt.Errorf("failed to wait for cluster to become healthy again: %w", err)
	}

	return nil
}

func enableLauncher(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	log.Info("enabling etcd-launcher...")
	if err := enableLauncherForCluster(ctx, client, cluster); err != nil {
		return fmt.Errorf("failed to enable etcd-launcher: %w", err)
	}

	if err := waitForClusterHealthy(ctx, log, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %w", err)
	}

	if err := waitForStrictTLSMode(ctx, log, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not running in strict TLS peer mode: %w", err)
	}

	active, err := isEtcdLauncherActive(ctx, client, cluster)
	if err != nil {
		return fmt.Errorf("failed to check StatefulSet command: %w", err)
	}

	if !active {
		return errors.New("feature flag had no effect on the StatefulSet")
	}

	return nil
}

func disableLauncher(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	log.Info("trying to disable etcd-launcher (not expected to succeed) ...")
	if err := disableEtcdlauncherForCluster(ctx, client, cluster); err == nil {
		return fmt.Errorf("no error disabling etcd-launcher, expected validation to fail")
	}

	return nil
}

func scaleUp(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	log.Infof("scaling etcd cluster up to %d nodes...", scaleUpCount)
	if err := resizeEtcd(ctx, client, cluster, scaleUpCount); err != nil {
		return fmt.Errorf("failed while trying to scale up the etcd cluster: %w", err)
	}

	if err := waitForRollout(ctx, log, client, cluster, scaleUpCount); err != nil {
		return fmt.Errorf("rollout got stuck: %w", err)
	}
	log.Info("etcd cluster scaled up successfully.")

	return nil
}

func scaleDown(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	log.Infof("scaling etcd cluster down to %d nodes...", scaleDownCount)
	if err := resizeEtcd(ctx, client, cluster, scaleDownCount); err != nil {
		return fmt.Errorf("failed while trying to scale down the etcd cluster: %w", err)
	}

	if err := waitForRollout(ctx, log, client, cluster, scaleDownCount); err != nil {
		return fmt.Errorf("rollout got stuck: %w", err)
	}
	log.Info("etcd cluster scaled down successfully.")

	return nil
}

func breakAndRecoverPV(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	// delete one of the etcd node PVs
	log.Info("testing etcd node PV automatic recovery...")
	if err := forceDeleteEtcdPV(ctx, client, cluster); err != nil {
		return fmt.Errorf("failed to delete etcd node PV: %w", err)
	}

	// wait for a bit before checking health as the PV recovery process
	// is a controller-manager loop that doesn't necessarily kick in immediately
	time.Sleep(30 * time.Second)

	// auto recovery should kick in. We need to wait for it
	if err := waitForClusterHealthy(ctx, log, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %w", err)
	}
	log.Info("etcd node PV recovered successfully.")

	return nil
}

func breakAndRecoverPVC(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	// delete one of the etcd node PVCs
	log.Info("testing etcd-launcher recovery from deleted PVC ...")
	if err := deleteEtcdPVC(ctx, client, cluster); err != nil {
		return fmt.Errorf("failed to delete etcd node PVC: %w", err)
	}

	time.Sleep(30 * time.Second)

	if err := waitForClusterHealthy(ctx, log, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %w", err)
	}

	log.Info("etcd node recovered from PVC deletion successfully.")

	return nil
}

// enable etcd launcher for the cluster.
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
		return false, fmt.Errorf("failed to get cluster: %w", err)
	}

	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	clusterSize := int32(3)
	if size := cluster.Spec.ComponentsOverride.Etcd.ClusterSize; size != nil {
		clusterSize = *size
	}

	// we are healthy if the cluster controller is happy and the sts has ready replicas
	// matching the cluster's expected etcd cluster size
	return cluster.Status.ExtendedHealth.Etcd == kubermaticv1.HealthStatusUp &&
		clusterSize == sts.Status.ReadyReplicas, nil
}

func isStrictTLSEnabled(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	etcdHealthy, err := isClusterEtcdHealthy(ctx, client, cluster)
	if err != nil {
		return false, fmt.Errorf("etcd health check failed: %w", err)
	}

	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	strictModeEnvSet := false

	for _, env := range sts.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "PEER_TLS_MODE" && env.Value == "strict" {
			strictModeEnvSet = true
		}
	}

	return etcdHealthy && strictModeEnvSet, nil
}

// isEtcdLauncherActive deduces from the StatefulSet's current spec whether or
// or not the etcd-launcher is enabled (and reconciled).
func isEtcdLauncherActive(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	etcdHealthy, err := isClusterEtcdHealthy(ctx, client, cluster)
	if err != nil {
		return false, fmt.Errorf("etcd health check failed: %w", err)
	}

	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	return etcdHealthy && sts.Spec.Template.Spec.Containers[0].Command[0] == "/opt/bin/etcd-launcher", nil
}

func isEtcdBackupCompleted(status *kubermaticv1.EtcdBackupConfigStatus) bool {
	if length := len(status.CurrentBackups); length != 1 {
		return false
	}

	if status.CurrentBackups[0].BackupPhase == kubermaticv1.BackupStatusPhaseCompleted {
		return true
	}

	return false
}

func isEtcdRestoreCompleted(status *kubermaticv1.EtcdRestoreStatus) bool {
	return status.Phase == kubermaticv1.EtcdRestorePhaseCompleted
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

func waitForEtcdBackup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, backup *kubermaticv1.EtcdBackupConfig) error {
	before := time.Now()
	if err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, types.NamespacedName{Name: backup.Name, Namespace: backup.Namespace}, backup); err != nil {
			return false, err
		}

		return isEtcdBackupCompleted(&backup.Status), nil
	}); err != nil {
		return err
	}

	log.Infof("etcd backup finished after %v.", time.Since(before))
	return nil
}

func waitForEtcdRestore(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, restore *kubermaticv1.EtcdRestore) error {
	before := time.Now()
	if err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, types.NamespacedName{Name: restore.Name, Namespace: restore.Namespace}, restore); err != nil {
			return false, err
		}

		return isEtcdRestoreCompleted(&restore.Status), nil
	}); err != nil {
		return fmt.Errorf("failed waiting for restore to complete: %w (%v)", err, restore.Status)
	}

	log.Infof("etcd restore finished after %v.", time.Since(before))
	return nil
}

func waitForClusterHealthy(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	before := time.Now()

	// let's briefly sleep to give controllers a chance to kick in
	time.Sleep(10 * time.Second)

	if err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 10*time.Minute, true, func(ctx context.Context) (bool, error) {
		// refresh cluster object for updated health status
		if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
			return false, fmt.Errorf("failed to get cluster: %w", err)
		}

		healthy, err := isClusterEtcdHealthy(ctx, client, cluster)
		if err != nil {
			log.Infof("failed to check cluster etcd health status: %v", err)
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd health status: %w", err)
	}

	log.Infof("etcd cluster became healthy after %v.", time.Since(before))

	return nil
}

func waitForStrictTLSMode(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	before := time.Now()
	if err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 10*time.Minute, true, func(ctx context.Context) (bool, error) {
		// refresh cluster object for updated health status
		if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
			return false, fmt.Errorf("failed to get cluster: %w", err)
		}

		healthy, err := isStrictTLSEnabled(ctx, client, cluster)
		if err != nil {
			log.Infof("failed to check cluster etcd health status: %v", err)
			return false, nil
		}
		return healthy, nil
	}); err != nil {
		return fmt.Errorf("failed to check etcd health status: %w", err)
	}

	log.Infof("etcd cluster is running in strict TLS mode after %v.", time.Since(before))

	return nil
}

func waitForRollout(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, targetSize int) error {
	log.Info("waiting for rollout...")

	if err := waitForClusterHealthy(ctx, log, client, cluster); err != nil {
		return fmt.Errorf("etcd cluster is not healthy: %w", err)
	}

	// count the pods
	readyPods, err := getStsReadyPodsCount(ctx, client, cluster)
	if err != nil {
		return fmt.Errorf("failed to check ready pods count: %w", err)
	}
	if int(readyPods) != targetSize {
		return fmt.Errorf("failed to scale etcd cluster: want [%d] nodes, got [%d]", targetSize, readyPods)
	}

	return nil
}

func forceDeleteEtcdPV(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	ns := clusterNamespace(cluster)

	selector, err := labels.Parse("app=etcd")
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %w", err)
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	opt := &ctrlruntimeclient.ListOptions{
		LabelSelector: selector,
		Namespace:     ns,
	}
	if err := client.List(ctx, pvcList, opt); err != nil || len(pvcList.Items) == 0 {
		return fmt.Errorf("failed to list PVCs or empty list in cluster namespace: %w", err)
	}

	// pick a random PVC, get its PV and delete it
	pvc := pvcList.Items[rand.Intn(len(pvcList.Items))]
	pvName := pvc.Spec.VolumeName
	typedName := types.NamespacedName{Name: pvName, Namespace: ns}

	pv := &corev1.PersistentVolume{}
	if err := client.Get(ctx, typedName, pv); err != nil {
		return fmt.Errorf("failed to get etcd node PV %s: %w", pvName, err)
	}
	oldPv := pv.DeepCopy()

	// first, we delete it
	if err := client.Delete(ctx, pv); err != nil {
		return fmt.Errorf("failed to delete etcd node PV %s: %w", pvName, err)
	}

	// now it will get stuck, we need to patch it to remove the pv finalizer
	pv.Finalizers = nil
	if err := client.Patch(ctx, pv, ctrlruntimeclient.MergeFrom(oldPv)); err != nil {
		return fmt.Errorf("failed to delete the PV %s finalizer: %w", pvName, err)
	}

	// make sure it's gone
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, typedName, pv); apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

func deleteEtcdPVC(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	ns := clusterNamespace(cluster)

	selector, err := labels.Parse("app=etcd")
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %w", err)
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	opt := &ctrlruntimeclient.ListOptions{
		LabelSelector: selector,
		Namespace:     ns,
	}
	if err := client.List(ctx, pvcList, opt); err != nil || len(pvcList.Items) == 0 {
		return fmt.Errorf("failed to list PVCs or empty list in cluster namespace: %w", err)
	}

	// pick a random PVC and get the corresponding pod
	index := rand.Intn(len(pvcList.Items))
	pvc := pvcList.Items[index]
	oldPvc := pvc.DeepCopy()

	podList := &corev1.PodList{}
	if err := client.List(ctx, podList, opt); err != nil || len(podList.Items) != len(pvcList.Items) {
		return fmt.Errorf("failed to list etcd pods or bad number of pods: %w", err)
	}

	pod := podList.Items[index]

	// first, we delete it
	if err := client.Delete(ctx, &pvc); err != nil {
		return fmt.Errorf("failed to delete etcd node PVC %s: %w", pvc.Name, err)
	}

	// now, we delete the pod so the PVC can be finalised
	if err := client.Delete(ctx, &pod); err != nil {
		return fmt.Errorf("failed to delete etcd pod %s: %w", pod.Name, err)
	}

	// make sure the PVC is recreated by checking the CreationTimestamp against a DeepCopy
	// created of the PVC resource.
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &pvc); err == nil {
			if oldPvc.CreationTimestamp.Before(&pvc.CreationTimestamp) {
				return true, nil
			}
		}
		return false, nil
	})
}

func getStsReadyPodsCount(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (int32, error) {
	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: clusterNamespace(cluster)}, sts); err != nil {
		return 0, fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	return sts.Status.ReadyReplicas, nil
}

func clusterNamespace(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s", cluster.Name)
}

type patchFunc func(cluster *kubermaticv1.Cluster) error

func patchCluster(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, patch patchFunc) error {
	if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	oldCluster := cluster.DeepCopy()
	if err := patch(cluster); err != nil {
		return err
	}

	if err := client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	// give KKP some time to reconcile
	time.Sleep(10 * time.Second)

	return nil
}
