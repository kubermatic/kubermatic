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
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

const (
	defaultTimeout  = 5 * time.Minute
	defaultInterval = 10 * time.Second
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

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("encryption-at-rest")

	logger.Info("setting up the cluster")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
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

	// Test Case 1: Enable encryption at rest and verify it works
	logger.Info("Test Case 1: Enabling encryption-at-rest")
	err = r.enableEAR(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to enable encryption-at-rest: %v", err)
	}

	err = ensureApiServerUpdated(ctx, logger, seedClient, cluster)
	if err != nil {
		t.Fatalf("User cluster API server does not contain configurations for encryption-at-rest")
	}

	err = encryptionJobFinishedSuccessfully(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("data-encryption Job failed to run, err: %v", err)
	}

	err = ensureDataEncryption(ctx, logger, cluster, secret, config, true)
	if err != nil {
		t.Fatalf("failed to ensure data encryption: %v", err)
	}

	// Test Case 2: Disable encryption at rest and verify data is decrypted automatically.
	// The secret will be updated with 'kubectl replace' command by the Job running on the seed cluster.
	logger.Info("Test Case 2: Disabling encryption-at-rest")

	err = r.disableEAR(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to disable encryption-at-rest: %v", err)
	}

	err = ensureDataEncryption(ctx, logger, cluster, secret, config, false)
	if err != nil {
		t.Fatalf("failed to verify data is no longer encrypted: %v", err)
	}
}

func ensureApiServerUpdated(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	logger.Info("waiting for ApiServer to contain configurations for encryption-at-rest")

	err := wait.PollImmediateLog(
		ctx, logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			updated, err := isApiserverUpdated(ctx, client, cluster)
			if err != nil {
				return fmt.Errorf("failed to check apiserver status, %w", err), nil
			}

			if updated {
				logger.Info("apiserver is updated")
				return nil, nil
			}

			logger.Info("apiserver is not updated, retrying...")
			return fmt.Errorf("apiserver is not updated"), nil
		},
	)

	return err
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

	if val, ok := secret.ObjectMeta.Labels[encryption.ApiserverEncryptionHashLabelKey]; !ok || val != hex.EncodeToString(hash.Sum(nil)) {
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

func ensureDataEncryption(
	ctx context.Context,
	logger *zap.SugaredLogger,
	cluster *kubermaticv1.Cluster,
	secret corev1.Secret,
	config *rest.Config,
	shouldBeEncrypted bool,
) error {
	logger.Info("waiting to see if the data encryption works")

	r := regexp.MustCompile(`"Value"\s*:\s*"k8s:enc:secretbox`)

	err := wait.PollImmediateLog(
		ctx, logger, defaultInterval, defaultTimeout*3,
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

			logger.Info("stdout from etcdctl", "stdout", stdout)

			encrypted := r.MatchString(stdout)
			if encrypted == shouldBeEncrypted {
				return nil, nil
			}

			return fmt.Errorf("etcd encryption at rest is not working as expected, got %v, expected %v", encrypted, shouldBeEncrypted), nil
		},
	)
	return err
}

const (
	encKeyVal  = "usPvwsI/cx3EHynJAeX5WZFfUYE84LckhiOBvnnZASo="
	encKeyName = "encryption-key-2025-04"
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

	r.logger.Info("Waiting for cluster to healthy after enabling encryption-at-rest")
	if err := r.testJig.WaitForHealthyControlPlane(ctx, defaultTimeout); err != nil {
		return fmt.Errorf("Cluster did not get healthy after enabling encryption-at-rest: %w", err)
	}

	// wait for cluster.status.encryption.phase to be active and status.condition contains
	// condition EncryptionControllerReconciledSuccessfully with status true, and
	// condition EncryptionInitialized with status true.
	r.logger.Info("waiting for cluster status to be updated after enabling encryption-at-rest")

	err = wait.PollImmediateLog(
		ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			c := cluster.DeepCopy()
			if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(c), c); err != nil {
				return fmt.Errorf("failed to get cluster: %w", err), nil
			}

			if c.Status.Encryption == nil {
				r.logger.Info("cluster.status.encryption is still nil, retrying...")

				return fmt.Errorf("cluster.status.encryption is nil"), nil
			}

			if c.Status.Encryption.Phase != kubermaticv1.ClusterEncryptionPhaseActive {
				r.logger.Info("cluster.status.encryption.phase is not active, retrying...")

				return fmt.Errorf("cluster.status.encryption.phase is not active"), nil
			}

			if !c.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess, corev1.ConditionTrue) {
				r.logger.Info("condition %s is not set yet, retrying...", kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess)

				return fmt.Errorf("condition %s is not set yet", kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess), nil
			}

			if !c.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue) {
				r.logger.Info("condition %s is not set yet, retrying...", kubermaticv1.ClusterConditionEncryptionInitialized)

				return fmt.Errorf("condition %s is not set yet", kubermaticv1.ClusterConditionEncryptionInitialized), nil
			}

			r.logger.Info("cluster status is updated as expected after enabling encryption-at-rest")
			return nil, nil
		},
	)
	return err
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

func encryptionJobFinishedSuccessfully(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client) error {
	logger.Info("waiting for the data-encryption job to finish successfully")
	err := wait.PollImmediateLog(
		ctx, logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			jobList := &batchv1.JobList{}
			err := c.List(ctx, jobList, ctrlruntimeclient.MatchingLabels{
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

	return nil
}
