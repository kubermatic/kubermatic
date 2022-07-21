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
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

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
)

var testEnv *envtest.Environment
var client ctrlruntimeclient.Client
var ctx context.Context
var cancel context.CancelFunc

var clusterWithWorkerName *kubermaticv1.Cluster
var pauseClusterWithWorkerName *kubermaticv1.Cluster
var clusterWithoutWorkerName *kubermaticv1.Cluster
var kubermaticNS *corev1.Namespace

func TestApplicationSecretClusterController(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "kkp-application-secret-cluster-controller test suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kubermaticv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	client, err = ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(client).ToNot(BeNil())

	// Create kubermatic namespace.
	// Intentionally using another name than kubermatic to be sure code don't use 'kubermatic' hardcoded value.
	kubermaticNS = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abc",
		},
	}
	Expect(client.Create(ctx, kubermaticNS)).To(Succeed())

	clusterWithWorkerName = createCluster(ctx, client, "with-worker-name", workerLabel, false, []string{})
	pauseClusterWithWorkerName = createCluster(ctx, client, "paused-with-worker-name", workerLabel, true, []string{})
	clusterWithoutWorkerName = createCluster(ctx, client, "without-worker-name", "", false, []string{})

	err = Add(ctx, mgr, kubermaticlog.Logger, 2, workerLabel, kubermaticNS.Name) // todo const
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

}, 60)

var _ = AfterSuite(func() {
	// clean up and stop controller
	cleanupClusterAndNs(clusterWithWorkerName)
	cleanupClusterAndNs(pauseClusterWithWorkerName)
	cleanupClusterAndNs(clusterWithoutWorkerName)

	cancel()
	By("tearing down the test environment")

	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanupClusterAndNs(cluster *kubermaticv1.Cluster) {
	if cluster != nil {
		cleanupNamespace(cluster.Status.NamespaceName)

		currentCluster := &kubermaticv1.Cluster{}
		err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), currentCluster)
		if err != nil && !apierrors.IsNotFound(err) {
			Fail(err.Error())
		}

		// Delete cluster if it exists
		err = client.Delete(ctx, currentCluster)
		if err != nil && !apierrors.IsNotFound(err) {
			Fail(err.Error())
		}
	}
}

func cleanupNamespace(name string) {
	ns := &corev1.Namespace{}
	err := client.Get(ctx, types.NamespacedName{Name: name}, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		Fail(err.Error())
	}

	// Delete ns if it exists
	err = client.Delete(ctx, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		Fail(err.Error())
	}
}

func createCluster(ctx context.Context, client ctrlruntimeclient.Client, clusterName string, workerLabel string, isPause bool, finalizers []string) *kubermaticv1.Cluster {
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

	ExpectWithOffset(1, client.Create(ctx, ns)).To(Succeed())

	// create cluster
	ExpectWithOffset(1, client.Create(ctx, cluster)).To(Succeed())

	// create wipe out status, so we update with needed field
	original := cluster.DeepCopy()
	cluster.Status.NamespaceName = kubernetes.NamespaceName(clusterName)
	ExpectWithOffset(1, client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original)))

	return cluster
}
