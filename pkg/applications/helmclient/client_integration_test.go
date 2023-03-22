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
	"path/filepath"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"

	"k8c.io/kubermatic/v2/pkg/applications/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
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
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
			},
		},
		{
			name: "installation from dir with default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
			},
		},
		{
			name: "installation from archive with custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false)
			},
		},
		{
			name: "installation from dir with custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false)
			},
		},
		{
			name: "installation from archive should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartArchiveV1Path, map[string]interface{}{})
			},
		},
		{
			name: "installation from dir should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartDirV1Path, map[string]interface{}{})
			},
		},
		{
			name: "upgrade from archive with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false, 2)
			},
		},
		{
			name: "upgrade from dir with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false, 2)
			},
		},
		{
			name: "upgrade from archive with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2, false, 2)
			},
		},
		{
			name: "upgrade from dir with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2, false, 2)
			},
		},
		{
			name: "upgrade from archive with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version", false, 2)
			},
		},
		{
			name: "upgrade from dir with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version", false, 2)
			},
		},
		// tests for https://github.com/kubermatic/kubermatic/issues/11864
		{
			name: "upgrade from archive with no values should upgrade chart with default values",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false, 2)
			},
		},
		{
			name: "upgrade from dir with no values should upgrade chart with default values",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false)
				upgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false, 2)
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
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2, 2)
			},
		},
		{
			name: "installOrUpgrade from archive should upgrades chart when it already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVerionLabelV2, 2)
			},
		},
		{
			name: "uninstall should be successful when chart is already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				uninstallTest(t, ctx, client, ns)
			},
		},
		{
			name: "uninstall should be successful when chart has already been installed (idempotent)",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				uninstallTest(t, ctx, client, ns)
			},
		},
		{
			name: "install should be successful when enabledDns is enabled",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, true)
			},
		},
		{
			name: "upgrade should be successful when enabledDns is switched to true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, true, 2)
			},
		},
		{
			name: "upgrade should be successful when enabledDns is switched to false",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, true)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false, 2)
			},
		},

		{
			name: "install should fail when timeout is exceeded",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)

				deployOpts, err := NewDeployOpts(true, 5*time.Second, false, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				releaseInfo, err := helmClient.Install(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check that even if install has failed (timeout), workload has been deployed.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultDataV2, test.DefaultVerionLabelV2, false)
				checkServiceDeployed(t, ctx, client, ns)
			},
		},
		{
			name: "install should fail and workloard should be reverted when timeout is exceeded and atomic=true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)

				deployOpts, err := NewDeployOpts(true, 5*time.Second, true, false)
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
				deployOpts, err := NewDeployOpts(true, 5*time.Second, false, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				// test install chart and run upgrade.
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check that even if upgrade has failed (timeout), workload has been updated.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultDataV2, test.DefaultVerionLabelV2, false)
				checkServiceDeployed(t, ctx, client, ns)
			},
		},
		{
			name: "upgrade should fail and workload should be reverted when timeout is exceeded and atomic is true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartDirV2Path)
				deployOpts, err := NewDeployOpts(true, 5*time.Second, true, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				// test install chart and run upgrade.
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVerionLabel, false)
				releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimemout(t, releaseInfo, err)

				// check atomic flag has revert release to previous version.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultData, test.DefaultVerionLabel, false)
				checkServiceDeleted(t, ctx, client, ns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func installTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, enableDNS bool) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	deployOpts, err := NewDeployOpts(false, 0, false, enableDNS)
	if err != nil {
		t.Fatalf("failed to build DeployOpts: %s", err)
	}

	releaseInfo, err := helmClient.Install(chartFullPath, releaseName, values, *deployOpts, AuthSettings{})

	if err != nil {
		t.Fatalf("helm install failed :%s", err)
	}

	if releaseInfo.Version != 1 {
		t.Fatalf("invalid helm release version. expected 1, got %v", releaseInfo.Version)
	}

	test.CheckConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel, enableDNS)
}

func installShouldFailedIfAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string, values map[string]interface{}) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)

	_, err := helmClient.Install(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err == nil {
		t.Fatalf("helm install when release is already installed should failed, but no error was raised")
	}
}

func upgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, enableDNS bool, expectedReleaseVersion int) {
	helmClient, chartFullPath := buildHelClient(t, ctx, ns, chartPath)
	deployOpts, err := NewDeployOpts(false, 0, false, enableDNS)
	if err != nil {
		t.Fatalf("failed to build DeployOpts: %s", err)
	}
	releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, values, *deployOpts, AuthSettings{})

	if err != nil {
		t.Fatalf("helm upgrade failed :%s", err)
	}

	if releaseInfo.Version != expectedReleaseVersion {
		t.Fatalf("invalid helm release version. expected %v, got %v", expectedReleaseVersion, releaseInfo.Version)
	}

	test.CheckConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel, enableDNS)
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

	test.CheckConfigMap(t, ctx, client, ns, expectedData, expectedVersionLabel, false)
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
	deployOpts, err := NewDeployOpts(false, 0, false, false)
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

