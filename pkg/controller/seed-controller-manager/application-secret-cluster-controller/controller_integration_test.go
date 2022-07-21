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

	"github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	applicationsecretsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/application-secret-synchronizer"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	workerLabel = "myworker"
	timeout     = time.Second * 10
	interval    = time.Second * 1
)

var testEnv *envtest.Environment
var client ctrlruntimeclient.Client
var ctx context.Context
var cancel context.CancelFunc

// clusterWithWorkerName is a cluster with wokername=workerLabel. Application Secret should be synced in this cluster's namespace.
var clusterWithWorkerName *kubermaticv1.Cluster

// pauseClusterWithWorkerName is a cluster with wokername=workerLabel in pause state. Application Secret should NOT be synced in this cluster's namespace.
var pauseClusterWithWorkerName *kubermaticv1.Cluster

// clusterWithoutWorkerName is a cluster with wokername="". Application Secret should NOT be synced in this cluster's namespace.
var clusterWithoutWorkerName *kubermaticv1.Cluster

// kubermaticNS is the namespace where kubermatic is installed; therefore the namespace where  Application Secrets live.
var kubermaticNS *corev1.Namespace

func Test_reconciler_reconcile(t *testing.T) {
	g := gomega.NewWithT(t)

	startTestEnvWithClusters(g)
	defer stopTestEnv(g)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name:     "when an application secret is created it should be created into cluster namespace",
			testFunc: secretCreationTest,
		},
		{
			name:     "when an application secret is update it should be updated into cluster namespace",
			testFunc: secretUpdatedTest,
		},
		{
			name:     "when an application secret is deleted it should be delete into cluster namespace and kubermatic namespace",
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
		t.Run(tt.name, tt.testFunc)
	}
}

func secretCreationTest(t *testing.T) {
	g := gomega.NewWithT(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-cred",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	g.Expect(client.Create(ctx, secret)).To(gomega.Succeed())

	// Secret should be synced on running cluster with same workername.
	expectSecretSync(g, clusterWithWorkerName.Status.NamespaceName, secret)
	expectSecretHasFinalizer(g, secret)

	// Secret should not be synced on paused cluster and cluster with different workerName.
	expectSecretNevertExist(g, pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func secretUpdatedTest(t *testing.T) {
	g := gomega.NewWithT(t)

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
	g.Expect(client.Create(ctx, secret)).To(gomega.Succeed())
	expectSecretSync(g, clusterWithWorkerName.Status.NamespaceName, secret)
	expectSecretNevertExist(g, pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)

	// Update the secret
	original := secret.DeepCopy()
	secret.Data = map[string][]byte{"pass": []byte("bG9vZHNlCg==")}
	secret.Labels["new"] = "val"
	g.Expect(client.Patch(ctx, secret, ctrlruntimeclient.MergeFrom(original))).To(gomega.Succeed())

	// Secret should be synced on running cluster with same workername.
	expectSecretSync(g, clusterWithWorkerName.Status.NamespaceName, secret)
	expectSecretHasFinalizer(g, secret)

	// Secret should not be synced on paused cluster and cluster with different workerName.
	expectSecretNevertExist(g, pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func secretIsDeletedTest(t *testing.T) {
	g := gomega.NewWithT(t)

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
	g.Expect(client.Create(ctx, secret)).To(gomega.Succeed())
	expectSecretSync(g, clusterWithWorkerName.Status.NamespaceName, secret)

	// Secret should not be synced on paused cluster and cluster with different workerName.
	expectSecretNevertExist(g, pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)

	// deleting Secret
	g.Expect(client.Delete(ctx, secret)).To(gomega.Succeed())

	// secret is deleted from cluster namespace and kubermatic namespaces (i.e. finalizer has been removed)
	expectSecretIsDeleted(g, clusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretIsDeleted(g, secret.Namespace, secret.Name)
}

func secretNotSyncWhenClusterBeingDeletedTest(t *testing.T) {
	g := gomega.NewWithT(t)

	// create a cluster
	cluster := createCluster(g, ctx, client, "deleting-cluster", workerLabel, false, []string{"something-to-keep-object"})
	defer func() {
		original := cluster.DeepCopy()
		cluster.Finalizers = []string{}
		g.Expect(client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original))).To(gomega.Succeed())
	}()

	g.Expect(client.Delete(ctx, cluster)).To(gomega.Succeed())

	// wait for cluster to be in deleting state
	g.Eventually(func() error {
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster); err != nil {
			return err
		}
		if cluster.DeletionTimestamp.IsZero() {
			return fmt.Errorf("cluster is not in deleting state")
		}
		return nil
	}, timeout, interval).Should(gomega.Succeed())

	// create secret and expect it not sync
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-cred",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
			Labels:       map[string]string{"foo": "bar"},
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}
	g.Expect(client.Create(ctx, secret)).To(gomega.Succeed())
	expectSecretNevertExist(g, cluster.Status.NamespaceName, secret.Name)
}

func nonApplicationSecretShouldNotBeSyncedTest(t *testing.T) {
	g := gomega.NewWithT(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "a-secret",
			Namespace:    kubermaticNS.Name,
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	g.Expect(client.Create(ctx, secret)).To(gomega.Succeed())

	expectSecretNevertExist(g, clusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func secretInAnotherNsThanKubermaticNotSyncTest(t *testing.T) {
	g := gomega.NewWithT(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "a-secret",
			Namespace:    "default",
			Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
		},
		Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
	}

	g.Expect(client.Create(ctx, secret)).To(gomega.Succeed())

	expectSecretNevertExist(g, clusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
	expectSecretNevertExist(g, clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
}

func expectSecretSync(g *gomega.WithT, clusterNamespace string, expectedSecert *corev1.Secret) {
	syncedSecret := &corev1.Secret{}
	g.EventuallyWithOffset(1, func() error {
		if err := client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: expectedSecert.Name}, syncedSecret); err != nil {
			return err
		}
		if !diff.DeepEqual(expectedSecert.Data, syncedSecret.Data) {
			return fmt.Errorf("secret data differs from expected:\n%s", diff.ObjectDiff(expectedSecert.Data, syncedSecret.Data))
		}
		if !diff.DeepEqual(expectedSecert.Labels, syncedSecret.Labels) {
			return fmt.Errorf("secret Labels differs from expected:\n%s", diff.ObjectDiff(expectedSecert.Labels, syncedSecret.Labels))
		}
		if !diff.DeepEqual(expectedSecert.Annotations, syncedSecret.Annotations) {
			return fmt.Errorf("secret Annotations differs from expected:\n%s", diff.ObjectDiff(expectedSecert.Annotations, syncedSecret.Annotations))
		}
		return nil
	}, timeout, interval).Should(gomega.Succeed())
}

func expectSecretNevertExist(g *gomega.WithT, clusterNamespace string, name string) {
	syncedSecret := &corev1.Secret{}
	g.ConsistentlyWithOffset(1, func() error {
		return client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, syncedSecret)
	}, timeout, interval).ShouldNot(gomega.Succeed())
}

func expectSecretIsDeleted(g *gomega.WithT, clusterNamespace string, name string) {
	syncedSecret := &corev1.Secret{}
	g.EventuallyWithOffset(1, func() error {
		return client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, syncedSecret)
	}, timeout, interval).ShouldNot(gomega.Succeed())
}

func expectSecretHasFinalizer(g *gomega.WithT, secret *corev1.Secret) {
	currentSecret := &corev1.Secret{}
	g.EventuallyWithOffset(1, func() error {
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(secret), currentSecret); err != nil {
			return err
		}
		expectedFinalizers := []string{applicationSecretCleanupFinalizer}
		if !diff.DeepEqual(expectedFinalizers, currentSecret.Finalizers) {
			return fmt.Errorf("secret finalizers differs from expected:\n%s", diff.ObjectDiff(expectedFinalizers, currentSecret.Finalizers))
		}
		return nil
	}, timeout, interval).Should(gomega.Succeed())
}

func startTestEnvWithClusters(g *gomega.WithT) {
	ctx, cancel = context.WithCancel(context.Background())

	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	// bbootstrapping test environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(cfg).NotTo(gomega.BeNil())

	err = kubermaticv1.AddToScheme(scheme.Scheme)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	client, err = ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(client).ToNot(gomega.BeNil())

	// Create kubermatic namespace.
	// Intentionally using another name than kubermatic to be sure code don't use 'kubermatic' hardcoded value.
	kubermaticNS = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abc",
		},
	}
	g.Expect(client.Create(ctx, kubermaticNS)).To(gomega.Succeed())

	clusterWithWorkerName = createCluster(g, ctx, client, "with-worker-name", workerLabel, false, []string{})
	pauseClusterWithWorkerName = createCluster(g, ctx, client, "paused-with-worker-name", workerLabel, true, []string{})
	clusterWithoutWorkerName = createCluster(g, ctx, client, "without-worker-name", "", false, []string{})

	err = Add(ctx, mgr, kubermaticlog.Logger, 2, workerLabel, kubermaticNS.Name)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	go func() {
		err = mgr.Start(ctx)
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}()
}

