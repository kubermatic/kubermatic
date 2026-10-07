/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package common

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// fakeHelmClient is a minimal helm.Client that records UninstallRelease invocations
// and returns a stub Release from GetRelease when releaseExists is true. Setting
// releaseExists to false models a partial prior --clean-nginx-lb run in which the
// release was already uninstalled but the namespace lingered.
type fakeHelmClient struct {
	releaseExists  bool
	uninstallCalls []struct{ namespace, name string }
}

func (f *fakeHelmClient) BuildChartDependencies(string, []string) error { return nil }
func (f *fakeHelmClient) InstallChart(string, string, string, string, map[string]string, []string) error {
	return nil
}
func (f *fakeHelmClient) GetRelease(namespace, name string) (*helm.Release, error) {
	if !f.releaseExists {
		return nil, nil
	}
	return &helm.Release{Name: name, Namespace: namespace, Status: "deployed"}, nil
}
func (f *fakeHelmClient) ListReleases(string) ([]helm.Release, error) { return nil, nil }
func (f *fakeHelmClient) UninstallRelease(namespace, name string) error {
	f.uninstallCalls = append(f.uninstallCalls, struct{ namespace, name string }{namespace, name})
	return nil
}
func (f *fakeHelmClient) RenderChart(string, string, string, string, map[string]string, []string) ([]byte, error) {
	return nil, nil
}
func (f *fakeHelmClient) GetValues(string, string) (*yamled.Document, error) { return nil, nil }

func nginxNamespace() *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: NginxIngressControllerNamespace}}
}

func newFakeClient(objs ...ctrlruntimeclient.Object) ctrlruntimeclient.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func TestNginxIngressNamespaceExistsTrue(t *testing.T) {
	exists, err := NginxIngressNamespaceExists(context.Background(), newFakeClient(nginxNamespace()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected namespace to be reported as existing")
	}
}

func TestNginxIngressNamespaceExistsFalse(t *testing.T) {
	exists, err := NginxIngressNamespaceExists(context.Background(), newFakeClient())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected namespace to be reported as absent")
	}
}

func TestUninstallNginxIngressControllerNoopWhenNamespaceAbsent(t *testing.T) {
	helmClient := &fakeHelmClient{}
	kubeClient := newFakeClient()

	if err := UninstallNginxIngressController(context.Background(), logrus.NewEntry(logrus.New()), kubeClient, helmClient); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(helmClient.uninstallCalls) != 0 {
		t.Fatalf("expected no helm uninstall calls, got %v", helmClient.uninstallCalls)
	}
}

func TestUninstallNginxIngressControllerRemovesReleaseAndNamespace(t *testing.T) {
	ctx := context.Background()
	helmClient := &fakeHelmClient{releaseExists: true}
	kubeClient := newFakeClient(nginxNamespace())

	if err := UninstallNginxIngressController(ctx, logrus.NewEntry(logrus.New()), kubeClient, helmClient); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(helmClient.uninstallCalls); got != 1 {
		t.Fatalf("expected 1 helm uninstall call, got %d", got)
	}
	call := helmClient.uninstallCalls[0]
	if call.namespace != NginxIngressControllerNamespace || call.name != NginxIngressControllerReleaseName {
		t.Fatalf("unexpected helm uninstall args: %+v", call)
	}

	ns := &corev1.Namespace{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: NginxIngressControllerNamespace}, ns)
	if err == nil {
		// fake client does not honour Kubernetes finalizers, so the object must be either
		// gone or in a terminating state with a deletion timestamp.
		if ns.DeletionTimestamp == nil {
			t.Fatalf("expected namespace to be deleted, got live object %+v", ns)
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error checking namespace: %v", err)
	}
}

// TestUninstallNginxIngressControllerRemovesNamespaceWhenReleaseAlreadyGone covers
// the retry-after-partial-cleanup case: a previous --clean-nginx-lb run (or a manual
// `helm uninstall`) removed the release, but the namespace or its LoadBalancer Service
// stuck around. The next run must skip the helm uninstall (release not registered) and
// still delete the namespace.
func TestUninstallNginxIngressControllerRemovesNamespaceWhenReleaseAlreadyGone(t *testing.T) {
	ctx := context.Background()
	helmClient := &fakeHelmClient{releaseExists: false}
	kubeClient := newFakeClient(nginxNamespace())

	if err := UninstallNginxIngressController(ctx, logrus.NewEntry(logrus.New()), kubeClient, helmClient); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(helmClient.uninstallCalls); got != 0 {
		t.Fatalf("expected no helm uninstall call when release is not registered, got %d", got)
	}

	ns := &corev1.Namespace{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: NginxIngressControllerNamespace}, ns)
	if err == nil {
		if ns.DeletionTimestamp == nil {
			t.Fatalf("expected lingering namespace to be deleted, got live object %+v", ns)
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error checking namespace: %v", err)
	}
}
