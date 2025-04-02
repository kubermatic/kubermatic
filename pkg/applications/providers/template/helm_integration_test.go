//go:build integration

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

package template

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	"k8c.io/kubermatic/v2/pkg/applications/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1
)

var (
	kubeconfigPath string
)

type Source string

var (
	git  Source = "git"
	helm Source = "helm"
)

func TestHelmProvider(t *testing.T) {
	const exampleChartLoc = "../../helmclient/testdata/examplechart"
	const exampleV2ChartLoc = "../../helmclient/testdata/examplechart-v2"
	var ctx context.Context
	var client ctrlruntimeclient.Client
	ctx, client, kubeconfigPath = test.StartTestEnvWithCleanup(t, "../../../crd/k8c.io")

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(kubermaticlog.Logger.Desugar().WithOptions(zap.AddCallerSkip(1))))

	// package and upload examplechart2 to test registry
	chartDir := t.TempDir()
	chartGlobPath := path.Join(chartDir, "*.tgz")
	test.PackageChart(t, "../../helmclient/testdata/examplechart2", chartDir)
	registryURL, regCredFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "when an application is created with no values, it should install app with default values",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil, "", nil, helm)

				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, test.DefaultData, test.DefaultVersionLabel, 1, false)
			},
		},
		{
			name: "when an application is created with customCmData, it should install app with customCmData",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				customCmData := map[string]string{"hello": "world", "a": "b"}
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.CmDataKey, customCmData), "", nil, helm)

				appendDefaultValues(customCmData, test.DefaultData) // its check that object values are merged with default object values
				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, customCmData, test.DefaultVersionLabel, 1, false)
			},
		},
		{
			name: "when an application is created with custom versionLabel, it should install app into user cluster with custom versionLabel",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				// its check that scalar values overwrite default  scalar values
				customVersionLabel := "1.2.3"
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.VersionLabelKey, customVersionLabel), "", nil, helm)

				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, test.DefaultData, customVersionLabel, 1, false)
			},
		},
		{
			name: "when an application is is updated with customCmData, should update app with new data",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				customCmData := map[string]string{"hello": "world", "a": "b"}
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.CmDataKey, customCmData), "", nil, helm)

				appendDefaultValues(customCmData, test.DefaultData)
				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, customCmData, test.DefaultVersionLabel, 1, false)

				// Upgrade application
				newCustomCmData := map[string]string{"c": "d", "e": "f"}
				app.Spec.Values.Raw = toHelmRawValues(t, test.CmDataKey, newCustomCmData)
				appendDefaultValues(newCustomCmData, test.DefaultData)
				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, newCustomCmData, test.DefaultVersionLabel, 2, false)
			},
		},
		{
			name: "when an application removed, it should uninstall app",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				customCmData := map[string]string{"hello": "world", "a": "b"}
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.CmDataKey, customCmData), "", nil, helm)

				appendDefaultValues(customCmData, test.DefaultData)
				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, customCmData, test.DefaultVersionLabel, 1, false)

				// test uninstall app
				template := HelmTemplate{
					Ctx:             context.Background(),
					Kubeconfig:      kubeconfigPath,
					CacheDir:        t.TempDir(),
					Log:             kubermaticlog.Logger,
					SecretNamespace: "abc",
					SeedClient:      client,
				}

				statusUpdater, err := template.Uninstall(app)
				if err != nil {
					t.Fatalf("failed to uninstall app: %s", err)
				}
				statusUpdater(&app.Status)

				// check configmap is removed
				cm := &corev1.ConfigMap{}
				if !utils.WaitFor(ctx, interval, timeout, func() bool {
					err := client.Get(ctx, types.NamespacedName{Namespace: testNs.Name, Name: test.ConfigmapName}, cm)
					return err != nil && apierrors.IsNotFound(err)
				}) {
					t.Fatal("configMap has not been removed when unsintalling app")
				}

				assertStatusIsUpdated(t, app, statusUpdater, 1)
			},
		},
		// The following tests ensure dependencies with credentials are correctly handled.
		// c.f. https://github.com/kubermatic/kubermatic/issues/10840 and https://github.com/kubermatic/kubermatic/issues/10845
		{
			name: "application installation should be successful when dependency requires credentials that are provided",
			testFunc: func(t *testing.T) {
				chartFullPath := createChartWithDependency(t, registryURL)
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil, "", nil, helm)

				app.Status.ApplicationVersion.Template.DependencyCredentials = &appskubermaticv1.DependencyCredentials{HelmCredentials: &appskubermaticv1.HelmCredentials{RegistryConfigFile: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "registry-secret"},
					Key:                  "cred",
					Optional:             nil,
				}}}

				createCredentialSecret(t, ctx, client, regCredFile)

				// Test chart is installed.
				installOrUpgradeTest(t, ctx, client, testNs, app, chartFullPath, test.DefaultData, test.DefaultVersionLabel, 1, false)

				// Ensure dependency has been installed too.
				cm := &corev1.ConfigMap{}
				var errorGet error
				if !utils.WaitFor(ctx, interval, timeout, func() bool {
					errorGet = client.Get(ctx, types.NamespacedName{Namespace: testNs.Name, Name: test.ConfigmapName2}, cm)
					return errorGet == nil
				}) {
					t.Fatalf("configMap2 has not been installed by helm. last error: %s", errorGet)
				}
			},
		},
		{
			name: "application installation should fail when dependency requires credentials that are not provided",
			testFunc: func(t *testing.T) {
				chartFullPath := createChartWithDependency(t, registryURL)
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil, "", nil, helm)

				template := HelmTemplate{
					Ctx:             context.Background(),
					Kubeconfig:      kubeconfigPath,
					CacheDir:        t.TempDir(),
					Log:             kubermaticlog.Logger,
					SecretNamespace: "default",
					SeedClient:      client,
				}

				_, err := template.InstallOrUpgrade(chartFullPath, &appskubermaticv1.ApplicationDefinition{}, app)
				if err == nil {
					t.Fatal("install of application should have failed but no error was raised")
				}
			},
		},
		// these tests ensure deploysOpt defined in CR are correctly passed to the helmClient
		{
			name: "application installation should fail when timeout is exceeded (timeout is defined at application Installation Level)",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				deployOpts := &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Wait: true, Timeout: metav1.Duration{Duration: 5 * time.Second}, Atomic: false}}

				// Create an application that deploy a LB service that will never get public ip -> helm release will not be be successful.
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.DeploySvcKey, true), "", deployOpts, helm)

				template := HelmTemplate{
					Ctx:             context.Background(),
					Kubeconfig:      kubeconfigPath,
					CacheDir:        t.TempDir(),
					Log:             kubermaticlog.Logger,
					SecretNamespace: "default",
					SeedClient:      client,
					ClusterName:     "cluster-default",
				}

				statusUpdater, err := template.InstallOrUpgrade(exampleV2ChartLoc, &appskubermaticv1.ApplicationDefinition{}, app)
				if err == nil {
					t.Fatalf("expect installation or upgrade failed when timeout is exceeded but not error was raised")
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expect context.DeadlineExceeded error. got %s", err)
				}

				// Check that status of helm release is set to failed.
				statusUpdater(&app.Status)
				if app.Status.HelmRelease == nil {
					t.Fatal("app.Status.HelmRelease should not be nil")
				}

				if app.Status.HelmRelease.Info.Status != appskubermaticv1.HelmReleaseStatus(release.StatusFailed) {
					t.Fatalf("expect releaseInfo.Info.Status to be '%s', got '%s'", release.StatusFailed, app.Status.HelmRelease.Info.Status)
				}
			},
		},
		{
			name: "application installation should fail when timeout is exceeded (timeout is defined at applicationDefinition Level)",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				deployOpts := &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Wait: true, Timeout: metav1.Duration{Duration: 5 * time.Second}, Atomic: false}}

				// Create an application that deploy a LB service that will never get public ip -> helm release will not be be successful.
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.DeploySvcKey, true), "", nil, helm)

				template := HelmTemplate{
					Ctx:             context.Background(),
					Kubeconfig:      kubeconfigPath,
					CacheDir:        t.TempDir(),
					Log:             kubermaticlog.Logger,
					SecretNamespace: "default",
					SeedClient:      client,
					ClusterName:     "cluster-default",
				}

				statusUpdater, err := template.InstallOrUpgrade(exampleV2ChartLoc, &appskubermaticv1.ApplicationDefinition{Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: deployOpts}}, app)
				if err == nil {
					t.Fatalf("expect installation or upgrade failed when timeout is exceeded but not error was raised")
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expect context.DeadlineExceeded error. got %s", err)
				}

				// Check that status of helm release is set to failed.
				statusUpdater(&app.Status)
				if app.Status.HelmRelease == nil {
					t.Fatal("app.Status.HelmRelease should not be nil")
				}

				if app.Status.HelmRelease.Info.Status != appskubermaticv1.HelmReleaseStatus(release.StatusFailed) {
					t.Fatalf("expect releaseInfo.Info.Status to be '%s', got '%s'", release.StatusFailed, app.Status.HelmRelease.Info.Status)
				}
			},
		},
		{
			name: "when an application is created with no values and enableDNS=true, it should install app with default values and dns should be resolved",
			testFunc: func(t *testing.T) {
				deployOpts := &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Wait: false, Timeout: metav1.Duration{Duration: 0}, Atomic: false, EnableDNS: true}}
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil, "", deployOpts, helm)

				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, test.DefaultData, test.DefaultVersionLabel, 1, true)
			},
		},
		{
			name: "when an application is created with valuesBlock, it is installed with values from valuesBlock",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil, "key: value", nil, helm)

				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, test.DefaultData, test.DefaultVersionLabel, 1, false)
			},
		},
		{
			name: "when an application with git source is created with no values, it should install app with default values",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				test.EnsureClusterWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil, "", nil, git)

				installOrUpgradeTest(t, ctx, client, testNs, app, exampleChartLoc, test.DefaultData, test.DefaultVersionLabel, 1, false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func installOrUpgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, testNS *corev1.Namespace, app *appskubermaticv1.ApplicationInstallation, chartLoc string, expectedData map[string]string, expectedVersionLabel string, expectedVersion int, enableDNS bool) {
	template := HelmTemplate{
		Ctx:             context.Background(),
		Kubeconfig:      kubeconfigPath,
		CacheDir:        t.TempDir(),
		Log:             kubermaticlog.Logger,
		SecretNamespace: "default",
		ClusterName:     "cluster-default",
		SeedClient:      client,
	}

	statusUpdater, err := template.InstallOrUpgrade(chartLoc, &appskubermaticv1.ApplicationDefinition{}, app)
	if err != nil {
		t.Fatalf("failed to install or upgrade chart: %s", err)
	}
	statusUpdater(&app.Status)

	test.CheckConfigMap(t, ctx, client, testNS, expectedData, expectedVersionLabel, enableDNS)
	assertStatusIsUpdated(t, app, statusUpdater, expectedVersion)
}
func createApplicationInstallation(testNs *corev1.Namespace, rawValues []byte, rawValuesBlock string, deployOpts *appskubermaticv1.DeployOptions, source Source) *appskubermaticv1.ApplicationInstallation {
	app := &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "app1",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &appskubermaticv1.AppNamespaceSpec{
				Name: testNs.Name,
			},
			Values:        runtime.RawExtension{},
			DeployOptions: deployOpts,
		},
		Status: appskubermaticv1.ApplicationInstallationStatus{
			Method: appskubermaticv1.HelmTemplateMethod,
			ApplicationVersion: &appskubermaticv1.ApplicationVersion{
				Version: "0.1.0",
				Template: appskubermaticv1.ApplicationTemplate{
					Source: appskubermaticv1.ApplicationSource{},
				},
			},
		},
	}
	if rawValues != nil {
		app.Spec.Values.Raw = rawValues
	}

	if rawValuesBlock != "" {
		app.Spec.ValuesBlock = rawValuesBlock
	}

	if source == helm {
		app.Status.ApplicationVersion.Template.Source.Helm = &appskubermaticv1.HelmSource{
			URL:          "localhost",
			ChartName:    "example",
			ChartVersion: "0.1.0",
			PlainHTTP:    ptr.To(true),
		}
	}

	if source == git {
		app.Status.ApplicationVersion.Template.Source.Git = &appskubermaticv1.GitSource{
			Remote: "../../helmclient/testdata/examplechart-git",
			Ref: appskubermaticv1.GitReference{
				Branch: "main",
			},
		}
	}

	return app
}