func stopTestEnv(g *gomega.WithT) {
	// clean up and stop controller
	cleanupClusterAndNs(g, clusterWithWorkerName)
	cleanupClusterAndNs(g, pauseClusterWithWorkerName)
	cleanupClusterAndNs(g, clusterWithoutWorkerName)
	cancel()

	// tearing down the test environment")
	err := testEnv.Stop()
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

func cleanupClusterAndNs(g *gomega.WithT, cluster *kubermaticv1.Cluster) {
	if cluster != nil {
		cleanupNamespace(g, cluster.Status.NamespaceName)

		currentCluster := &kubermaticv1.Cluster{}
		err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), currentCluster)
		if err != nil && !apierrors.IsNotFound(err) {
			g.Fail(err.Error())
		}

		// Delete cluster if it exists
		err = client.Delete(ctx, currentCluster)
		if err != nil && !apierrors.IsNotFound(err) {
			g.Fail(err.Error())
		}
	}
}

func cleanupNamespace(g *gomega.WithT, name string) {
	ns := &corev1.Namespace{}
	err := client.Get(ctx, types.NamespacedName{Name: name}, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		g.Fail(err.Error())
	}

	// Delete ns if it exists
	err = client.Delete(ctx, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		g.Fail(err.Error())
	}
}

func createCluster(g *gomega.WithT, ctx context.Context, client ctrlruntimeclient.Client, clusterName string, workerLabel string, isPause bool, finalizers []string) *kubermaticv1.Cluster {
	cluster := test.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
		cluster.Namespace = kubermaticNS.Name
		if workerLabel != "" {
			cluster.Labels[kubermaticv1.WorkerNameLabelKey] = workerLabel
		}

		cluster.Spec.Pause = isPause
		cluster.Finalizers = finalizers
	})
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Status.NamespaceName,
		},
	}

	g.ExpectWithOffset(1, client.Create(ctx, ns)).To(gomega.Succeed())

	// create cluster
	g.ExpectWithOffset(1, client.Create(ctx, cluster)).To(gomega.Succeed())

	// create wipe out status, so we update with needed field
	original := cluster.DeepCopy()
	cluster.Status.NamespaceName = kubernetes.NamespaceName(clusterName)
	g.ExpectWithOffset(1, client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original)))

	return cluster
}
