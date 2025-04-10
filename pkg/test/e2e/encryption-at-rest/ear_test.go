//go:build e2e

/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package encryptionatrest

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/encryption"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/podexec"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	credentials jig.BYOCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

const (
	defaultTimeout         = 10 * time.Minute
	defaultInterval        = 5 * time.Second
	minioBackupDestination = "minio"
)

type runner struct {
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client
	config     *rest.Config
	logger     *zap.SugaredLogger
	testJig    *jig.TestJig
}

func TestEncryptionAtRest(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, config, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	testJig := jig.NewBYOCluster(seedClient, logger, credentials)
	testJig.ClusterJig.WithTestName("ear").WithFeatures(map[string]bool{
		kubermaticv1.ClusterFeatureEtcdLauncher: true,
	})

	logger.Info("setting up the cluster")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	err = testJig.WaitForHealthyControlPlane(ctx, defaultTimeout)
	if err != nil {
		t.Fatalf("Cluster did not get healthy after enabling encryption-at-rest: %v", err)
	}

	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		t.Fatalf("failed to create user cluster client: %v", err)
	}

	logger.Info("creating a dummy secret for testing encryption-at-rest")
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dummy-",
			Namespace:    "default",
		},
		Data: map[string][]byte{
			"dummy-key": []byte("dummy-value"),
		},
	}

	err = userClient.Create(ctx, &secret)
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	r := runner{
		seedClient: seedClient,
		userClient: userClient,
		config:     config,
		logger:     logger,
		testJig:    testJig,
	}

	// Test Case 1:
	// Enable encryption at rest and verify it works
	logger.Info("Test Case 1 running...")
	err = r.enableEAR(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to enable encryption-at-rest: %v", err)
	}

	err = r.ensureDataEncryption(ctx, cluster, secret, config, true, encKeyName)
	if err != nil {
		t.Fatalf("failed to ensure data encryption: %v", err)
	}

	// Test Case 2:
	// Create an etcd backup for testing. Then, create a new secret that is not going to be part of the backup.
	// Rotate the encryption key to a new one. So, update the primary encryption key to the new one: `rotatedKeyName`.
	// Verify that the new secret is not encrypted with the initial encryption key; instead it is encrypted with the new encryption key.
	logger.Info("Test Case 2 running...")

	err = r.createEtcdBackup(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to create etcd backup: %v", err)
	}

	postBackupSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "post-backup-",
			Namespace:    "default",
		},
		Data: map[string][]byte{
			"post-backup-key": []byte("post-backup-value"),
		},
	}

	err = userClient.Create(ctx, &postBackupSecret)
	if err != nil {
		t.Fatalf("failed to create post-backup test secret: %v", err)
	}

	err = r.ensureDataEncryption(ctx, cluster, postBackupSecret, config, true, encKeyName)
	if err != nil {
		t.Fatalf("failed to ensure post-backup secret encryption: %v", err)
	}

	err = r.rotateEncryptionKey(ctx, cluster, secret, postBackupSecret)
	if err != nil {
		t.Fatalf("failed to rotate encryption key: %v", err)
	}

	// Test Case 3: Restore etcd backup created with previous key, which is now the secondary key.
	// After restore, verify the original secret (included in backup) is accessible.
	// Verify that the original secret is still properly encrypted in etcd after restore.
	// Verify the post-backup secret is not present as it wasn't in the backup.
	logger.Info("Test Case 3 running...")
	err = r.restoreEtcdBackup(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to restore etcd backup: %v", err)
	}

	// after restoring, verify that the secret is encrypted with the previous primary key.
	err = r.verifyDataAccess(ctx, cluster, secret, encKeyName)
	if err != nil {
		t.Fatalf("failed to access original secret after restore: %v", err)
	}

	// since postBackupSecret was created after the backup was created,
	// it should not be part of the backup restore.
	err = r.verifySecretDoesNotExist(ctx, postBackupSecret.Name, postBackupSecret.Namespace)
	if err != nil {
		t.Fatalf("post-backup secret unexpectedly exists after restore: %v", err)
	}

	// Test Case 4:
	// After restoring etcd backup, verify encryption still works for new secrets.
	logger.Info("Test Case 4 running...")
	postRestoreSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "post-restore-",
			Namespace:    "default",
		},
		Data: map[string][]byte{
			"post-restore-key": []byte("post-restore-value"),
		},
	}

	err = userClient.Create(ctx, &postRestoreSecret)
	if err != nil {
		t.Fatalf("failed to create post-restore test secret: %v", err)
	}

	err = r.ensureDataEncryption(ctx, cluster, postRestoreSecret, config, true, rotatedKeyName)
	if err != nil {
		t.Fatalf("encryption not working for new secrets after restore: %v", err)
	}

	// Test Case 5: Disable encryption at rest and verify data is decrypted automatically.
	logger.Info("Test Case 5 running...")

	err = r.disableEAR(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to disable encryption-at-rest: %v", err)
	}

	err = r.ensureDataEncryption(ctx, cluster, secret, config, false, encKeyName)
	if err != nil {
		t.Fatalf("failed to verify original secret is no longer encrypted: %v", err)
	}
}

