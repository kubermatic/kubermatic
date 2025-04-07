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
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/utils/ptr"
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
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
			},
		},
		{
			name: "installation from dir with default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
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
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartArchiveV1Path, map[string]interface{}{})
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"}})
			},
		},
		{
			name: "installation from dir should failed if already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				installShouldFailedIfAlreadyInstalledTest(t, ctx, ns, chartDirV1Path, map[string]interface{}{})
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"}})
			},
		},
		{
			name: "upgrade from archive with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade from dir with same version and different values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade from archive with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade from dir with new version and default values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade from archive with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version", false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade from dir with new version and custom values should be successful",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, "another-version", false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		// tests for https://github.com/kubermatic/kubermatic/issues/11864
		{
			name: "upgrade from archive with no values should upgrade chart with default values",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade from dir with no values should upgrade chart with default values",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}, test.VersionLabelKey: "another-version"}, map[string]string{"hello": "world", "foo": "bar"}, "another-version", false)
				upgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
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
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, 1)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
				})
			},
		},
		{
			name: "installOrUpgrade from Dir should installs chart when it not already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, 1)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
				})
			},
		},
		{
			name: "installOrUpgrade from archive should upgrade v1 chart when it was already installed and its version changed to v2",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "installOrUpgrade from archive should upgrade v1 chart when it was already installed and its values changed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}}, map[string]string{"hello": "world", "foo": "bar"}, test.DefaultVersionLabel, 2)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "installOrUpgrade from archive should not upgrade v1 chart when it was already installed and its values weren't changed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				installOrUpgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, 1)
				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
				})
			},
		},
		{
			name: "uninstall should be successful when chart is already installed",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
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
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, true)
			},
		},
		{
			name: "upgrade should be successful when enabledDns is switched to true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, true, 2)

				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade should be successful when enabledDns is switched to false",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, true)
				upgradeTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false, 2)

				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},

		{
			name: "install should fail when timeout is exceeded",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartDirV2Path)

				deployOpts, err := NewDeployOpts(true, 5*time.Second, false, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				releaseInfo, err := helmClient.Install(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimeout(t, releaseInfo, err)

				// check that even if install has failed (timeout), workload has been deployed.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultDataV2, test.DefaultVersionLabelV2, false)
				checkServiceDeployed(t, ctx, client, ns)

				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
				})
			},
		},
		{
			name: "install should fail and workloard should be reverted when timeout is exceeded and atomic=true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartDirV2Path)

				deployOpts, err := NewDeployOpts(true, 5*time.Second, true, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				releaseInfo, err := helmClient.Install(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimeout(t, releaseInfo, err)

				// check atomic has removed resources
				checkServiceDeleted(t, ctx, client, ns)

				// no release should be stored as install failed and atomic=true
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{})
			},
		},
		{
			name: "upgrade should fail when timeout is exceeded",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartDirV2Path)
				deployOpts, err := NewDeployOpts(true, 5*time.Second, false, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				// test install chart and run upgrade.
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimeout(t, releaseInfo, err)

				// check that even if upgrade has failed (timeout), workload has been updated.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultDataV2, test.DefaultVersionLabelV2, false)
				checkServiceDeployed(t, ctx, client, ns)

				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
				})
			},
		},
		{
			name: "upgrade should fail and workload should be reverted when timeout is exceeded and atomic is true",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartDirV2Path)
				deployOpts, err := NewDeployOpts(true, 5*time.Second, true, false)
				if err != nil {
					t.Fatalf("failed to build DeployOpts: %s", err)
				}

				// test install chart and run upgrade.
				installTest(t, ctx, client, ns, chartArchiveV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				releaseInfo, err := helmClient.Upgrade(chartFullPath, releaseName, map[string]interface{}{test.DeploySvcKey: true}, *deployOpts, AuthSettings{})

				// check helm operation has failed.
				checkReleaseFailedWithTimeout(t, releaseInfo, err)

				// check atomic flag has revert release to previous version.
				test.CheckConfigMap(t, ctx, client, ns, test.DefaultData, test.DefaultVersionLabel, false)
				checkServiceDeleted(t, ctx, client, ns)

				// check only desired release are stored on cluster
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{
					{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"},
					{Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"},
					{Name: "sh.helm.release.v1." + releaseName + ".v3", Version: "3"}, // this one is due to the rollback (atomic= true)
				})
			},
		},
		// tests for https://github.com/kubermatic/kubermatic/issues/12078
		{
			name: "release history should be prune after an upgrade",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, false)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, false, 2)
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"}, {Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"}})

				// history limit is 2 so these next upgrades should trigger a clean up of old helm release
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, false, 3)
				upgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, false, 4)

				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v3", Version: "3"}, {Name: "sh.helm.release.v1." + releaseName + ".v4", Version: "4"}})
			},
		},

		{
			name: "release history should be prune after an InstallOrupgrade",
			testFunc: func(t *testing.T) {
				ns := test.CreateNamespaceWithCleanup(t, ctx, client)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV1Path, map[string]interface{}{}, test.DefaultData, test.DefaultVersionLabel, 1)
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"}})

				// upgrade
				installOrUpgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{}, test.DefaultDataV2, test.DefaultVersionLabelV2, 2)
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"}, {Name: "sh.helm.release.v1." + releaseName + ".v2", Version: "2"}})

				// history limit is 2 so these next upgrades should trigger a clean up of old helm release
				installOrUpgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"hello": "world"}}, map[string]string{"hello": "world", "foo-version-2": "bar-version-2"}, test.DefaultVersionLabelV2, 3)
				installOrUpgradeTest(t, ctx, client, ns, chartDirV2Path, map[string]interface{}{test.CmDataKey: map[string]interface{}{"next": "update"}}, map[string]string{"next": "update", "foo-version-2": "bar-version-2"}, test.DefaultVersionLabelV2, 4)
				checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v3", Version: "3"}, {Name: "sh.helm.release.v1." + releaseName + ".v4", Version: "4"}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func installTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, enableDNS bool) {
	helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartPath)

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
	checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{{Name: "sh.helm.release.v1." + releaseName + ".v1", Version: "1"}})
}