func assertStatusIsUpdated(t *testing.T, app *appskubermaticv1.ApplicationInstallation, statusUpdater util.StatusUpdater, expectedVersion int) {
	t.Helper()

	if app.Status.HelmRelease == nil {
		t.Fatal("app.Status.HelmRelease should not be nil")
	}
	expectedRelName := getReleaseName(app)
	if app.Status.HelmRelease.Name != expectedRelName {
		t.Errorf("app.Status.HelmRelease.Name. expected '%s', got '%s'", expectedRelName, app.Status.HelmRelease.Name)
	}
	if app.Status.HelmRelease.Version != expectedVersion {
		t.Errorf("invalid app.Status.HelmRelease.Version. expected '%d', got '%d'", expectedVersion, app.Status.HelmRelease.Version)
	}
	if app.Status.HelmRelease.Info == nil {
		t.Error(" app.Status.HelmRelease.Info should not be nil")
	}
}

// appendDefaultValues merges the source with the defaultValues by simply copy key, values of defaultValues into source.
func appendDefaultValues(source map[string]string, defaultValues map[string]string) {
	for k, v := range defaultValues {
		source[k] = v
	}
}

// toHelmRawValues build the helm value map and transforms it to runtime.RawExtension.Raw.
// Key is the key (i.e. name) in the value.yaml and values it's corresponding value.
// example:
// toHelmRawValues("cmData", map[string]string{"hello": "world", "a": "b"}) produces this helm value file
// cmData:
//
//	hello: world
//	a: b
func toHelmRawValues(t *testing.T, key string, values any) []byte {
	helmValues := map[string]any{key: values}
	rawValues, err := json.Marshal(helmValues)
	if err != nil {
		t.Fatalf("failed to create raw values: %s", err)
	}
	return rawValues
}