func (r *runner) ensureAPIServerUpdated(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("waiting for ApiServer to contain configurations for encryption-at-rest")

	err := wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			updated, err := isApiserverUpdated(ctx, r.seedClient, cluster)
			if err != nil {
				return fmt.Errorf("failed to check apiserver status, %w", err), nil
			}

			if updated {
				return nil, nil
			}

			return fmt.Errorf("apiserver is not updated"), nil
		},
	)
	if err != nil {
		return err
	}

	r.logger.Info("apiserver is updated")
	return nil
}

func isApiserverUpdated(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	var secret corev1.Secret
	if err := client.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		return false, ctrlruntimeclient.IgnoreNotFound(err)
	}

	spec, err := json.Marshal(cluster.Spec.EncryptionConfiguration)
	if err != nil {
		return false, err
	}

	hash := sha1.New()
	hash.Write(spec)

	val, ok := secret.ObjectMeta.Labels[encryption.ApiserverEncryptionHashLabelKey] //nolint
	if !ok || val != hex.EncodeToString(hash.Sum(nil)) {
		return false, nil
	}

	var podList corev1.PodList
	if err := client.List(ctx, &podList,
		ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName),
		ctrlruntimeclient.MatchingLabels{resources.AppLabelKey: "apiserver"},
	); err != nil {
		return false, err
	}

	if len(podList.Items) == 0 {
		return false, nil
	}

	for _, pod := range podList.Items {
		if val, ok := pod.Labels[encryption.ApiserverEncryptionRevisionLabelKey]; !ok || val != secret.ResourceVersion {
			return false, nil
		}
	}

	return true, nil
}

func clusterNamespace(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s", cluster.Name)
}

func (r *runner) ensureDataEncryption(
	ctx context.Context,
	cluster *kubermaticv1.Cluster,
	secret corev1.Secret,
	config *rest.Config,
	shouldBeEncrypted bool,
	keyName string,
) error {
	// k8s:enc:secretbox:v1:encryption-key-2025-04
	regexPattern := fmt.Sprintf(`"Value"\s*:\s*"k8s:enc:secretbox:v1:%s:`, keyName)
	r.logger.Infof(
		"check if the secret data is encrypted with specific key, keyName: %s, secret: %s",
		keyName, ctrlruntimeclient.ObjectKeyFromObject(&secret).String(),
	)

	reg := regexp.MustCompile(regexPattern)

	err := wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout*2,
		func(ctx context.Context) (transient error, terminal error) {
			stdout, stderr, err := podexec.ExecuteCommand(
				ctx,
				config,
				types.NamespacedName{
					Namespace: clusterNamespace(cluster),
					Name:      "etcd-0",
				},
				"etcd",
				"etcdctl",
				"get",
				fmt.Sprintf("/registry/secrets/%s/%s", secret.Namespace, secret.Name),
				"-w", "fields",
			)
			if err != nil {
				return fmt.Errorf("failed to get data from etcd (stdout=%s, stderr=%s): %w", stdout, stderr, err), nil
			}
			if stderr != "" {
				return fmt.Errorf("failed to get data from etcd (stdout=%s, stderr=%s)", stdout, stderr), nil
			}

			r.logger.Infof("stdout from etcdctl: %s", stdout)

			encrypted := reg.MatchString(stdout)
			if encrypted == shouldBeEncrypted {
				return nil, nil
			}

			return fmt.Errorf(
				"etcd encryption at rest is not working as expected, got %v, expected %v",
				encrypted,
				shouldBeEncrypted,
			), nil
		},
	)
	return err
}

const (
	encKeyVal  = "usPvwsI/cx3EHynJAeX5WZFfUYE84LckhiOBvnnZASo="
	encKeyName = "encryption-key-2025-04"

	rotatedKeyVal  = "F7THAMOu8QCRl2R7JHyS83lMVLwSf8zdBAVzv2p+22k="
	rotatedKeyName = "encryption-key-2025-05"
)

