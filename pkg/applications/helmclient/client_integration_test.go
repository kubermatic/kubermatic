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
	"errors"
	"os"
	"path"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/release"

	"k8c.io/kubermatic/v2/pkg/applications/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout     = time.Second * 10
	interval    = time.Second * 1
	releaseName = "local"
)

var (
	kubeconfigPath string
)

func TestHelmClient(t *testing.T) {
	var ctx context.Context
	var client ctrlruntimeclient.Client
	ctx, client, kubeconfigPath = test.StartTestEnvWithCleanup(t, "../../crd/k8c.io")

	const chartDirV1Path = "testdata/examplechart"
	const chartDirV2Path = "testdata/examplechart-v2"

	// packages Helm charts
	helmTempDir := t.TempDir()
	chartArchiveV1Path, _ := test.PackageChart(t, chartDirV1Path, helmTempDir)
	chartArchiveV2Path, _ := test.PackageChart(t, chartDirV2Path, helmTempDir)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "installation from archive with default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
			},
		},
		{
			name: "installation from dir with default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
			},
		},
		{
			name: "installation from archive with custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "installation from dir with custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "installation from archive should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartArchiveV1Path, map[string]interface{}{})
			},
		},
		{
			name: "installation from dir should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartDirV1Path, map[string]interface{}{})
			},
		},
		{
			name: "upgrade from archive with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "upgrade from dir with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version")
			},
		},
		{
			name: "upgrade from archive with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2)
			},
		},
		{
			name: "upgrade from dir with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2)
			},
		},
		{
			name: "upgrade from archive with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version")
			},
		},
		{
			name: "upgrade from dir with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version")
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
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, 1)
			},
		},
		{
			name: "installOrUpgrade from Dir should installs chart when it not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, 1)
			},
		},
		{
			name: "installOrUpgrade from archive should upgrades chart when it already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2, 2)
			},
		},
		{
			name: "installOrUpgrade from archive should upgrades chart when it already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2, 2)
			},
		},
		{
			name: "uninstall should be successful when chart is already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
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
		{
			name: "install should fail when timeout is exceeded",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)

				deployOpts, err := NewDeployOpts(true, 5*time.Second, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				releaseInfo, err := helmClient.Install(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check that even if install has failed (timeout), workload has been deployed.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultDataV2, test.DefaultVerionLabelV2)
				checkServiceDeployed(t, ctx, client, ns)
			},
		},

		{
			name: "install should fail and workloard should be reverted when timeout is exceeded and atomic=true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)

				deployOpts, err := NewDeployOpts(true, 5*time.Second, true)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				releaseInfo, err := helmClient.Install(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check atomic has removed resources
				checkServiceDeleted(t, ctx, client, ns)
			},
		},
		{
			name: "upgrade should fail when timeout is exceeded",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)
				deployOpts, err := NewDeployOpts(true, 5*time.Second, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				// test install chart and run upgrade.
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check that even if upgrade has failed (timeout), workload has been updated.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultDataV2, test.DefaultVerionLabelV2)
				checkServiceDeployed(t, ctx, client, ns)
			},
		},
		{
			name: "upgrade should fail and workload should be reverted when timeout is exceeded and atomic is true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)
				deployOpts, err := NewDeployOpts(true, 5*time.Second, true)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				// test install chart and run upgrade.
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel)
				releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check atomic flag has revert release to previous version.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultData, test.DefaultVerionLabel)
				checkServiceDeleted(t, ctx, client, ns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func installTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	releaseInfo, err := helmClient.Install(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err != nil {
		t.Fatalf("helm install failed :%s", err)
	}

	if releaseInfo.Version != 1 {
		t.Fatalf("invalid helm release version. expected 1, got %v", releaseInfo.Version)
	}

	test.CheckConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel)
}

func installShouldFailedIfAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string, values map[string]interface{}) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	_, err := helmClient.Install(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err == nil {
		t.Fatalf("helm install when release is already installed should failed, but no error was raised")
	}
}

func upgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err != nil {
		t.Fatalf("helm upgrade failed :%s", err)
	}

	if releaseInfo.Version != 2 {
		t.Fatalf("invalid helm release version. expected 2, got %v", releaseInfo.Version)
	}

	test.CheckConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel)
}

func upgradeShouldFailedIfNotAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string, values map[string]interface{}) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	_, err := helmClient.Upgrade(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err == nil {
		t.Fatalf("helm upgrade when release is not already installed should failed, but no error was raised")
	}
}

func installOrUpgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, expectedRelVersion int) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	releaseInfo, err := helmClient.InstallOrUpgrade(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err != nil {
		t.Fatalf("helm InstallOrUpgrade failed :%s", err)
	}

	if releaseInfo.Version != expectedRelVersion {
		t.Fatalf("invalid helm release version. expected %v, got %v", expectedRelVersion, releaseInfo.Version)
	}

	test.CheckConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel)
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
		err := client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: test.ConfigmapName}, cm)
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
		if err := test.CopyDir(chartPath, chartFullPath); err != nil {
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

// defaultDeployOpts creates DeployOpts with wait=false and atomic=false.
func defaultDeployOpts(t *testing.T) *DeployOpts {
	deployOpts, err := NewDeployOpts(false, 0, false)
	if err != nil {
		t.Fatalf("failed to build default deployOpts: %s", err)
	}
	return deployOpts
}

// checkReleaseFailedWithTimemout checks that install / upgrade has failed with timeout error and releaseInfo has failure status.
func checkReleaseFailedWithTimemout(t *testing.T, releaseInfo *release.Release, err error) {
	if err == nil {
		t.Fatalf("expect installation or upgrade failed when timeout is exceeded but not error was raised")
	}
	if !errors.Is(err, wait.ErrWaitTimeout) {
		t.Fatalf("expect wait.ErrWaitTimeout error. got %s", err)
	}
	if releaseInfo.Info.Status != release.StatusFailed {
		t.Fatalf("expect releaseInfo.Info.Status to be '%s', got '%s'", release.StatusFailed, releaseInfo.Info.Status)
	}
}

// checkServiceDeleted checks that test.SvcName does not exist in namespace "ns" otherwise fails the test.
func checkServiceDeleted(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace) {
	svc := &corev1.Service{}
	if !utils.WaitFor(interval, timeout, func() bool {
		err := client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: test.SvcName}, svc)
		return err != nil && apierrors.IsNotFound(err)
	}) {
		t.Fatal("service has not been removed by helm unsintall or atomic flag.")
	}
}

// checkServiceDeployed checks that test.SvcName exist in namespace "ns" otherwise fails the test.
func checkServiceDeployed(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace) {
	var err error
	if !utils.WaitFor(interval, timeout, func() bool {
		svc := &corev1.Service{}
		err = client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: test.SvcName}, svc)
		return err == nil
	}) {
		t.Fatalf("service has not been deployed: %s", err)
	}
}
