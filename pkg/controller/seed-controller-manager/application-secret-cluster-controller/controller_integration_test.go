//go:build integration

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package applicationsecretclustercontroller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	applicationsecretsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/application-secret-synchronizer"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	workerLabel = "myworker"
	timeout     = time.Second * 10
	interval    = time.Second * 1
)

// clusterWithWorkerName is a cluster with wokername=workerLabel. Application Secret should be synced in this cluster's namespace.
var clusterWithWorkerName *kubermaticv1.Cluster

// pausedClusterWithWorkerName is a cluster with wokername=workerLabel in pause state. Application Secret should NOT be synced in this cluster's namespace.
var pausedClusterWithWorkerName *kubermaticv1.Cluster

// clusterWithoutWorkerName is a cluster with wokername="". Application Secret should NOT be synced in this cluster's namespace.
var clusterWithoutWorkerName *kubermaticv1.Cluster

// kubermaticNS is the namespace where kubermatic is installed; therefore the namespace where Application Secrets live.
var kubermaticNS *corev1.Namespace

func Test_reconciler_reconcile(t *testing.T) {
	ctx := context.Background()
	client := startTestEnvWithClusters(t, ctx)

	tests := []struct {
		name     string
		testFunc func(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client)
	}{
		{
			name:     "when an application secret is created it should be created into cluster namespace",
			testFunc: secretCreationTest,
		},
		{
			name:     "when an application secret is updated it should be updated into cluster namespace",
			testFunc: secretUpdatedTest,
		},
		{
			name:     "when an application secret is deleted it should be deleted into cluster namespace and kubermatic namespace",
			testFunc: secretIsDeletedTest,
		},
		{
			name:     "when cluster is being deleted, application secret should not be synced",
			testFunc: secretNotSyncWhenClusterBeingDeletedTest,
		},
		{
			name:     "non application secret (i.e. without annotation applicationsecretsynchronizer.SecretTypeAnnotatio) should not be synced",
			testFunc: nonApplicationSecretShouldNotBeSyncedTest,
		},
		{
			name:     "secret in another namespace than kubermatic should not be synced",
			testFunc: secretInAnotherNsThanKubermaticNotSyncTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, ctx, client)
		})
	}
}

func secretCreationTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-cred",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	if err := client.Create(ctx, secret); err != nil {
		t.Fatalf("failed to create secret: %s", err)
	}

	// Secret should be synced on running cluster with same workername than controller.
	expectSecretSync(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret)
	expectSecretHasFinalizer(t, ctx, client, secret)

	// Secret should not be synced on paused cluster and cluster with different workerName than controller.
	expectSecretNevertExist(t, ctx, client, pausedClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func secretUpdatedTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-cred",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
			Labels:       map[string]string{"foo": "bar"},
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	// Create secret and wait for secret to be synced for the first time.
	if err := client.Create(ctx, secret); err != nil {
		t.Fatalf("failed to create secret: %s", err)
	}
	expectSecretSync(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret)
	expectSecretNevertExist(t, ctx, client, pausedClusterWithWorkerName.Status.NamespaceName, secret.Name)

	// Update the secret.
	original := secret.DeepCopy()
	secret.Data = map[string][]byte{"pass": []byte("bG9vZHNlCg==")}
	secret.Labels["new"] = "val"
	if err := client.Patch(ctx, secret, ctrlruntimeclient.MergeFrom(original)); err != nil {
		t.Fatalf("failed to update secret")
	}

	// Secret should be synced on running cluster with same workername than controller.
	expectSecretSync(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret)
	expectSecretHasFinalizer(t, ctx, client, secret)

	// Secret should not be synced on paused cluster and cluster with different workerName than controller.
	expectSecretNevertExist(t, ctx, client, pausedClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func secretIsDeletedTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-cred",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
			Labels:       map[string]string{"foo": "bar"},
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	// Create secret and wait for secret to be synced for the first time.
	if err := client.Create(ctx, secret); err != nil {
		t.Fatalf("failed to create secret: %s", err)
	}
	expectSecretSync(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret)

	// Secret should not be synced on paused cluster and cluster with different workerName than controller.
	expectSecretNevertExist(t, ctx, client, pausedClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)

	// deleting Secret
	if err := client.Delete(ctx, secret); err != nil {
		t.Fatalf("failed to delete secrer: %s", err)
	}

	// Secret should be deleted from cluster's namespace and kubermatic namespaces (i.e. finalizer has been removed).
	expectSecretIsDeleted(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretIsDeleted(t, ctx, client, secret.Namespace, secret.Name)
}

func secretNotSyncWhenClusterBeingDeletedTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	// Create a cluster.
	cluster := createCluster(t, ctx, client, "deleting-cluster", workerLabel, false, []string{"something-to-keep-object"})
	defer func() {
		original := cluster.DeepCopy()
		cluster.Finalizers = []string{}
		if err := client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original)); err != nil {
			t.Fatalf("failed to delete cluster: %s", err)
		}
	}()

	if err := client.Delete(ctx, cluster); err != nil {
		t.Fatalf("failed to delete secret: %s", err)
	}

	// Wait for cluster to be in deleting state.
	var err error
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster); err != nil {
			return false
		}
		if cluster.DeletionTimestamp.IsZero() {
			err = fmt.Errorf("DeletionTimestamp of cluster is 0")
			return false
		}
		return true
	}) {
		t.Fatalf("cluster not in deleting state: %s", err)
	}

	// Create secret and expect it not synced.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-cred",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
			Labels:       map[string]string{"foo": "bar"},
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}
	if err := client.Create(ctx, secret); err != nil {
		t.Fatalf("failed to create secret: %s", err)
	}
	expectSecretNevertExist(t, ctx, client, cluster.Status.NamespaceName, secret.Name)
}