func (r *runner) enableEAR(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("enabling encryption-at-rest")

	cc := cluster.DeepCopy()
	cluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest] = true
	cluster.Spec.EncryptionConfiguration = &kubermaticv1.EncryptionConfiguration{
		Enabled:   true,
		Resources: []string{"secrets"},
		Secretbox: &kubermaticv1.SecretboxEncryptionConfiguration{
			Keys: []kubermaticv1.SecretboxKey{
				{
					Name:  encKeyName,
					Value: encKeyVal,
				},
			},
		},
	}

	err := r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(cc))
	if err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	err = r.waitForClusterEncryption(ctx, cluster)
	if err != nil {
		return err
	}

	return nil
}

func (r *runner) waitForClusterEncryption(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("Waiting for cluster to healthy")

	err := r.testJig.WaitForHealthyControlPlane(ctx, defaultTimeout)
	if err != nil {
		return fmt.Errorf("Cluster did not get healthy after enabling encryption-at-rest: %w", err)
	}

	// wait for cluster.status.encryption.phase to be active and status.condition contains
	// condition EncryptionControllerReconciledSuccessfully with status true, and
	// condition EncryptionInitialized with status true.
	err = wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			if err := r.clusterEncryptionInitialized(ctx, cluster); err != nil {
				return err, nil
			}

			r.logger.Info("cluster status is updated as expected after enabling encryption-at-rest")
			return nil, nil
		},
	)
	if err != nil {
		return err
	}

	err = r.ensureAPIServerUpdated(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to ensure apiserver is updated: %w", err)
	}

	err = r.encryptionJobFinishedSuccessfully(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *runner) clusterEncryptionInitialized(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if cluster.Status.Encryption == nil {
		return fmt.Errorf("cluster.status.encryption is nil")
	}

	if cluster.Status.Encryption.Phase != kubermaticv1.ClusterEncryptionPhaseActive {
		return fmt.Errorf("cluster.status.encryption.phase is not active")
	}

	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess, corev1.ConditionTrue) {
		return fmt.Errorf("condition %s is not set yet", kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess)
	}

	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue) {
		return fmt.Errorf("condition %s is not set yet", kubermaticv1.ClusterConditionEncryptionInitialized)
	}

	return nil
}

func (r *runner) disableEAR(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("disabling encryption-at-rest")

	cc := cluster.DeepCopy()
	cluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest] = false
	cluster.Spec.EncryptionConfiguration = nil

	err := r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(cc))
	if err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	r.logger.Info("Waiting for cluster to healthy after disabling encryption-at-rest")
	if err := r.testJig.WaitForHealthyControlPlane(ctx, defaultTimeout); err != nil {
		return fmt.Errorf("Cluster did not get healthy after disabling encryption-at-rest: %w", err)
	}

	// wait for cluster.status.encryption is nil and status.condition contains
	// condition EncryptionControllerReconciledSuccessfully with status true, and
	// condition EncryptionInitialized with status 'false'.
	r.logger.Info("waiting for cluster status to be updated after disabling encryption-at-rest")

	err = wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			c := cluster.DeepCopy()
			if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(c), c); err != nil {
				return fmt.Errorf("failed to get cluster: %w", err), nil
			}

			if c.Status.Encryption != nil {
				return fmt.Errorf("cluster.status.encryption is not nil"), nil
			}

			if !c.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess, corev1.ConditionTrue) {
				return fmt.Errorf("condition %s is not set to true", kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess), nil
			}

			if !c.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionFalse) {
				return fmt.Errorf("condition %s is not set to false", kubermaticv1.ClusterConditionEncryptionInitialized), nil
			}

			return nil, nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to wait for cluster to healthy after disabling encryption-at-rest: %w", err)
	}

	return nil
}

func (r *runner) encryptionJobFinishedSuccessfully(ctx context.Context) error {
	r.logger.Info("waiting for the data-encryption job to finish successfully")
	err := wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			jobList := &batchv1.JobList{}
			err := r.seedClient.List(ctx, jobList, ctrlruntimeclient.MatchingLabels{
				resources.AppLabelKey: encryption.AppLabelValue,
			})
			if err != nil {
				return fmt.Errorf("failed to list jobs: %w", err), nil
			}

			if len(jobList.Items) == 0 {
				return fmt.Errorf(
					"no jobs found with label '%s: %s'", resources.AppLabelKey, encryption.AppLabelValue,
				), nil
			}

			job := jobList.Items[0]
			if len(job.Status.Conditions) == 0 {
				return fmt.Errorf("job status is not updated yet"), nil
			}

			expectedCompletions := int32(1)
			if job.Spec.Completions != nil {
				expectedCompletions = *job.Spec.Completions
			}

			if job.Status.Succeeded < expectedCompletions {
				return fmt.Errorf(
					"job pod is not succeeded yet, conditions: %+v", job.Status.Conditions,
				), nil
			}

			return nil, nil
		},
	)
	if err != nil {
		return err
	}

	r.logger.Info("data-encryption job finished successfully")
	return nil
}