func installShouldFailedIfAlreadyInstalledTest(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string, values map[string]interface{}) {
	helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartPath)

	_, err := helmClient.Install(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err == nil {
		t.Fatalf("helm install when release is already installed should failed, but no error was raised")
	}
}

func upgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, enableDNS bool, expectedReleaseVersion int) {
	helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartPath)
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
	helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartPath)

	_, err := helmClient.Upgrade(chartFullPath, releaseName, values, *defaultDeployOpts(t), AuthSettings{})

	if err == nil {
		t.Fatalf("helm upgrade when release is not already installed should failed, but no error was raised")
	}
}

func installOrUpgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, chartPath string, values map[string]interface{}, expectedData map[string]string, expectedVersionLabel string, expectedRelVersion int) {
	helmClient, chartFullPath := buildHelmClient(t, ctx, ns, chartPath)

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
		KubeConfig: ptr.To(kubeconfigPath),
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
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		err := client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: test.ConfigmapName}, cm)
		return err != nil && apierrors.IsNotFound(err)
	}) {
		t.Fatal("configMap has not been removed by helm unsintall")
	}

	// check release has been remove from cluster
	checkExpectedReleases(t, ctx, client, ns, []test.ReleaseStorageInfo{})
}

// buildHelmClient creates Helm client and returns it with the full path of the chart.
//
// if chartPath is a directory then we copy the chart directory into Helm TempDir to avoid any concurrency problem
// when building dependencies and return the full path to the copied chart. (i.e. Helm_TMP_DIR/chartDir)
//
// if chartPath is an archive (i.e. chart.tgz) then we simply returns the path to the chart archive.
func buildHelmClient(t *testing.T, ctx context.Context, ns *corev1.Namespace, chartPath string) (*HelmClient, string) {
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
		KubeConfig: ptr.To(kubeconfigPath),
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

// checkReleaseFailedWithTimeout checks that install / upgrade has failed with timeout error and releaseInfo has failure status.
func checkReleaseFailedWithTimeout(t *testing.T, releaseInfo *release.Release, err error) {
	if err == nil {
		t.Fatalf("expect installation or upgrade failed when timeout is exceeded but not error was raised")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expect context.DeadlineExceeded error. got %s", err)
	}
	if releaseInfo.Info.Status != release.StatusFailed {
		t.Fatalf("expect releaseInfo.Info.Status to be '%s', got '%s'", release.StatusFailed, releaseInfo.Info.Status)
	}
}

// checkServiceDeleted checks that test.SvcName does not exist in namespace "ns" otherwise fails the test.
func checkServiceDeleted(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace) {
	svc := &corev1.Service{}
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		err := client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: test.SvcName}, svc)
		return err != nil && apierrors.IsNotFound(err)
	}) {
		t.Fatal("service has not been removed by helm unsintall or atomic flag.")
	}
}

// checkServiceDeployed checks that test.SvcName exist in namespace "ns" otherwise fails the test.
func checkServiceDeployed(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace) {
	var err error
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		svc := &corev1.Service{}
		err = client.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: test.SvcName}, svc)
		return err == nil
	}) {
		t.Fatalf("service has not been deployed: %s", err)
	}
}

