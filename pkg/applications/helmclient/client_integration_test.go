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

package helmclient

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	timeout     = time.Second * 10
	interval    = time.Second * 1
	releaseName = "local"
)

var (
	kubeconfigPath     string
	chartArchiveV1Path string
	chartArchiveV2Path string
)

func TestHelmClient(t *testing.T) {
	ctx, client := startTestEnvAndPackageChartsWithCleanup(t)

	const chartDirV1Path = "testdata/examplechart"
	const chartDirV2Path = "testdata/examplechart-v2"
	defaultData := map[string]string{"foo": "bar"}
	defaultDataV2 := map[string]string{"foo-version-2": "bar-version-2"}
	defaultVerionLabel := "1.0"
	defaultVerionLabelV2 := "2.0"

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "installation from archive with default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
			},
		},
		{
			name: "installation from dir with default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
			},
		},
		{
			name: "installation from archive with custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{"cmData": map[string]interface{}{"hello": "world"}, "versionLabel": "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "installation from dir with custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{"cmData": map[string]interface{}{"hello": "world"}, "versionLabel": "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "installation from archive should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartArchiveV1Path, map[string]interface{}{})
			},
		},
		{
			name: "installation from dir should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartDirV1Path, map[string]interface{}{})
			},
		},
		{
			name: "upgrade from archive with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{"cmData": map[string]interface{}{"hello": "world"}, "versionLabel": "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "upgrade from dir with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{"cmData": map[string]interface{}{"hello": "world"}, "versionLabel": "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "upgrade from archive with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, defaultDataV2, defaultVerionLabelV2)
			},
		},
		{
			name: "upgrade from dir with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, defaultDataV2, defaultVerionLabelV2)
			},
		},
		{
			name: "upgrade from archive with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{"cmData": map[string]interface{}{"hello": "world"}, "versionLabel": "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version")
			},
		},
		{
			name: "upgrade from dir with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{"cmData": map[string]interface{}{"hello": "world"}, "versionLabel": "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version")
			},
		},
		{
			name: "upgrade from archive should failed if not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				upgradeShouldFailedIfNotAlreadyInstalledTest(t, ctx, ns, chartArchiveV1Path, map[string]interface{}{})
			},
		},
		{
			name: "upgrade from dir should failed if not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				upgradeShouldFailedIfNotAlreadyInstalledTest(t, ctx, ns, chartDirV1Path, map[string]interface{}{})
			},
		},
		{
			name: "installOrUpgrade from archive should installs chart when it not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel, 1)
			},
		},
		{
			name: "installOrUpgrade from Dir should installs chart when it not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel, 1)
			},
		},
		{
			name: "installOrUpgrade from archive should upgrades chart when it already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, defaultDataV2, defaultVerionLabelV2, 2)
			},
		},
		{
			name: "installOrUpgrade from archive should upgrades chart when it already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, defaultDataV2, defaultVerionLabelV2, 2)
			},
		},
		{
			name: "uninstall should be successful when chart is already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, defaultData, defaultVerionLabel)
				uninstallTest(t, ctx, client, ns)
			},
		},
		{
			name: "uninstall should be failed when chart is not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				uninstallShloulFailedIfnotAlreadyInstalledTest(t, ctx, ns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func installTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	releaseInfo, err := helmClient.Install(chartFullPath, releaseName, values, AuthSettings{})

	if err != nil {
		t.Fatalf("helm install failed :%s", err)
	}

	if releaseInfo.Version != 1 {
		t.Fatalf("invalid helm release version. expected 1, got %v", releaseInfo.Version)
	}

	checkConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel)
}

func installShouldFailedIfAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string, values map[string]interface{}) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	_, err := helmClient.Install(chartFullPath, releaseName, values, AuthSettings{})

	if err == nil {
		t.Fatalf("helm install when release is already installed should failed, but no error was raised")
	}
}

func upgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, values, AuthSettings{})

	if err != nil {
		t.Fatalf("helm upgrade failed :%s", err)
	}

	if releaseInfo.Version != 2 {
		t.Fatalf("invalid helm release version. expected 2, got %v", releaseInfo.Version)
	}

	checkConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel)
}

func upgradeShouldFailedIfNotAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string, values map[string]interface{}) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	_, err := helmClient.Upgrade(chartFullPath, releaseName, values, AuthSettings{})

	if err == nil {
		t.Fatalf("helm upgrade when release is not already installed should failed, but no error was raised")
	}
}

func installOrUpgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, expectedRelVersion int) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	releaseInfo, err := helmClient.InstallOrUpgrade(chartFullPath, releaseName, values, AuthSettings{})

	if err != nil {
		t.Fatalf("helm InstallOrUpgrade failed :%s", err)
	}

	if releaseInfo.Version != expectedRelVersion {
		t.Fatalf("invalid helm release version. expected %v, got %v", expectedRelVersion, releaseInfo.Version)
	}

	checkConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel)
}

func uninstallTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace) {
	tempDir := t.TempDir()
	settings := NewSettings(tempDir)

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: pointer.String(kubeconfigPath),
		Namespace:  &ns.Name,
	}

	helmClient, err := NewClient(ctx, restClientGetter, settings, ns.Name, kubermaticlog.Logger)
	if err != nil {
		t.Fatalf("failed to create helm client: %s", err)
	}

	_, err = helmClient.Uninstall(releaseName)

	if err != nil {
		t.Fatalf("helm uninstall failed :%s", err)
	}

	cm := &corev1.ConfigMap{}
	if !utils.WaitFor(interval, timeout, func() bool {
		err := client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: "testcm"}, cm)
		return err != nil && apierrors.IsNotFound(err)
	}) {
		t.Fatal("configMap has not been removed by helm unsintall")
	}
}

func uninstallShloulFailedIfnotAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace) {
	tempDir := t.TempDir()
	settings := NewSettings(tempDir)

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: pointer.String(kubeconfigPath),
		Namespace:  &ns.Name,
	}

	helmClient, err := NewClient(ctx, restClientGetter, settings, ns.Name, kubermaticlog.Logger)
	if err != nil {
		t.Fatalf("failed to create helm client: %s", err)
	}

	_, err = helmClient.Uninstall(releaseName)

	if err == nil {
		t.Fatal("helm uninstall should failed if release it not already installed, but not error was raised")
	}
}

// checkConfigMap asserts that configMap deployed by helm chart has been created/ updated with the desired data and labels.
func checkConfigMap(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, expectedData map[string]string, expectedVersionLabel string) {
	t.Helper()

	cm := &corev1.ConfigMap{}
	var errorGetCm error
	if !utils.WaitFor(interval, timeout, func() bool {
		errorGetCm = client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: "testcm"}, cm)
		// if config map has not been  updated
		if errorGetCm != nil || !diff.SemanticallyEqual(expectedData, cm.Data) || cm.Labels["versionLabel"] != expectedVersionLabel {
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
		if cm.Labels["versionLabel"] != expectedVersionLabel {
			t.Errorf("ConfigMap versionLabel has invalid value. expected '%s', got '%s'", expectedVersionLabel, cm.Labels["versionLabel"])
		}
	}
}

// buildHelClient creates Helm client and returns it with the full path of the chart.
//
// if chartPath is a directory then we copy the chart directory into Helm TempDir to avoid any concurrency problem
// when building dependencies and return the full path to the copied chart. (i.e. Helm_TMP_DIR/chartDir)
//
// if chartPath is an archive (i.e. chart.tgz) then we simply returns the path to the chart archive.
func buildHelClient(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string) (*HelmClient, string) {
	t.Helper()
	tempDir := t.TempDir()
	settings := NewSettings(tempDir)

	fi, err := os.Stat(chartPath)
	if err != nil {
		t.Fatalf("can not find chart at `%s': %s", chartPath, err)
	}
	var chartFullPath = chartPath
	if fi.IsDir() {
		chartFullPath = path.Join(tempDir, path.Base(chartPath))
		if err := copyDir(chartPath, chartFullPath); err != nil {
			t.Fatalf("failed to copy chart directory to temp dir: %s", err)
		}
	}

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: pointer.String(kubeconfigPath),
		Namespace:  &ns.Name,
	}

	helmClient, err := NewClient(ctx, restClientGetter, settings, ns.Name, kubermaticlog.Logger)
	if err != nil {
		t.Fatalf("failed to create helm client: %s", err)
	}
	return helmClient, chartFullPath
}

func copyDir(source string, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		// error may occurred if path is not accessible
		if err != nil {
			return err
		}
		relPath := strings.Replace(path, source, "", 1)
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			data, err := os.ReadFile(filepath.Join(source, relPath))
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(destination, relPath), data, 0755)
		}
	})
}

// startTestEnvAndPackageChartsWithCleanup bootstraps the testing environment and registers a hook on T.Cleanup to
// clean it.
//
// more precisely it:
//   - starts envTest
//   - packages charts testdata/examplechart and testdata/examplechart-v2. archive path are stored in chartArchiveV1Path and chartArchiveV2Path variables
//   - writes the kubeconfig to access envTest cluster in temporary directory. The path is stored in kubeconfigPath variable.
func startTestEnvAndPackageChartsWithCleanup(t *testing.T) (context.Context, ctrlruntimeclient.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	// Bootstrapping test environment.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start envTest: %s", err)
	}

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

	// packages Helm charts
	helmTempDir := t.TempDir()
	chartArchiveV1Path, _ = test.PackageChart(t, "testdata/examplechart", helmTempDir)
	chartArchiveV2Path, _ = test.PackageChart(t, "testdata/examplechart-v2", helmTempDir)

	// get envTest kubeconfig
	kubeconfig := *clientcmdapi.NewConfig()
	kubeconfig.Clusters["testenv"] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}
	kubeconfig.CurrentContext = "default-context"
	kubeconfig.Contexts["default-context"] = &clientcmdapi.Context{
		Cluster:   "testenv",
		Namespace: "default",
		AuthInfo:  "auth-info",
	}
	kubeconfig.AuthInfos["auth-info"] = &clientcmdapi.AuthInfo{ClientCertificateData: cfg.CertData, ClientKeyData: cfg.KeyData}

	kubeconfigPath = path.Join(helmTempDir, "testEnv-kubeconfig")
	err = clientcmd.WriteToFile(kubeconfig, kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to write testEnv kubeconfig: %s", err)
	}

	return ctx, client
}