func (r *runner) rotateEncryptionKey(
	ctx context.Context,
	cluster *kubermaticv1.Cluster,
	secretEncryptedWithOriginalKey corev1.Secret,
	postBackupSecret corev1.Secret,
) error {
	err := r.addSecondaryKey(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to add secondary key: %w", err)
	}

	err = r.verifyDataAccess(ctx, cluster, secretEncryptedWithOriginalKey, encKeyName)
	if err != nil {
		return fmt.Errorf("verification failed after adding secondary key: %w", err)
	}

	err = r.updatePrimaryEncryptionKey(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to update primary encryption key: %w", err)
	}

	// after rotating the key, the secret should be encrypted with the new primary key
	err = r.verifyDataAccess(ctx, cluster, secretEncryptedWithOriginalKey, rotatedKeyName)
	if err != nil {
		return fmt.Errorf("failed to access data with the new primary key: %w", err)
	}

	err = r.verifyDataAccess(ctx, cluster, postBackupSecret, rotatedKeyName)
	if err != nil {
		return fmt.Errorf("failed to access data with new primary key: %w", err)
	}

	r.logger.Info("key rotation completed successfully - keeping old key as secondary for backup/restore testing")
	return nil
}

func (r *runner) addSecondaryKey(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("adding secondary key")

	err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	cc := cluster.DeepCopy()
	cluster.Spec.EncryptionConfiguration.Secretbox.Keys = []kubermaticv1.SecretboxKey{
		{
			Name:  encKeyName,
			Value: encKeyVal,
		},
		{
			Name:  rotatedKeyName,
			Value: rotatedKeyVal,
		},
	}

	err = r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(cc))
	if err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	err = r.waitForClusterEncryption(ctx, cluster)
	if err != nil {
		return fmt.Errorf("cluster did not get healthy after adding new key: %w", err)
	}

	r.logger.Info("secondary key added successfully")
	return nil
}

func (r *runner) updatePrimaryEncryptionKey(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("updating primary encryption key")

	err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	cc := cluster.DeepCopy()
	cluster.Spec.EncryptionConfiguration.Secretbox.Keys = []kubermaticv1.SecretboxKey{
		{
			Name:  rotatedKeyName,
			Value: rotatedKeyVal,
		},
		{
			Name:  encKeyName,
			Value: encKeyVal,
		},
	}

	err = r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(cc))
	if err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	err = r.waitForClusterEncryption(ctx, cluster)
	if err != nil {
		return fmt.Errorf("cluster did not get healthy after swapping keys: %w", err)
	}

	return nil
}

func (r *runner) verifyDataAccess(ctx context.Context, cluster *kubermaticv1.Cluster, secret corev1.Secret, expectedKeyName string) error {
	r.logger.Infof(
		"verifying data access, expectedKeyName: %s, secret: %s",
		expectedKeyName,
		ctrlruntimeclient.ObjectKeyFromObject(&secret).String(),
	)

	err := r.ensureDataAccessible(ctx, secret.Name, secret.Namespace)
	if err != nil {
		return fmt.Errorf("failed to access data: %w", err)
	}

	err = r.ensureDataEncryption(ctx, cluster, secret, r.config, true, expectedKeyName)
	if err != nil {
		return fmt.Errorf("data not encrypted with expected key %s: %w", expectedKeyName, err)
	}

	r.logger.Info("data verified successfully")
	return nil
}

func (r *runner) createEtcdBackup(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("creating one-time etcd backup")

	backupConfig := kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ear-test-backup",
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

	err := r.seedClient.Create(ctx, &backupConfig)
	if err != nil {
		return fmt.Errorf("failed to create etcd backup config: %w", err)
	}

	r.logger.Info("waiting for backup to be completed")
	err = wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout*2,
		func(ctx context.Context) (transient error, terminal error) {
			if err := r.seedClient.Get(ctx, types.NamespacedName{
				Name:      backupConfig.Name,
				Namespace: backupConfig.Namespace,
			}, &backupConfig); err != nil {
				return fmt.Errorf("failed to get backup config: %w", err), nil
			}

			if len(backupConfig.Status.CurrentBackups) == 0 {
				return fmt.Errorf("no backups listed in status yet"), nil
			}

			for _, backup := range backupConfig.Status.CurrentBackups {
				if backup.BackupPhase == kubermaticv1.BackupStatusPhaseCompleted {
					r.logger.Info("backup completed successfully")
					return nil, nil
				}

				r.logger.Infof("backup not completed yet, backupName: %s, backupPhase: %s", backup.BackupName, backup.BackupPhase)
			}

			return fmt.Errorf("backup not completed yet, status: %+v", backupConfig.Status), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to wait for backup completion: %w", err)
	}

	r.logger.Info("etcd backup created successfully")
	return nil
}

