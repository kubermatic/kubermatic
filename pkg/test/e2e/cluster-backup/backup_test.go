//go:build e2e

/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package clusterbackup

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

func env(key string) (string, error) {
	value := os.Getenv(key)
	if len(value) == 0 {
		return "", fmt.Errorf("no %s environment variable defined", key)
	}

	return value, nil
}

type backupCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
}

func (c *backupCredentials) AddFlags(fs *flag.FlagSet) {
	// NOP
}

func (c *backupCredentials) Parse() (err error) {
	if c.AccessKeyID, err = env("BACKUP_ACCESS_KEY_ID"); err != nil {
		return err
	}

	if c.SecretAccessKey, err = env("BACKUP_SECRET_ACCESS_KEY"); err != nil {
		return err
	}

	if c.Bucket, err = env("BACKUP_BUCKET"); err != nil {
		return err
	}

	if c.Region, err = env("BACKUP_REGION"); err != nil {
		return err
	}

	return nil
}

const (
	veleroNamespace    = "velero"
	veleroDeployment   = "velero"
	userClusterBSLName = "default-cluster-backup-bsl"
)

var (
	credentials   jig.AWSCredentials
	s3Credentials backupCredentials
	logOptions    = utils.DefaultLogOptions

	dummyObject = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "canary",
			Namespace: "kube-public",
		},
		Data: map[string]string{
			"backups": "... are fun!",
		},
	}
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	s3Credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestClusterBackupAndRestore(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	if err := s3Credentials.Parse(); err != nil {
		t.Fatalf("Failed to get S3 credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	defer testJig.Cleanup(ctx, t, true)

	testJig.ClusterJig.WithTestName("clusterbackup")

	project, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	cbslName, err := createBackupConfiguration(ctx, logger, seedClient, project, s3Credentials)
	if err != nil {
		t.Fatalf("Failed to setup backup config: %v", err)
	}

	// let the games begin

	logger.Info("Enabling cluster backups...")
	if err := setClusterBackupConfig(ctx, seedClient, cluster, cbslName); err != nil {
		t.Fatalf("Failed to enable cluster backups: %v", err)
	}

	logger.Info("Waiting for Velero to be up and running...")
	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create user cluster client: %v", err)
	}

	if err := velerov1.AddToScheme(userClient.Scheme()); err != nil {
		t.Fatalf("Failed to register velero/v1 scheme: %v", err)
	}

	if err := wait.PollImmediateLog(ctx, logger, 5*time.Second, 5*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		key := types.NamespacedName{Name: veleroDeployment, Namespace: veleroNamespace}
		health, err := resources.HealthyDeployment(ctx, userClient, key, -1)
		if err != nil {
			return fmt.Errorf("failed to check health: %w", err), nil
		}

		if health != kubermaticv1.HealthStatusUp {
			return fmt.Errorf("Velero is still %v", health), nil
		}

		return nil, nil
	}); err != nil {
		t.Fatalf("Velero never became available: %v", err)
	}

	logger.Info("Waiting for BackupStorageLocation to be synced...")
	if err := wait.PollImmediateLog(ctx, logger, 5*time.Second, 5*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		key := types.NamespacedName{Name: userClusterBSLName, Namespace: veleroNamespace}
		userCBSL := &velerov1.BackupStorageLocation{}

		return userClient.Get(ctx, key, userCBSL), nil
	}); err != nil {
		t.Fatalf("BSL never became available: %v", err)
	}

	logger.Info("Creating canary...")
	canary := dummyObject.DeepCopy()
	if err := userClient.Create(ctx, canary); err != nil {
		t.Fatalf("Failed to create canary: %v", err)
	}

	logger.Info("Creating backup through Velero...")
	clusterBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: veleroNamespace,
		},
		Spec: velerov1.BackupSpec{
			StorageLocation:    userClusterBSLName,
			IncludedNamespaces: []string{canary.Namespace},
			IncludedResources:  []string{"configmaps"},
		},
	}

	if err := userClient.Create(ctx, clusterBackup); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	logger.Info("Waiting for backup to complete...")
	if err := wait.PollImmediateLog(ctx, logger, 5*time.Second, 5*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		// refresh Backup information
		if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(clusterBackup), clusterBackup); err != nil {
			return err, nil
		}

		if phase := clusterBackup.Status.Phase; phase != velerov1.BackupPhaseCompleted {
			return fmt.Errorf("backup is still %v", phase), nil
		}

		return nil, nil
	}); err != nil {
		t.Fatalf("Backup never finished: %v", err)
	}

	logger.Info("Backup finished successfully.")

	// make some modifications

	logger.Info("Modifying cluster state...")
	if err := userClient.Delete(ctx, canary); err != nil {
		t.Fatalf("Failed to delete canary: %v", err)
	}

	// undo these changes using Velero

	logger.Info("Creating restore through Velero...")
	clusterRestore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-test-backup",
			Namespace: veleroNamespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName: clusterBackup.Name,
		},
	}

	if err := userClient.Create(ctx, clusterRestore); err != nil {
		t.Fatalf("Failed to create restore object: %v", err)
	}

	logger.Info("Waiting for restore to complete...")
	if err := wait.PollImmediateLog(ctx, logger, 5*time.Second, 5*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		// refresh Restore information
		if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(clusterRestore), clusterRestore); err != nil {
			return err, nil
		}

		if phase := clusterRestore.Status.Phase; phase != velerov1.RestorePhaseCompleted {
			return fmt.Errorf("restore is still %v", phase), nil
		}

		return nil, nil
	}); err != nil {
		t.Fatalf("Restore never finished: %v", err)
	}

	logger.Info("Restore finished successfully.")

	// verify that the canary was resurrected

	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(canary), canary); err != nil {
		t.Fatalf("Failed to get restored canary: %v", err)
	}

	logger.Info("Disabling cluster backups...")
	if err := setClusterBackupConfig(ctx, seedClient, cluster, ""); err != nil {
		t.Fatalf("Failed to disable cluster backups: %v", err)
	}
}

