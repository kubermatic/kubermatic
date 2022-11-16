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
	"testing"
	"time"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	"k8c.io/kubermatic/v2/pkg/applications/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1

	chartLoc = "../../helmclient/testdata/examplechart"
)

var (
	kubeconfigPath string
)

func TestHelmProvider(t *testing.T) {
	var ctx context.Context
	var client ctrlruntimeclient.Client
	ctx, client, kubeconfigPath = test.StartTestEnvWithCleanup(t, "../../../crd/k8c.io")

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "when an application is created with no values, it should install app with default values",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				app := createApplicationInstallation(testNs, nil)

				installorUpgradeTest(t, ctx, client, testNs, app, test.DefaultData, test.DefaultVerionLabel, 1)
			},
		},
		{
			name: "when an application is created with customCmData, it should install app with customCmData",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				customCmData := map[string]string{"hello": "world", "a": "b"}
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.CmDataKey, customCmData))

				appendDefaultValues(customCmData, test.DefaultData) // its check that object values are merged with default object values
				installorUpgradeTest(t, ctx, client, testNs, app, customCmData, test.DefaultVerionLabel, 1)
			},
		},
		{
			name: "when an application is created with custom versionLabel, it should install app into user cluster with custom versionLabel",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				// its check that scalar values overwrite default  scalar values
				customVersionLabel := "1.2.3"
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.VersionLabelKey, customVersionLabel))

				installorUpgradeTest(t, ctx, client, testNs, app, test.DefaultData, customVersionLabel, 1)
			},
		},
		{
			name: "when an application is is updated with customCmData, should update app with new data",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				customCmData := map[string]string{"hello": "world", "a": "b"}
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.CmDataKey, customCmData))

				appendDefaultValues(customCmData, test.DefaultData)
				installorUpgradeTest(t, ctx, client, testNs, app, customCmData, test.DefaultVerionLabel, 1)

				// Upgrade application
				newCustomCmData := map[string]string{"c": "d", "e": "f"}
				app.Spec.Values.Raw = toHelmRawValues(t, test.CmDataKey, newCustomCmData)
				appendDefaultValues(newCustomCmData, test.DefaultData)
				installorUpgradeTest(t, ctx, client, testNs, app, newCustomCmData, test.DefaultVerionLabel, 2)
			},
		},
		{
			name: "when an application removed, it should uninstall app",
			testFunc: func(t *testing.T) {
				testNs := test.CreateNamespaceWithCleanup(t, ctx, client)
				customCmData := map[string]string{"hello": "world", "a": "b"}
				app := createApplicationInstallation(testNs, toHelmRawValues(t, test.CmDataKey, customCmData))

				appendDefaultValues(customCmData, test.DefaultData)
				installorUpgradeTest(t, ctx, client, testNs, app, customCmData, test.DefaultVerionLabel, 1)

				// test uninstall app
				template := HelmTemplate{
					Ctx:                     context.Background(),
					Kubeconfig:              kubeconfigPath,
					CacheDir:                t.TempDir(),
					Log:                     kubermaticlog.Logger,
					ApplicationInstallation: app,
					SecretNamespace:         "abc",
					SeedClient:              client,
				}

				statusUpdater, err := template.Uninstall(app)
				if err != nil {
					t.Fatalf("failed to uninstall app: %s", err)
				}
				statusUpdater(&app.Status)

				//check configmap is removed
				cm := &corev1.ConfigMap{}
				if !utils.WaitFor(interval, timeout, func() bool {
					err := client.Get(ctx, types.NamespacedName{Namespace: testNs.Name, Name: test.ConfigmapName}, cm)
					return err != nil && apierrors.IsNotFound(err)
				}) {
					t.Fatal("configMap has not been removed when unsintalling app")
				}

				assertStatusIsUpdated(t, app, statusUpdater, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func installorUpgradeTest(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, testNs *corev1.Namespace, app *appskubermaticv1.ApplicationInstallation, expectedData map[string]string, expectedVersionLabel string, expectedVersion int) {
	template := HelmTemplate{
		Ctx:                     context.Background(),
		Kubeconfig:              kubeconfigPath,
		CacheDir:                t.TempDir(),
		Log:                     kubermaticlog.Logger,
		ApplicationInstallation: app,
		SecretNamespace:         "abc",
		SeedClient:              client,
	}

	statusUpdater, err := template.InstallOrUpgrade(chartLoc, app)
	if err != nil {
		t.Fatalf("failed to install or upgrade chart: %s", err)
	}
	statusUpdater(&app.Status)

	test.CheckConfigMap(t, ctx, client, testNs, expectedData, expectedVersionLabel)
	assertStatusIsUpdated(t, app, statusUpdater, expectedVersion)
}
func createApplicationInstallation(testNs *corev1.Namespace, rawValues []byte) *appskubermaticv1.ApplicationInstallation {
	app := &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "app1",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name: testNs.Name,
			},
			Values: runtime.RawExtension{},
		},
		Status: appskubermaticv1.ApplicationInstallationStatus{
			Method: appskubermaticv1.HelmTemplateMethod,
			ApplicationVersion: &appskubermaticv1.ApplicationVersion{
				Version: "0.1.0",
				Template: appskubermaticv1.ApplicationTemplate{
					Source: appskubermaticv1.ApplicationSource{
						Helm: &appskubermaticv1.HelmSource{
							URL:          "localhost",
							ChartName:    "example",
							ChartVersion: "0.1.0",
						},
					},
				},
			},
		},
	}
	if rawValues != nil {
		app.Spec.Values.Raw = rawValues
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
