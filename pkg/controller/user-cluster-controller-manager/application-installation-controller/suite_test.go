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

package applicationinstallationcontroller

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/fake"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var testEnv *envtest.Environment
var userClient ctrlruntimeclient.Client
var ctx context.Context
var cancel context.CancelFunc
var applicationInstallerRecorder fake.ApplicationInstallerRecorder

func TestApplicationInstallerController(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Application Installation controller test suite")
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

	userClient, err = ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(userClient).ToNot(BeNil())

	applicationInstallerRecorder = fake.ApplicationInstallerRecorder{}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationNamespace,
		},
	}
	Expect(userClient.Create(ctx, ns)).To(Succeed())

	err = Add(ctx, kubermaticlog.Logger, mgr, mgr, func(ctx context.Context) (bool, error) {
		return false, nil
	}, &applicationInstallerRecorder)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

}, 60)

var _ = AfterSuite(func() {
	// stop controller
	cleanupAppsNamespace()
	cancel()
	By("tearing down the test environment")

	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanupAppsNamespace() {
	ns := &corev1.Namespace{}
	err := userClient.Get(ctx, types.NamespacedName{Name: applicationNamespace}, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		Fail(err.Error())
	}

	// Delete ns if it exists
	err = userClient.Delete(ctx, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		Fail(err.Error())
	}
}