// createCredentialSecret creates the secret that holds credentials to the helm registry.
func createCredentialSecret(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, regCredFile string) {
	t.Helper()
	credAsByte, err := os.ReadFile(regCredFile)
	if err != nil {
		t.Fatalf("failed tor ead registry credentials file: %s", err)
	}

	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "registry-secret",
		},
		Data: map[string][]byte{"cred": credAsByte},
	}
	if err := client.Create(ctx, credSecret); err != nil {
		t.Fatalf("failed to create secret that contains registry credentials: %s", err)
	}
}

// createChartWithDependency takes testdata/examplechart as base and adds examplechart2 as a dependency. The full path to the chart directory is returned.
// The dependency is stored on the helm registry accessible by registryUrl.
func createChartWithDependency(t *testing.T, registryURL string) string {
	// copy exampleChart  and add examplechart2 as dependency
	chartFullPath := path.Join(t.TempDir(), "chartWithDependencies")
	if err := test.CopyDir("../../helmclient/testdata/examplechart", chartFullPath); err != nil {
		t.Fatalf("failed to copy chart directory to temp dir: %s", err)
	}

	chartToInstall, err := loader.Load(chartFullPath)
	if err != nil {
		t.Fatalf("failed to load chart: %s", err)
	}
	chartToInstall.Metadata.Dependencies = []*chart.Dependency{{Name: "examplechart2", Version: "0.1.0", Repository: registryURL}}
	// Save the chart file.
	out, err := yaml.Marshal(chartToInstall.Metadata)
	if err != nil {
		t.Fatalf("failed to Marshal chart :%s", err)
	}
	if err := os.WriteFile(filepath.Join(chartFullPath, chartutil.ChartfileName), out, 0644); err != nil {
		t.Fatalf("failed to write chart.yaml with dependency")
	}
	return chartFullPath
}