// checkExpectedReleases checks that only expected releases are stored on the clusters.
// Helm releases are stored in secrets (because we use the secret driver) labeled with name=<the name of the release>, version=<the version of the release> and owner=helm.
func checkExpectedReleases(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, ns *corev1.Namespace, expectedReleases []test.ReleaseStorageInfo) {
	t.Helper()

	secrets := &corev1.SecretList{}
	if err := client.List(ctx, secrets, &ctrlruntimeclient.ListOptions{Namespace: ns.Name}, &ctrlruntimeclient.MatchingLabels{"name": releaseName, "owner": "helm"}); err != nil {
		t.Fatalf("failed to list secrets: %s", err)
	}
	test.AssertContainsExactly(t, "", test.MapToReleaseStorageInfo(secrets.Items), expectedReleases)
}

func TestDownloadChart(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	chartArchiveDir := t.TempDir()

	chartGlobPath := path.Join(chartArchiveDir, "*.tgz")
	chartArchiveV1Path, chartArchiveV1Size := test.PackageChart(t, "testdata/examplechart", chartArchiveDir)
	chartArchiveV2Path, chartArchiveV2Size := test.PackageChart(t, "testdata/examplechart-v2", chartArchiveDir)
	chartArchiveV1Name := path.Base(chartArchiveV1Path)
	chartArchiveV2Name := path.Base(chartArchiveV2Path)

	httpRegistryURL := test.StartHTTPRegistryWithCleanup(t, chartGlobPath)
	httpRegistryWithAuthURL := test.StartHTTPRegistryWithAuthAndCleanup(t, chartGlobPath)

	ociRegistryURL := test.StartOciRegistry(t, chartGlobPath)
	ociregistryWithAuthURL, registryConfigFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	testCases := []struct {
		name                string
		repoURL             string
		chartName           string
		chartVersion        string
		auth                AuthSettings
		expectedArchiveName string
		expectedChartSize   int64
		wantErr             bool
	}{
		{
			name:                "Download from HTTP repository should be successful",
			repoURL:             httpRegistryURL,
			chartName:           "examplechart",
			chartVersion:        "0.1.0",
			auth:                AuthSettings{},
			expectedArchiveName: chartArchiveV1Name,
			expectedChartSize:   chartArchiveV1Size,
			wantErr:             false,
		},
		{
			name:                "Download from HTTP repository with empty version should get the latest version",
			repoURL:             httpRegistryURL,
			chartName:           "examplechart",
			chartVersion:        "",
			auth:                AuthSettings{},
			expectedArchiveName: chartArchiveV2Name,
			expectedChartSize:   chartArchiveV2Size,
			wantErr:             false,
		},
		{
			name:                "Download from HTTP repository with auth should be successful",
			repoURL:             httpRegistryWithAuthURL,
			chartName:           "examplechart",
			chartVersion:        "0.1.0",
			auth:                AuthSettings{Username: "username", Password: "password"},
			expectedArchiveName: chartArchiveV1Name,
			expectedChartSize:   chartArchiveV1Size,
			wantErr:             false,
		},
		{
			name:              "Download from HTTP repository should fail when chart does not exist",
			repoURL:           httpRegistryURL,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
		{
			name:              "Download from HTTP repository should fail when version is not a semversion",
			repoURL:           httpRegistryURL,
			chartName:         "examplechart",
			chartVersion:      "notSemver",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},

		{
			name:                "Download from OCI repository should be successful",
			repoURL:             ociRegistryURL,
			chartName:           "examplechart",
			chartVersion:        "0.1.0",
			auth:                AuthSettings{PlainHTTP: true},
			expectedArchiveName: chartArchiveV1Name,
			expectedChartSize:   chartArchiveV1Size,
			wantErr:             false,
		},
		{
			name:                "Download from OCI repository with empty version should get the latest version",
			repoURL:             ociRegistryURL,
			chartName:           "examplechart",
			chartVersion:        "",
			auth:                AuthSettings{PlainHTTP: true},
			expectedArchiveName: chartArchiveV2Name,
			expectedChartSize:   chartArchiveV2Size,
			wantErr:             false,
		},
		{
			name:                "Download from OCI repository with auth should be successful",
			repoURL:             ociregistryWithAuthURL,
			chartName:           "examplechart",
			chartVersion:        "0.1.0",
			auth:                AuthSettings{RegistryConfigFile: registryConfigFile, PlainHTTP: true},
			expectedArchiveName: chartArchiveV1Name,
			expectedChartSize:   chartArchiveV1Size,
			wantErr:             false,
		},
		{
			name:              "Download from OCI repository should fail when chart does not exist",
			repoURL:           ociRegistryURL,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			auth:              AuthSettings{PlainHTTP: true},
			expectedChartSize: 0,
			wantErr:           true,
		},
		{
			name:              "Download from OCI repository should fail when version is not a semversion",
			repoURL:           ociRegistryURL,
			chartName:         "examplechart",
			chartVersion:      "notSemver",
			auth:              AuthSettings{PlainHTTP: true},
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

				chartLoc, err := helmClient.DownloadChart(tc.repoURL, tc.chartName, tc.chartVersion, downloadDest, tc.auth)

				if (err != nil) != tc.wantErr {
					t.Fatalf("DownloadChart() error = %v, wantErr %v", err, tc.wantErr)
				}

				// No need to proceed with further tests if an error is expected.
				if tc.wantErr {
					return
				}

				// Test chart is downloaded where we expect
				expectedChartLoc := downloadDest + "/" + tc.expectedArchiveName
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

	httpRegistryURL := test.StartHTTPRegistryWithCleanup(t, chartGlobPath)
	httpRegistryWithAuthURL := test.StartHTTPRegistryWithAuthAndCleanup(t, chartGlobPath)

	httpsRegistryURL, httpsRegistryPKI := test.StartHTTPSRegistryWithCleanup(t, chartGlobPath)

	ociPlainRegistryURL := test.StartOciRegistry(t, chartGlobPath)
	ociPlainRegistryWithAuthURL, plainRegistryConfigFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	ociSecureRegistryURL, ociSecurePKI := test.StartSecureOciRegistry(t, chartGlobPath)

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
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryURL}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociPlainRegistryURL}},
			hasLockFile:  true,
			auth:         AuthSettings{PlainHTTP: true},
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociPlainRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{PlainHTTP: true},
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
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryURL}, {Name: "examplechart2", Version: "0.1.0", Repository: ociPlainRegistryURL}},
			hasLockFile:  true,
			auth:         AuthSettings{PlainHTTP: true},
			wantErr:      false,
		},
		{
			name:         "http and oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryURL}, {Name: "examplechart2", Version: "0.1.0", Repository: ociPlainRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{PlainHTTP: true},
			wantErr:      false,
		},
		{
			name:         "http dependencies with Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryWithAuthURL}},
			hasLockFile:  true,
			auth:         AuthSettings{Username: "username", Password: "password"},
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryWithAuthURL}},
			hasLockFile:  false,
			auth:         AuthSettings{Username: "username", Password: "password"},
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociPlainRegistryWithAuthURL}},
			hasLockFile:  true,
			auth:         AuthSettings{PlainHTTP: true, RegistryConfigFile: plainRegistryConfigFile},
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociPlainRegistryWithAuthURL}},
			hasLockFile:  false,
			auth:         AuthSettings{PlainHTTP: true, RegistryConfigFile: plainRegistryConfigFile},
			wantErr:      false,
		},
		{
			name:         "http dependency with empty version should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: httpRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "oci dependency with empty version should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: ociPlainRegistryWithAuthURL}},
			hasLockFile:  false,
			auth:         AuthSettings{PlainHTTP: true},
			wantErr:      true,
		},
		{
			name:         "http dependency with non semver should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: httpRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "oci dependency with non semver should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: ociPlainRegistryWithAuthURL}},
			hasLockFile:  false,
			auth:         AuthSettings{PlainHTTP: true},
			wantErr:      true,
		},

		{
			name:         "secure oci registry with a valid certificate",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociSecureRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{CAFile: ociSecurePKI.CAFile},
			wantErr:      false,
		},
		{
			name:         "secure oci registry without a valid certificate",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociSecureRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "secure oci registry without a valid certificate, but ignoring it",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociSecureRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{Insecure: true},
			wantErr:      false,
		},
		{
			name:         "secure oci registry failing to try plain HTTP",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociSecureRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{CAFile: ociSecurePKI.CAFile, PlainHTTP: true},
			wantErr:      true,
		},

		{
			name:         "HTTPS registry with a valid certificate",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpsRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{CAFile: httpsRegistryPKI.CAFile},
			wantErr:      false,
		},
		{
			name:         "HTTPS registry without a valid certificate",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpsRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "HTTPS registry without a valid certificate, but ignoring it",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpsRegistryURL}},
			hasLockFile:  false,
			auth:         AuthSettings{Insecure: true},
			wantErr:      false,
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