func TestDownloadChart(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	chartArchiveDir := t.TempDir()

	chartGlobPath := path.Join(chartArchiveDir, "*.tgz")
	chartArchiveV1Path, chartArchiveV1Size := test.PackageChart(t, "testdata/examplechart", chartArchiveDir)
	chartArchiveV2Path, chartArchiveV2Size := test.PackageChart(t, "testdata/examplechart-v2", chartArchiveDir)
	chartArchiveV1Name := path.Base(chartArchiveV1Path)
	chartArchiveV2Name := path.Base(chartArchiveV2Path)

	httpRegistryUrl := test.StartHttpRegistryWithCleanup(t, chartGlobPath)
	httpRegistryWithAuthUrl := test.StartHttpRegistryWithAuthAndCleanup(t, chartGlobPath)

	ociRegistryUrl := test.StartOciRegistry(t, chartGlobPath)
	ociregistryWithAuthUrl, registryConfigFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	testCases := []struct {
		name               string
		repoUrl            string
		chartName          string
		chartVersion       string
		auth               AuthSettings
		expectedAchiveName string
		expectedChartSize  int64
		wantErr            bool
	}{
		{
			name:               "Download from HTTP repository should be successful",
			repoUrl:            httpRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:               "Download from HTTP repository with empty version should get the latest version",
			repoUrl:            httpRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV2Name,
			expectedChartSize:  chartArchiveV2Size,
			wantErr:            false,
		},
		{
			name:               "Download from HTTP repository with auth should be successful",
			repoUrl:            httpRegistryWithAuthUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{Username: "username", Password: "password"},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:              "Download from HTTP repository should fail when chart does not exist",
			repoUrl:           httpRegistryUrl,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
		{
			name:              "Download from HTTP repository should fail when version is not a semversion",
			repoUrl:           httpRegistryUrl,
			chartName:         "examplechart",
			chartVersion:      "notSemver",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},

		{
			name:               "Download from OCI repository should be successful",
			repoUrl:            ociRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:               "Download from oci repository with empty version should get the latest version",
			repoUrl:            ociRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV2Name,
			expectedChartSize:  chartArchiveV2Size,
			wantErr:            false,
		},
		{
			name:               "Download from OCI repository with auth should be successful",
			repoUrl:            ociregistryWithAuthUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{RegistryConfigFile: registryConfigFile},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:              "Download from oci repository should fail when chart does not exist",
			repoUrl:           ociRegistryUrl,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
		{
			name:              "Download from oci repository should fail when version is not a semversion",
			repoUrl:           ociRegistryUrl,
			chartName:         "examplechart",
			chartVersion:      "notSemver",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				downloadDest := t.TempDir()
				settings := NewSettings(downloadDest)

				tf := cmdtesting.NewTestFactory().WithNamespace(defaultNs)
				defer tf.Cleanup()

				helmClient, err := NewClient(context.Background(), tf, settings, defaultNs, log)
				if err != nil {
					t.Fatalf("can not init helm Client: %s", err)
				}

				chartLoc, err := helmClient.DownloadChart(tc.repoUrl, tc.chartName, tc.chartVersion, downloadDest, tc.auth)

				if (err != nil) != tc.wantErr {
					t.Fatalf("DownloadChart() error = %v, wantErr %v", err, tc.wantErr)
				}

				// No need to proceed with further tests if an error is expected.
				if tc.wantErr {
					return
				}

				// Test chart is downloaded where we expect
				expectedChartLoc := downloadDest + "/" + tc.expectedAchiveName
				if chartLoc != expectedChartLoc {
					t.Fatalf("charLoc is invalid. got '%s'. expect '%s'", chartLoc, expectedChartLoc)
				}

				// Smoke test: check downloaded chart has expected size
				downloadChartInfo, err := os.Stat(chartLoc)
				if err != nil {
					t.Fatalf("can not check size of downloaded chart: %s", err)
				}

				if tc.expectedChartSize != downloadChartInfo.Size() {
					t.Errorf("size of download chart should be '%d' but was '%d'", tc.expectedChartSize, downloadChartInfo.Size())
				}
			}()
		})
	}
}

