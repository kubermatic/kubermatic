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

package kubermaticmaster

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeHelmClient struct {
	uninstallCalls []struct{ namespace, name string }
}

func (f *fakeHelmClient) BuildChartDependencies(string, []string) error { return nil }
func (f *fakeHelmClient) InstallChart(string, string, string, string, map[string]string, []string) error {
	return nil
}
func (f *fakeHelmClient) GetRelease(string, string) (*helm.Release, error) { return nil, nil }
func (f *fakeHelmClient) ListReleases(string) ([]helm.Release, error)      { return nil, nil }
func (f *fakeHelmClient) UninstallRelease(namespace, name string) error {
	f.uninstallCalls = append(f.uninstallCalls, struct{ namespace, name string }{namespace, name})
	return nil
}
func (f *fakeHelmClient) RenderChart(string, string, string, string, map[string]string, []string) ([]byte, error) {
	return nil, nil
}
func (f *fakeHelmClient) GetValues(string, string) (*yamled.Document, error) { return nil, nil }

func nginxNamespaceObject() *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: common.NginxIngressControllerNamespace}}
}

func TestCleanupLegacyNginxIngressFlagSetUninstallsRelease(t *testing.T) {
	helmClient := &fakeHelmClient{}
	kubeClient := fake.NewClientBuilder().WithObjects(nginxNamespaceObject()).Build()

	opt := stack.DeployOptions{CleanNginxLB: true}
	if err := cleanupLegacyNginxIngress(context.Background(), logrus.NewEntry(logrus.New()), kubeClient, helmClient, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(helmClient.uninstallCalls); got != 1 {
		t.Fatalf("expected 1 helm uninstall call, got %d", got)
	}
	call := helmClient.uninstallCalls[0]
	if call.namespace != common.NginxIngressControllerNamespace || call.name != common.NginxIngressControllerReleaseName {
		t.Fatalf("unexpected helm uninstall args: %+v", call)
	}
}

func TestCleanupLegacyNginxIngressFlagSetNoopWhenAbsent(t *testing.T) {
	helmClient := &fakeHelmClient{}
	kubeClient := fake.NewClientBuilder().Build()

	opt := stack.DeployOptions{CleanNginxLB: true}
	if err := cleanupLegacyNginxIngress(context.Background(), logrus.NewEntry(logrus.New()), kubeClient, helmClient, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(helmClient.uninstallCalls); got != 0 {
		t.Fatalf("expected no helm uninstall calls when nginx namespace is absent, got %d", got)
	}
}

func TestCleanupLegacyNginxIngressFlagUnsetWarnsWhenNamespaceExists(t *testing.T) {
	helmClient := &fakeHelmClient{}
	kubeClient := fake.NewClientBuilder().WithObjects(nginxNamespaceObject()).Build()
	hook := &warnHook{}
	logger := logrus.New()
	logger.AddHook(hook)

	opt := stack.DeployOptions{CleanNginxLB: false}
	if err := cleanupLegacyNginxIngress(context.Background(), logrus.NewEntry(logger), kubeClient, helmClient, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(helmClient.uninstallCalls); got != 0 {
		t.Fatalf("expected no helm uninstall calls when --clean-nginx-lb is unset, got %d", got)
	}
	if hook.warnings != 1 {
		t.Fatalf("expected 1 warning log entry, got %d", hook.warnings)
	}

	// namespace is left in place
	ns := &corev1.Namespace{}
	if err := kubeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: common.NginxIngressControllerNamespace}, ns); err != nil {
		t.Fatalf("expected nginx namespace to remain, got error: %v", err)
	}
}

func TestCleanupLegacyNginxIngressFlagUnsetNoopWhenNamespaceAbsent(t *testing.T) {
	helmClient := &fakeHelmClient{}
	kubeClient := fake.NewClientBuilder().Build()
	hook := &warnHook{}
	logger := logrus.New()
	logger.AddHook(hook)

	opt := stack.DeployOptions{CleanNginxLB: false}
	if err := cleanupLegacyNginxIngress(context.Background(), logrus.NewEntry(logger), kubeClient, helmClient, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hook.warnings != 0 {
		t.Fatalf("expected no warnings when nginx namespace is absent, got %d", hook.warnings)
	}
	if got := len(helmClient.uninstallCalls); got != 0 {
		t.Fatalf("expected no helm uninstall calls, got %d", got)
	}
}

type warnHook struct {
	warnings int
}

func (h *warnHook) Levels() []logrus.Level { return []logrus.Level{logrus.WarnLevel} }
func (h *warnHook) Fire(*logrus.Entry) error {
	h.warnings++
	return nil
}