func createBackupConfiguration(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, project *kubermaticv1.Project, creds backupCredentials) (string, error) {
	log.Info("Creating backup configuration...")

	credentials := &corev1.Secret{}
	credentials.Name = "cluster-backup-e2e-credentials"
	credentials.Namespace = resources.KubermaticNamespace

	credentials.Data = map[string][]byte{
		"accessKeyId":     []byte(creds.AccessKeyID),
		"secretAccessKey": []byte(creds.SecretAccessKey),
	}

	if err := client.Create(ctx, credentials); err != nil {
		return "", fmt.Errorf("failed to create credential secret: %w", err)
	}

	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	cbsl.Name = "cluster-backup-e2e"
	cbsl.Namespace = credentials.Namespace
	cbsl.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: project.Name,
	}
	cbsl.Spec = velerov1.BackupStorageLocationSpec{
		Provider: "aws",
		Config: map[string]string{
			"region": creds.Region,
		},
		StorageType: velerov1.StorageType{
			ObjectStorage: &velerov1.ObjectStorageLocation{
				Bucket: creds.Bucket,
				Prefix: jig.BuildID(),
			},
		},
		Credential: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: credentials.Name,
			},
			// KKP's controllers will later create a suitable Secret with a suitable key like this.
			Key: "cloud-credentials",
		},
	}

	if err := client.Create(ctx, cbsl); err != nil {
		return "", fmt.Errorf("failed to create CBSL: %w", err)
	}

	return cbsl.Name, nil
}

func setClusterBackupConfig(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, cbslName string) error {
	oldCluster := cluster.DeepCopy()

	if cbslName == "" {
		cluster.Spec.BackupConfig = nil
	} else {
		cluster.Spec.BackupConfig = &kubermaticv1.BackupConfig{
			BackupStorageLocation: &corev1.LocalObjectReference{
				Name: cbslName,
			},
		}
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