func (r *runner) restoreEtcdBackup(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	r.logger.Info("restoring etcd from backup")

	var backupConfig kubermaticv1.EtcdBackupConfig
	if err := r.seedClient.Get(ctx, types.NamespacedName{
		Name:      "ear-test-backup",
		Namespace: cluster.Status.NamespaceName,
	}, &backupConfig); err != nil {
		return fmt.Errorf("failed to get backup config: %w", err)
	}

	if len(backupConfig.Status.CurrentBackups) == 0 {
		return fmt.Errorf("no backups found in backup config status")
	}

	var latestBackupName string
	var latestBackupTime time.Time
	for _, backup := range backupConfig.Status.CurrentBackups {
		if backup.BackupPhase == kubermaticv1.BackupStatusPhaseCompleted {
			if latestBackupName == "" || backup.ScheduledTime.After(latestBackupTime) {
				latestBackupName = backup.BackupName
				latestBackupTime = backup.ScheduledTime.Time
			}
		}
	}

	if latestBackupName == "" {
		return fmt.Errorf("no completed backups found for restoration")
	}

	r.logger.Infof("found backup for restoration: %s", latestBackupName)

	restore := &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ear-test-restore",
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.EtcdRestoreSpec{
			Cluster: corev1.ObjectReference{
				Name:       cluster.Name,
				APIVersion: cluster.APIVersion,
				Kind:       cluster.Kind,
			},
			BackupName:  latestBackupName,
			Destination: minioBackupDestination,
		},
	}

	err := r.seedClient.Create(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to create etcd restore: %w", err)
	}

	err = r.waitForEtcdRestore(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to wait for etcd restore: %w", err)
	}

	err = r.waitForClusterEncryption(ctx, cluster)
	if err != nil {
		return fmt.Errorf("cluster did not become healthy after restore: %w", err)
	}

	return nil
}

func (r *runner) waitForEtcdRestore(ctx context.Context, restore *kubermaticv1.EtcdRestore) error {
	r.logger.Info("waiting for etcd restore to complete")

	before := time.Now()
	if err := wait.PollImmediateLog(ctx, r.logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := r.seedClient.Get(ctx, types.NamespacedName{Name: restore.Name, Namespace: restore.Namespace}, restore); err != nil {
			return fmt.Errorf("failed to get restore status: %w", err), nil
		}

		if restore.Status.Phase == kubermaticv1.EtcdRestorePhaseCompleted {
			return nil, nil
		}

		return fmt.Errorf("restore in progress, current phase: %s", restore.Status.Phase), nil
	}); err != nil {
		return fmt.Errorf("failed waiting for restore to complete: %w (%v)", err, restore.Status)
	}

	r.logger.Infof("etcd restore finished after %v.", time.Since(before))
	return nil
}

func (r *runner) ensureDataAccessible(ctx context.Context, secretName, namespace string) error {
	return wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			var secret corev1.Secret
			err := r.userClient.Get(ctx, types.NamespacedName{
				Name:      secretName,
				Namespace: namespace,
			}, &secret)
			if err != nil {
				return fmt.Errorf("failed to get secret: %w", err), nil
			}

			return nil, nil
		},
	)
}

func (r *runner) verifySecretDoesNotExist(ctx context.Context, secretName, namespace string) error {
	r.logger.Infof(
		"verifying secret does not exist, secretName: %s, namespace: %s",
		secretName, namespace,
	)

	err := wait.PollImmediateLog(
		ctx,
		r.logger,
		defaultInterval,
		defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			var secret corev1.Secret
			err := r.userClient.Get(ctx, types.NamespacedName{
				Name:      secretName,
				Namespace: namespace,
			}, &secret)
			if err == nil {
				return fmt.Errorf("secret still exists when it shouldn't"), nil
			}

			if ctrlruntimeclient.IgnoreNotFound(err) == nil {
				return nil, nil
			}

			return fmt.Errorf("failed to check if secret exists: %w", err), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to verify secret does not exist: %w", err)
	}

	r.logger.Info("secret does not exist as expected")
	return nil
}