func TestBuildDependencies(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	chartArchiveDir := t.TempDir()
	chartGlobPath := path.Join(chartArchiveDir, "*.tgz")

	test.PackageChart(t, "testdata/examplechart", chartArchiveDir)
	test.PackageChart(t, "testdata/examplechart2", chartArchiveDir)

	httpRegistryUrl := test.StartHttpRegistryWithCleanup(t, chartGlobPath)
	httpRegistryWithAuthUrl := test.StartHttpRegistryWithAuthAndCleanup(t, chartGlobPath)

	ociRegistryUrl := test.StartOciRegistry(t, chartGlobPath)
	ociRegistryWithAuthUrl, registryConfigFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	const fileDepChartName = "filedepchart"
	const fileDepChartVersion = "2.3.4"
	testCases := []struct {
		name         string
		dependencies []*chart.Dependency
		hasLockFile  bool
		auth         AuthSettings
		wantErr      bool
	}{
		{
			name:         "no dependencies",
			dependencies: []*chart.Dependency{},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "file dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: fileDepChartName, Version: fileDepChartVersion, Repository: "file://../" + fileDepChartName}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "file dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: fileDepChartName, Version: fileDepChartVersion, Repository: "file://../" + fileDepChartName}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http and oci dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}, {Name: "examplechart2", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http and oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}, {Name: "examplechart2", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies with Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryWithAuthUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{Username: "username", Password: "password"},
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{Username: "username", Password: "password"},
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{RegistryConfigFile: registryConfigFile},
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{RegistryConfigFile: registryConfigFile},
			wantErr:      false,
		},
		{
			name:         "http dependency with empty version should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: httpRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "oci dependency with empty version should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "http dependency with non semver should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: httpRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "oci dependency with non semver should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				tempDir := t.TempDir()
				settings := NewSettings(tempDir)

				tf := cmdtesting.NewTestFactory().WithNamespace(defaultNs)
				defer tf.Cleanup()

				// This chart may be used as a file dependency by testingChart.
				fileDepChart := &chart.Chart{
					Metadata: &chart.Metadata{
						APIVersion: chart.APIVersionV2,
						Name:       fileDepChartName,
						Version:    fileDepChartVersion,
					},
				}
				if err := chartutil.SaveDir(fileDepChart, tempDir); err != nil {
					t.Fatal(err)
				}

				chartName := "testing-chart"
				testingChart := &chart.Chart{
					Metadata: &chart.Metadata{
						APIVersion:   chart.APIVersionV2,
						Name:         chartName,
						Version:      "1.2.3",
						Dependencies: tc.dependencies,
					},
				}
				if err := chartutil.SaveDir(testingChart, tempDir); err != nil {
					t.Fatal(err)
				}

				lockFile := filepath.Join(tempDir, chartName, "Chart.lock")
				generatedTime := time.Now()
				// chartutil.SaveDir does not save Chart.lock so we do it manually
				if tc.hasLockFile {
					digest, err := HashReq(tc.dependencies, tc.dependencies)
					if err != nil {
						t.Fatal(err)
					}
					loc := &chart.Lock{
						Generated:    generatedTime,
						Digest:       digest,
						Dependencies: tc.dependencies,
					}
					out, err := yaml.Marshal(loc)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(lockFile, out, 0644); err != nil {
						t.Fatal(err)
					}
				}

				helmClient, err := NewClient(context.Background(), tf, settings, defaultNs, log)
				if err != nil {
					t.Fatalf("can not init helm client: %s", err)
				}

				chartUnderTest, err := helmClient.buildDependencies(filepath.Join(tempDir, chartName), tc.auth)

				if (err != nil) != tc.wantErr {
					t.Fatalf("buildDependencies() error = %v, wantErr %v", err, tc.wantErr)
				}

				// No need to proceed with further tests if an error is expected.
				if tc.wantErr {
					return
				}

				// check dependencies
				for _, dep := range tc.dependencies {
					depArchiveName := dep.Name + "-" + dep.Version + ".tgz"
					desiredDependency := filepath.Join(tempDir, chartName, "charts", depArchiveName)
					if _, err := os.Stat(desiredDependency); err != nil {
						t.Fatalf("dependency %v has not been downloaded in charts directory: %s", dep, err)
					}
					assertDependencyLoaded(chartUnderTest, dep, t)
				}
				if tc.hasLockFile {
					actualLock := &chart.Lock{}
					in, err := os.ReadFile(lockFile)
					if err != nil {
						t.Fatalf("can not read actual Chart.lock: %s", err)
					}
					if err := yaml.Unmarshal(in, actualLock); err != nil {
						t.Fatalf("can not unmarshamm Chart.lock: %s", err)
					}
					if !generatedTime.Equal(actualLock.Generated) {
						t.Fatalf("lock file should not have been modified. expected generatedTime:%v, actual generatedTime:%v", generatedTime, actualLock.Generated)
					}
				}
			}()
		})
	}
}