func nonApplicationSecretShouldNotBeSyncedTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "a-secret",
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	if err := client.Create(ctx, secret); err != nil {
		t.Fatalf("failed to create secret: %s", err)
	}

	expectSecretNevertExist(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, pausedClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func secretInAnotherNsThanKubermaticNotSyncTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "a-secret",
			Namespace:    "default",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	if err := client.Create(ctx, secret); err != nil {
		t.Fatalf("failed to create secret: %s", err)
	}

	expectSecretNevertExist(t, ctx, client, clusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, pausedClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(t, ctx, client, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func expectSecretSync(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, clusterNamespace string, expectedSecert *corev1.Secret) {
	syncedSecret := &corev1.Secret{}
	var err error
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err = client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: expectedSecert.Name}, syncedSecret); err != nil {
			return false
		}
		if !diff.DeepEqual(expectedSecert.Data, syncedSecret.Data) {
			err = fmt.Errorf("secret data differs from expected:\n%s", diff.ObjectDiff(expectedSecert.Data, syncedSecret.Data))
			return false
		}
		if !diff.DeepEqual(expectedSecert.Labels, syncedSecret.Labels) {
			err = fmt.Errorf("secret Labels differs from expected:\n%s", diff.ObjectDiff(expectedSecert.Labels, syncedSecret.Labels))
			return false
		}
		if !diff.DeepEqual(expectedSecert.Annotations, syncedSecret.Annotations) {
			err = fmt.Errorf("secret Annotations differs from expected:\n%s", diff.ObjectDiff(expectedSecert.Annotations, syncedSecret.Annotations))
			return false
		}
		return true
	}) {
		t.Fatalf("secret has not been synced: %s", err)
	}
}

func expectSecretNevertExist(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, clusterNamespace string, name string) {
	syncedSecret := &corev1.Secret{}
	// Consistently check secret does not exist.
	if utils.WaitFor(ctx, interval, timeout, func() bool {
		return client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, syncedSecret) == nil
	}) {
		t.Fatalf("secret should not have been created %v", syncedSecret)
	}
}

func expectSecretIsDeleted(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, clusterNamespace string, name string) {
	t.Helper()
	syncedSecret := &corev1.Secret{}
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		err := client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, syncedSecret)
		return err != nil && apierrors.IsNotFound(err)
	}) {
		t.Fatalf("secret has not been removed")
	}
}

func expectSecretHasFinalizer(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, secret *corev1.Secret) {
	t.Helper()
	currentSecret := &corev1.Secret{}
	var err error
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(secret), currentSecret); err != nil {
			return false
		}
		expectedFinalizers := []string{applicationSecretCleanupFinalizer}
		if !diff.DeepEqual(expectedFinalizers, currentSecret.Finalizers) {
			err = fmt.Errorf("finalizers differs from expected:\n%s", diff.ObjectDiff(expectedFinalizers, currentSecret.Finalizers))
			return false
		}
		return true
	}) {
		t.Fatalf("secret has not expected finalizers: %s", err)
	}
}

func startTestEnvWithClusters(t *testing.T, ctx context.Context) ctrlruntimeclient.Client {
	rawLog := kubermaticlog.New(true, kubermaticlog.FormatJSON)
	kubermaticlog.Logger = rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	// Bootstrapping test environment.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start envTest: %s", err)
	}

	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Fatalf("failed to stop testEnv: %s", err)
		}
	})

	if err := kubermaticv1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to add kubermaticv1 scheme: %s", err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		t.Fatalf("failed to create manager: %s", err)
	}

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	// Create kubermatic namespace.
	// Intentionally using another name than kubermatic to be sure code don't use 'kubermatic' hardcoded value.
	kubermaticNS = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abc",
		},
	}
	if err := client.Create(ctx, kubermaticNS); err != nil {
		t.Fatalf("failed to create namespace: %s", err)
	}

	clusterWithWorkerName = createCluster(t, ctx, client, "with-worker-name", workerLabel, false, []string{})
	pausedClusterWithWorkerName = createCluster(t, ctx, client, "paused-with-worker-name", workerLabel, true, []string{})
	clusterWithoutWorkerName = createCluster(t, ctx, client, "without-worker-name", "", false, []string{})

	if err := Add(ctx, mgr, kubermaticlog.Logger, 2, workerLabel, kubermaticNS.Name); err != nil {
		t.Fatalf("failed to add controller to manager: %s", err)
	}

	mgrCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	go func() {
		if err := mgr.Start(mgrCtx); err != nil {
			t.Errorf("failed to start manager: %s", err)
		}
	}()

	return client
}

func createCluster(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, clusterName string, workerLabel string, isPause bool, finalizers []string) *kubermaticv1.Cluster {
	t.Helper()
	cluster := generator.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
		cluster.Namespace = kubermaticNS.Name
		if workerLabel != "" {
			cluster.Labels[kubermaticv1.WorkerNameLabelKey] = workerLabel
		}

		cluster.Spec.Pause = isPause
		cluster.Finalizers = finalizers
	})

	// Create cluster.
	if err := client.Create(ctx, cluster); err != nil {
		t.Fatalf("failed to create cluster %s: %s", clusterName, err)
	}

	// Create operation wipe out status, so we update with needed fields.
	original := cluster.DeepCopy()
	cluster.Status.NamespaceName = kubernetes.NamespaceName(clusterName)
	if err := client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original)); err != nil {
		t.Fatalf("failed to update cluster status: %s", err)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Status.NamespaceName,
		},
	}

	if err := client.Create(ctx, ns); err != nil {
		t.Fatalf("failed to create cluster namespace %s: %s", cluster.Status.NamespaceName, err)
	}

	return cluster
}
