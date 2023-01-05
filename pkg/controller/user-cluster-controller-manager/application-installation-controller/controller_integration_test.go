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

package applicationinstallationcontroller

import (
	"context"
	"fmt"
	"testing"
	"time"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/fake"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1
)

func TestController(t *testing.T) {
	applicationInstallerRecorder := fake.ApplicationInstallerRecorder{}
	ctx, client := startTestEnvWithCleanup(t, &applicationInstallerRecorder)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "when app is created, it should install it and update application.Status with applicationVersion",
			testFunc: func(t *testing.T) {
				appDefName := "app-def-1"
				appInstallName := "app-1"

				def := createApplicationDef(t, ctx, client, appDefName)
				app := createApplicationInstallation(t, ctx, client, appInstallName, appDefName, "1.0.0")

				if !utils.WaitFor(interval, timeout, func() bool {
					if err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app); err != nil {
						return false
					}
					return equality.Semantic.DeepEqual(&def.Spec.Versions[0], app.Status.ApplicationVersion)
				}) {
					t.Fatalf("app.Status.ApplicationVersion differs from expected: %s", diff.ObjectDiff(def.Spec.Versions[0], app.Status.ApplicationVersion))
				}

				expectApplicationInstalledWithVersion(t, &applicationInstallerRecorder, app.Name, def.Spec.Versions[0])
				expectStatusHasConditions(t, ctx, client, app.Name)
			},
		},
		{
			name: "when creating an application that references an ApplicationDefinton that does not exist then nothing should happen",
			testFunc: func(t *testing.T) {
				appDefName := "app-def-2"
				appInstallName := "app-2"

				createApplicationDef(t, ctx, client, appDefName)
				app := createApplicationInstallation(t, ctx, client, appInstallName, "app-def-not-exist", "1.0.0")

				// Ensure application is not deleted.
				if utils.WaitFor(interval, timeout, func() bool {
					err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app)
					return err != nil && apierrors.IsNotFound(err)
				}) {
					t.Fatal("applicationInstallation should not have deen deleted")
				}

				if _, found := applicationInstallerRecorder.ApplyEvents.Load(appInstallName); found {
					t.Fatal("application should not have been uninstalled")
				}
			},
		},
		{
			name: "when creating an application that references an applicationVersion that does not exist then nothing should happen",
			testFunc: func(t *testing.T) {
				appDefName := "app-def-3"
				appInstallName := "app-3"

				createApplicationDef(t, ctx, client, appDefName)
				app := createApplicationInstallation(t, ctx, client, appInstallName, appDefName, "1.0.0-not-exist")

				// Ensure application is not deleted.
				if utils.WaitFor(interval, timeout, func() bool {
					err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app)
					return err != nil && apierrors.IsNotFound(err)
				}) {
					t.Fatal("applicationInstallation should not have deen deleted")
				}

				if _, found := applicationInstallerRecorder.ApplyEvents.Load(appInstallName); found {
					t.Fatal("application should not have been uninstalled")
				}
			},
		},
		{
			name: "when an applicationDefinition is removed then it should remove the application using this ApplicationDefinition",
			testFunc: func(t *testing.T) {
				appDefName := "app-def-5"
				appInstallName := "app-5"

				def := createApplicationDef(t, ctx, client, appDefName)
				createApplicationInstallation(t, ctx, client, appInstallName, appDefName, "1.0.0")
				expectApplicationInstalledWithVersion(t, &applicationInstallerRecorder, appInstallName, def.Spec.Versions[0])

				// Removing applicationDefinition.
				if err := client.Delete(ctx, def); err != nil {
					t.Fatal("failed to delete applicationDefinition")
				}

				// Checking application Installation CR is removed.
				if !utils.WaitFor(interval, timeout, func() bool {
					err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &appskubermaticv1.ApplicationInstallation{})
					return err != nil && apierrors.IsNotFound(err)
				}) {
					t.Fatal("applicationInstallation CR should have been deleted but was not")
				}

				expectApplicationUninstalledWithVersion(t, &applicationInstallerRecorder, appInstallName, def.Spec.Versions[0])
			},
		},
		{
			name: "when an applicationVersion is removed it should remove the application using this appVersion",
			testFunc: func(t *testing.T) {
				appDefName := "app-def-4"
				appInstallName := "app-4"

				def := createApplicationDef(t, ctx, client, appDefName)
				createApplicationInstallation(t, ctx, client, appInstallName, appDefName, "1.0.0")
				expectApplicationInstalledWithVersion(t, &applicationInstallerRecorder, appInstallName, def.Spec.Versions[0])

				previousVersion := def.Spec.Versions[0]

				// Removing applicationVersion from applicationDefinition.
				def.Spec.Versions = []appskubermaticv1.ApplicationVersion{
					{
						Version: "3.0.0",
						Template: appskubermaticv1.ApplicationTemplate{
							Source: appskubermaticv1.ApplicationSource{
								Helm: &appskubermaticv1.HelmSource{
									URL:          "http://helmrepo.local",
									ChartName:    "someChartName",
									ChartVersion: "12",
									Credentials:  nil,
								},
							},
						},
					}}
				if err := client.Update(ctx, def); err != nil {
					t.Fatalf("failed to update applicationDefiniton: %s", err)
				}

				// Checking application Installation CR is removed.
				if !utils.WaitFor(interval, timeout, func() bool {
					err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &appskubermaticv1.ApplicationInstallation{})
					return err != nil && apierrors.IsNotFound(err)
				}) {
					t.Fatal("applicationInstallation CR should have been deleted but was not")
				}

				expectApplicationUninstalledWithVersion(t, &applicationInstallerRecorder, appInstallName, previousVersion)
			},
		},
		{
			name: "when app is updated, it should update app and update application.Status with new applicationVersion",
			testFunc: func(t *testing.T) {
				appDefName := "app-def-5"
				appInstallName := "app-5"

				def := createApplicationDef(t, ctx, client, appDefName)
				app := createApplicationInstallation(t, ctx, client, appInstallName, appDefName, "1.0.0")

				if !utils.WaitFor(interval, timeout, func() bool {
					if err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app); err != nil {
						return false
					}
					return equality.Semantic.DeepEqual(&def.Spec.Versions[0], app.Status.ApplicationVersion)
				}) {
					t.Fatalf("app.Status.ApplicationVersion differs from expected: %s", diff.ObjectDiff(def.Spec.Versions[0], app.Status.ApplicationVersion))
				}

				expectApplicationInstalledWithVersion(t, &applicationInstallerRecorder, app.Name, def.Spec.Versions[0])
				expectStatusHasConditions(t, ctx, client, app.Name)

				// Update application Installation.
				app.Spec.ApplicationRef.Version = "2.0.0"
				if err := client.Update(ctx, &app); err != nil {
					t.Fatalf("failed to update applicationInstallation: %s", err)
				}

				if !utils.WaitFor(interval, timeout, func() bool {
					if err := client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app); err != nil {
						return false
					}
					return equality.Semantic.DeepEqual(&def.Spec.Versions[1], app.Status.ApplicationVersion)
				}) {
					t.Fatalf("app.Status.ApplicationVersion differs from expected: %s", diff.ObjectDiff(def.Spec.Versions[1], app.Status.ApplicationVersion))
				}

				expectApplicationInstalledWithVersion(t, &applicationInstallerRecorder, app.Name, def.Spec.Versions[1])
				expectStatusHasConditions(t, ctx, client, app.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func expectApplicationInstalledWithVersion(t *testing.T, applicationInstallerRecorder *fake.ApplicationInstallerRecorder, appName string, expectedVersion appskubermaticv1.ApplicationVersion) {
	var errReason string
	if !utils.WaitFor(interval, timeout, func() bool {
		if _, found := applicationInstallerRecorder.DownloadEvents.Load(appName); !found {
			errReason = "Application " + appName + "'s sources have not been download"
			return false
		}

		result, found := applicationInstallerRecorder.ApplyEvents.Load(appName)
		if !found {
			errReason = "Application " + appName + " has not been installed"
			return false
		}

		currentVersion := *result.(appskubermaticv1.ApplicationInstallation).Status.ApplicationVersion
		if !equality.Semantic.DeepEqual(expectedVersion, currentVersion) {
			errReason = "app.Status.ApplicationVersion differs from the one that has been installed: " + diff.ObjectDiff(expectedVersion, currentVersion)
			return false
		}
		return true
	}) {
		t.Fatal(errReason)
	}
}

func expectApplicationUninstalledWithVersion(t *testing.T, applicationInstallerRecorder *fake.ApplicationInstallerRecorder, appName string, expectedVersion appskubermaticv1.ApplicationVersion) {
	var errReason string
	if !utils.WaitFor(interval, timeout, func() bool {
		result, found := applicationInstallerRecorder.DeleteEvents.Load(appName)
		if !found {
			errReason = "Application " + appName + " has not been uninstalled"
			return false
		}

		currentVersion := *result.(appskubermaticv1.ApplicationInstallation).Status.ApplicationVersion
		if !equality.Semantic.DeepEqual(expectedVersion, currentVersion) {
			errReason = "version selected for the deletion differ from expected " + diff.ObjectDiff(expectedVersion, currentVersion)
			return false
		}
		return true
	}) {
		t.Fatal(errReason)
	}
}

func expectStatusHasConditions(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, appName string) {
	app := &appskubermaticv1.ApplicationInstallation{}
	var errReason string
	if !utils.WaitFor(interval, timeout, func() bool {
		if err := client.Get(ctx, types.NamespacedName{Name: appName, Namespace: applicationNamespace}, app); err != nil {
			errReason = err.Error()
			return false
		}
		if cond, found := app.Status.Conditions[appskubermaticv1.ManifestsRetrieved]; !found || cond.Status != corev1.ConditionTrue {
			errReason = fmt.Sprintf("condition %s is not True", appskubermaticv1.ManifestsRetrieved)
			return false
		}
		if cond, found := app.Status.Conditions[appskubermaticv1.Ready]; !found || cond.Status != corev1.ConditionTrue {
			errReason = fmt.Sprintf("condition %s is not True", appskubermaticv1.Ready)
			return false
		}
		return true
	}) {
		t.Fatal(errReason)
	}
}

func createApplicationDef(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, appDefName string) *appskubermaticv1.ApplicationDefinition {
	if err := client.Create(ctx, genApplicationDefinition(appDefName)); err != nil {
		t.Fatalf("failed to create applicationDefinition: %s", err)
	}

	def := &appskubermaticv1.ApplicationDefinition{}
	if !utils.WaitFor(interval, timeout, func() bool {
		return client.Get(ctx, types.NamespacedName{Name: appDefName}, def) == nil
	}) {
		t.Fatal("failed to get applicationDefinition")
	}
	return def
}

func createApplicationInstallation(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, appInstallName string, appDefName string, version string) appskubermaticv1.ApplicationInstallation {
	// Create applicationInstallation.
	if err := client.Create(ctx, genApplicationInstallation(appInstallName, appDefName, version, 0)); err != nil {
		t.Fatalf("failed to create applicationInstallation: %s", err)
	}

	// Wait for application to be created.
	app := appskubermaticv1.ApplicationInstallation{}
	if !utils.WaitFor(interval, 3*time.Second, func() bool {
		return client.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app) == nil
	}) {
		t.Fatal("failed to create get applicationInstallation")
	}
	return app
}

func startTestEnvWithCleanup(t *testing.T, applicationInstaller *fake.ApplicationInstallerRecorder) (context.Context, ctrlruntimeclient.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	// Bootstrapping test environment.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../crd/k8c.io"},
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

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationNamespace,
		},
	}
	if err := client.Create(ctx, ns); err != nil {
		t.Fatalf("failed to create namespace")
	}

	isClusterPausedFunc := func(ctx context.Context) (bool, error) { return false, nil }

	if err := Add(ctx, kubermaticlog.Logger, mgr, mgr, isClusterPausedFunc, applicationInstaller); err != nil {
		t.Fatalf("failed to add controller to manager: %s", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("failed to start manager: %s", err)
			return
		}
	}()

	t.Cleanup(func() {
		// Delete ns if it exists.
		err = client.Delete(ctx, ns)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("failed to delete namespace: %s", err)
		}

		// Clean up and stop controller.
		cancel()

		// Tearing down the test environment.
		if err := testEnv.Stop(); err != nil {
			t.Fatalf("failed to stop testEnv: %s", err)
		}
	})

	return ctx, client
}
