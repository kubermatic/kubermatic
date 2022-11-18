/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package test

import (
	"context"
	"path"
	"testing"
	"time"

	"k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// CreateNamespaceWithCleanup creates a namespace with a generated name test-<something> and registers a hook on T.Cleanup
// that removes it at the end of the test.
func CreateNamespaceWithCleanup(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}
	if err := client.Create(ctx, ns); err != nil {
		t.Fatalf("failed to create test namespace: %s", err)
	}

	t.Cleanup(func() {
		if err := client.Delete(ctx, ns); err != nil {
			t.Fatalf("failed to cleanup test namespace: %s", err)
		}
	})

	if !utils.WaitFor(time.Second*1, time.Second*10, func() bool {
		namespace := &corev1.Namespace{}
		return client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(ns), namespace) == nil
	}) {
		t.Fatalf("timeout waiting for namespace creation")
	}
	return ns
}

// StartTestEnvWithCleanup bootstraps the testing environment and return the to the kubeconfig. It also registers a hook on T.Cleanup to
// stop the env and remove the kubeconfig.
func StartTestEnvWithCleanup(t *testing.T) (context.Context, ctrlruntimeclient.Client, string) {
	ctx, cancel := context.WithCancel(context.Background())
	log.Logger = log.New(true, log.FormatJSON).Sugar()

	// Bootstrapping test environment.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start envTest: %s", err)
	}

	if err := v1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to add kubermaticv1 scheme: %s", err)
	}

	mgr, err := controllerruntime.NewManager(cfg, controllerruntime.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		t.Fatalf("failed to create manager: %s", err)
	}

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("failed to start manager: %s", err)
			return
		}
	}()

	t.Cleanup(func() {
		// Clean up and stop controller.
		cancel()

		// Tearing down the test environment.
		if err := testEnv.Stop(); err != nil {
			t.Fatalf("failed to stop testEnv: %s", err)
		}
	})

	// get envTest kubeconfig
	kubeconfig := *api.NewConfig()
	kubeconfig.Clusters["testenv"] = &api.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}
	kubeconfig.CurrentContext = "default-context"
	kubeconfig.Contexts["default-context"] = &api.Context{
		Cluster:   "testenv",
		Namespace: "default",
		AuthInfo:  "auth-info",
	}
	kubeconfig.AuthInfos["auth-info"] = &api.AuthInfo{ClientCertificateData: cfg.CertData, ClientKeyData: cfg.KeyData}

	kubeconfigPath := path.Join(t.TempDir(), "testEnv-kubeconfig")
	err = clientcmd.WriteToFile(kubeconfig, kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to write testEnv kubeconfig: %s", err)
	}

	return ctx, client, kubeconfigPath
}

// CheckConfigMap asserts that configMap deployed by helm chart has been created/ updated with the desired data and labels.
func CheckConfigMap(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, expectedData map[string]string, expectedVersionLabel string) {
	t.Helper()

	cm := &corev1.ConfigMap{}
	var errorGetCm error
	if !utils.WaitFor(time.Second*1, time.Second*10, func() bool {
		errorGetCm = client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: ConfigmapName}, cm)
		// if config map has not been  updated
		if errorGetCm != nil || !diff.SemanticallyEqual(expectedData, cm.Data) || cm.Labels[VersionLabelKey] != expectedVersionLabel {
			return false
		}
		return true
	}) {
		if errorGetCm != nil {
			t.Fatalf("failed to get configMap: %s", errorGetCm)
		}
		// test object values are merged
		if !diff.SemanticallyEqual(expectedData, cm.Data) {
			t.Errorf("ConfigMap.Data differs from expected:\n%v", diff.ObjectDiff(expectedData, cm.Data))
		}

		// test scalar values are overiwritten
		if cm.Labels[VersionLabelKey] != expectedVersionLabel {
			t.Errorf("ConfigMap versionLabel has invalid value. expected '%s', got '%s'", expectedVersionLabel, cm.Labels[VersionLabelKey])
		}
	}
}
