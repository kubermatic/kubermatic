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
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
	"regexp"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
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

func TestEncryptionAtRest(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	client, config, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	testJig := jig.NewAWSCluster(client, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("opa")

	logger.Info("setting up the cluster")
	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
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

	err = client.Create(ctx, &secret)
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	err = enableEAR(ctx, logger, client, cluster)
	if err != nil {
		t.Fatalf("failed to enable encryption-at-rest: %v", err)
	}

	logger.Info("Waiting for cluster to healthy after enabling encryption-at-rest")
	if err := testJig.WaitForHealthyControlPlane(ctx, defaultTimeout); err != nil {
		t.Fatalf("Cluster did not get healthy after enabling encryption-at-rest: %v", err)
	}

	err = ensureApiServerUpdated(ctx, logger, client, cluster)
	if err != nil {
		t.Fatalf("User cluster API server does not contain configurations for encryption-at-rest")
	}

	err = encryptionJobFinishedSuccessfully(ctx, logger, client)
	if err != nil {
		t.Fatalf("data-encryption Job failed to run, err: %v", err)
	}

	err = ensureDataEncryption(ctx, logger, cluster, secret, config)
	if err != nil {
		t.Fatalf("failed to ensure data encryption: %v", err)
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
) error {
	logger.Info("waiting to see if the data is encrypted")

	r := regexp.MustCompile(`"Value"\s*:\s*"k8s:enc:secretbox`)

	err := wait.PollImmediateLog(
		ctx, logger, defaultInterval, defaultTimeout*3,
		func(ctx context.Context) (transient error, terminal error) {
			stdout, stderr, err := podexec.ExecuteCommand(
				ctx,
				config,
				types.NamespacedName{
					Namespace: clusterNamespace(cluster),
					Name:      "etcd-0", // todo: grab this programmatically
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
			if encrypted {
				return nil, nil
			}

			return fmt.Errorf("the data in etcd is not encrypted, output %v", stdout), nil
		},
	)
	return err
}

const (
	encKeyVal  = "usPvwsI/cx3EHynJAeX5WZFfUYE84LckhiOBvnnZASo="
	encKeyName = "encryption-key-2025-04"
)

func enableEAR(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	logger.Info("enabling encryption-at-rest")

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

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(cc))
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

func disableEAR(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	cc := cluster.DeepCopy()
	cluster.Spec.EncryptionConfiguration = &kubermaticv1.EncryptionConfiguration{Enabled: false}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(cc))
}
