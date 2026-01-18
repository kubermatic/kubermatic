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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	controllerruntime "sigs.k8s.io/controller-runtime"
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

	if !utils.WaitFor(ctx, time.Second*1, time.Second*10, func() bool {
		namespace := &corev1.Namespace{}
		return client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(ns), namespace) == nil
	}) {
		t.Fatalf("timeout waiting for namespace creation")
	}
	return ns
}

// EnsureClusterWithCleanup creates a cluster resource with a static name "default" and needed status fields for fetching it
// based on the namespacedname and registers a hook on T.Cleanup that removes it at the end of the test.
func EnsureClusterWithCleanup(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-default",
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: "humanreadable-cluster-name",
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				ProxyMode: "iptables",
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.0.0.0/24"},
				},
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.1.0.0/24"},
				},
			},
			ExposeStrategy: "NodePort",
		},
	}
	if err := client.Create(ctx, cluster); err != nil {
		t.Fatalf("failed to create test cluster: %v", err)
	}
	cluster.Status = kubermaticv1.ClusterStatus{
		UserEmail: "owner@email.com",
		Address: kubermaticv1.ClusterAddress{
			URL:  "https://cluster.seed.kkp",
			Port: 6443,
		},
		Versions: kubermaticv1.ClusterVersionsStatus{
			ControlPlane: semver.Semver("v1.32.1"),
		},
		NamespaceName: "cluster-default",
	}

	if err := client.Status().Update(ctx, cluster); err != nil {
		t.Fatalf("failed to update test cluster status: %v", err)
	}

	t.Cleanup(func() {
		if err := client.Delete(ctx, cluster); err != nil {
			t.Fatalf("failed to cleanup test cluster: %v", err)
		}
	})
}

// StartTestEnvWithCleanup bootstraps the testing environment and return the to the kubeconfig. It also registers a hook on T.Cleanup to
// stop the env and remove the kubeconfig.
func StartTestEnvWithCleanup(t *testing.T, crdPath string) (context.Context, ctrlruntimeclient.Client, string) {
	ctx, cancel := context.WithCancel(context.Background())
	log.Logger = log.New(true, log.FormatJSON).Sugar()

	// Bootstrapping test environment.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{crdPath},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start envTest: %s", err)
	}

	if err := kubermaticv1.AddToScheme(scheme.Scheme); err != nil {
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
func CheckConfigMap(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, expectedData map[string]string, expectedVersionLabel string, enableDNS bool) {
	t.Helper()

	cm := &corev1.ConfigMap{}
	var errorGetCm error
	if !utils.WaitFor(ctx, time.Second*1, time.Second*10, func() bool {
		errorGetCm = client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: ConfigmapName}, cm)
		// if config map has not been  updated
		if errorGetCm != nil || !diff.SemanticallyEqual(expectedData, cm.Data) || cm.Labels[VersionLabelKey] != expectedVersionLabel || !isExpectedDNSValue(cm.Labels[EnableDNSLabelKey], enableDNS) {
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
		if !isExpectedDNSValue(cm.Labels[EnableDNSLabelKey], enableDNS) {
			if enableDNS {
				t.Errorf("ConfigMap label '%s' should not be empty as enableDns is enabled", EnableDNSLabelKey)
			}
			t.Errorf("ConfigMap label '%s' should be empty as enableDns is disabled", EnableDNSLabelKey)
		}
	}
}

func isExpectedDNSValue(value string, enableDNS bool) bool {
	if enableDNS {
		// is enableDns the value should be equal to the ip of the hostname but to make test more resilient we just check is not empty.
		return len(value) > 0
	}
	return len(value) == 0
}

// AssertContainsExactly failed the test if the actual slice does not contain exactly the element of the expected slice.
func AssertContainsExactly[T comparable](t *testing.T, prefixMsg string, actual []T, expected []T) {
	t.Helper()

	missing := map[T]struct{}{}
	notExpected := map[T]struct{}{}
	for _, val := range expected {
		missing[val] = struct{}{}
	}

	for _, val := range actual {
		notExpected[val] = struct{}{}
	}

	for _, val := range actual {
		if _, found := missing[val]; found {
			delete(missing, val)
			delete(notExpected, val)
		}
	}
	if len(missing) != 0 || len(notExpected) != 0 {
		t.Fatalf("%s. expect %+v, to contains only %+v.\nMissing elements %+v\nUnexpected elements %+v", prefixMsg, actual, expected, keys(missing), keys(notExpected))
	}
}

// keys return the keys of the map.
func keys[T comparable](dict map[T]struct{}) []T {
	var res []T
	for k := range dict {
		res = append(res, k)
	}
	return res
}

// ReleaseStorageInfo holds information about the secret storing the Helm release information.
type ReleaseStorageInfo struct {
	// Name of the secret containing release information.
	Name string

	// Version of the release.
	Version string
}

// MapToReleaseStorageInfo maps the secrets containing the Helm release information to a smaller struct only containing the Name of the secret and the release version.
func MapToReleaseStorageInfo(secrets []corev1.Secret) []ReleaseStorageInfo {
	res := []ReleaseStorageInfo{}
	for _, secret := range secrets {
		res = append(res, ReleaseStorageInfo{
			Name:    secret.Name,
			Version: secret.Labels["version"],
		})
	}
	return res
}
