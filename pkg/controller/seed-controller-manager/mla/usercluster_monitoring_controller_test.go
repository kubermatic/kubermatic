/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// fakeUserClusterClientProvider satisfies UserClusterClientProvider for tests.
type fakeUserClusterClientProvider struct {
	client ctrlruntimeclient.Client
}

func (f *fakeUserClusterClientProvider) GetClient(_ context.Context, _ *kubermaticv1.Cluster, _ ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}

func makeAddon(name, clusterNamespace string) *kubermaticv1.Addon {
	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: clusterNamespace,
		},
	}
}

func makeHealthyCluster(name string, monitoringEnabled bool) *kubermaticv1.Cluster {
	var mlaSpec *kubermaticv1.MLASettings
	if monitoringEnabled {
		mlaSpec = &kubermaticv1.MLASettings{MonitoringEnabled: true}
	}
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticv1.ClusterSpec{
			MLA: mlaSpec,
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-" + name,
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver:                    kubermaticv1.HealthStatusUp,
				Scheduler:                    kubermaticv1.HealthStatusUp,
				Controller:                   kubermaticv1.HealthStatusUp,
				Etcd:                         kubermaticv1.HealthStatusUp,
				CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
				UserClusterControllerManager: kubermaticv1.HealthStatusUp,
				ApplicationController:        kubermaticv1.HealthStatusUp,
			},
		},
	}
}

func makeAppDef(name, version, defaultValuesBlock string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Versions: []appskubermaticv1.ApplicationVersion{
				{Version: version},
			},
			DefaultValuesBlock: defaultValuesBlock,
		},
	}
}

func makeExistingAppInstallation(name, namespace, valuesBlock string) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
			},
			ResourceVersion: "1",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			ValuesBlock: valuesBlock,
		},
	}
}

func newTestUserClusterMonitoringReconciler(seedObjects []ctrlruntimeclient.Object, userClusterObjects []ctrlruntimeclient.Object) *userClusterMonitoringReconciler {
	seedClient := fake.NewClientBuilder().WithObjects(seedObjects...).Build()
	userClusterClient := fake.NewClientBuilder().WithObjects(userClusterObjects...).Build()

	return &userClusterMonitoringReconciler{
		Client:                        seedClient,
		log:                           kubermaticlog.Logger,
		versions:                      kubermatic.GetFakeVersions(),
		recorder:                      events.NewFakeRecorder(10),
		userClusterConnectionProvider: &fakeUserClusterClientProvider{client: userClusterClient},
	}
}

func runReconcile(t *testing.T, cluster *kubermaticv1.Cluster, seedObjects []ctrlruntimeclient.Object, userClusterObjects []ctrlruntimeclient.Object) (ctrlruntimeclient.Client, error) {
	t.Helper()
	allSeed := make([]ctrlruntimeclient.Object, len(seedObjects)+1)
	copy(allSeed, seedObjects)
	allSeed[len(seedObjects)] = cluster
	r := newTestUserClusterMonitoringReconciler(allSeed, userClusterObjects)
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: cluster.Name},
	})
	return r.userClusterConnectionProvider.(*fakeUserClusterClientProvider).client, err
}

// assertInstalled fetches and returns the named ApplicationInstallation, failing the test if absent.
func assertInstalled(t *testing.T, client ctrlruntimeclient.Client, name, namespace string) *appskubermaticv1.ApplicationInstallation {
	t.Helper()
	app := &appskubermaticv1.ApplicationInstallation{}
	if err := client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, app); err != nil {
		t.Fatalf("%s/%s ApplicationInstallation not found: %v", namespace, name, err)
	}
	return app
}

// assertNotInstalled fails the test if the named ApplicationInstallation exists.
func assertNotInstalled(t *testing.T, client ctrlruntimeclient.Client, name, namespace string) {
	t.Helper()
	app := &appskubermaticv1.ApplicationInstallation{}
	if err := client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, app); err == nil {
		t.Errorf("%s/%s ApplicationInstallation should not exist", namespace, name)
	}
}

// assertInstallCount fails the test if the total number of ApplicationInstallations differs from want.
func assertInstallCount(t *testing.T, client ctrlruntimeclient.Client, want int) {
	t.Helper()
	list := &appskubermaticv1.ApplicationInstallationList{}
	if err := client.List(context.Background(), list); err != nil {
		t.Fatalf("failed to list ApplicationInstallations: %v", err)
	}
	if len(list.Items) != want {
		t.Errorf("expected %d ApplicationInstallations, got %d", want, len(list.Items))
	}
}

// assertValuesSeeded fails the test if the ApplicationInstallation has no values set.
func assertValuesSeeded(t *testing.T, app *appskubermaticv1.ApplicationInstallation) {
	t.Helper()
	values, err := app.Spec.GetParsedValues()
	if err != nil {
		t.Fatalf("failed to parse values for %s: %v", app.Name, err)
	}
	if len(values) == 0 {
		t.Errorf("%s ValuesBlock should be seeded from defaultValuesBlock, but is empty", app.Name)
	}
}

// assertValueKey fails the test if the ApplicationInstallation values do not contain the given key.
func assertValueKey(t *testing.T, app *appskubermaticv1.ApplicationInstallation, key string) {
	t.Helper()
	values, err := app.Spec.GetParsedValues()
	if err != nil {
		t.Fatalf("failed to parse values for %s: %v", app.Name, err)
	}
	if _, ok := values[key]; !ok {
		t.Errorf("%s values should contain key %q, but it is missing", app.Name, key)
	}
}

const (
	testClusterName = "test-cluster"
	neVersion       = "1.10.2"
	ksmVersion      = "2.18.0"
	neDefValues     = "tolerations:\n- effect: NoExecute\n  operator: Exists\n"
	ksmDefValues    = "resources:\n  limits:\n    cpu: 100m\n"
	customValues    = "customKey: customValue\n"
)

func bothAppDefs() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		makeAppDef(nodeExporterAppName, neVersion, neDefValues),
		makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues),
	}
}

func TestUserClusterMonitoringReconcile_MonitoringDisabled(t *testing.T) {
	t.Parallel()

	t.Run("no applications installed when both AppDefs exist", func(t *testing.T) {
		t.Parallel()
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, false), bothAppDefs(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertInstallCount(t, client, 0)
	})

	t.Run("existing KKP-managed installations are removed", func(t *testing.T) {
		t.Parallel()
		existing := []ctrlruntimeclient.Object{
			makeExistingAppInstallation(nodeExporterAppName, nodeExporterNamespace, neDefValues),
			makeExistingAppInstallation(kubeStateMetricsAppName, kubeStateMetricsNamespace, ksmDefValues),
		}
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, false), nil, existing)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertInstallCount(t, client, 0)
	})

	t.Run("non-KKP-managed installation is left alone", func(t *testing.T) {
		t.Parallel()
		existing := []ctrlruntimeclient.Object{
			&appskubermaticv1.ApplicationInstallation{
				ObjectMeta: metav1.ObjectMeta{
					Name:            nodeExporterAppName,
					Namespace:       nodeExporterNamespace,
					ResourceVersion: "1",
				},
			},
		}
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, false), nil, existing)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertInstallCount(t, client, 1)
	})
}

func TestUserClusterMonitoringReconcile_AppDefMissing(t *testing.T) {
	t.Parallel()

	t.Run("node-exporter AppDef missing: only kube-state-metrics installed", func(t *testing.T) {
		t.Parallel()
		seedObjs := []ctrlruntimeclient.Object{makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues)}
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, true), seedObjs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertNotInstalled(t, client, nodeExporterAppName, nodeExporterNamespace)
		assertInstalled(t, client, kubeStateMetricsAppName, kubeStateMetricsNamespace)
	})

	t.Run("kube-state-metrics AppDef missing: only node-exporter installed", func(t *testing.T) {
		t.Parallel()
		seedObjs := []ctrlruntimeclient.Object{makeAppDef(nodeExporterAppName, neVersion, neDefValues)}
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, true), seedObjs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertInstalled(t, client, nodeExporterAppName, nodeExporterNamespace)
		assertNotInstalled(t, client, kubeStateMetricsAppName, kubeStateMetricsNamespace)
	})
}

func TestUserClusterMonitoringReconcile_Install(t *testing.T) {
	t.Parallel()

	t.Run("both applications installed with defaultValuesBlock seeded", func(t *testing.T) {
		t.Parallel()
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, true), bothAppDefs(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		neApp := assertInstalled(t, client, nodeExporterAppName, nodeExporterNamespace)
		if neApp.Spec.ApplicationRef.Version != neVersion {
			t.Errorf("node-exporter version: got %q, want %q", neApp.Spec.ApplicationRef.Version, neVersion)
		}
		if neApp.Labels[appskubermaticv1.ApplicationManagedByLabel] != appskubermaticv1.ApplicationManagedByKKPValue {
			t.Errorf("node-exporter missing managed-by label")
		}
		assertValuesSeeded(t, neApp)

		ksmApp := assertInstalled(t, client, kubeStateMetricsAppName, kubeStateMetricsNamespace)
		if ksmApp.Spec.ApplicationRef.Version != ksmVersion {
			t.Errorf("kube-state-metrics version: got %q, want %q", ksmApp.Spec.ApplicationRef.Version, ksmVersion)
		}
		if ksmApp.Labels[appskubermaticv1.ApplicationManagedByLabel] != appskubermaticv1.ApplicationManagedByKKPValue {
			t.Errorf("kube-state-metrics missing managed-by label")
		}
		assertValuesSeeded(t, ksmApp)
	})

	t.Run("existing installation preserves user-customized values", func(t *testing.T) {
		t.Parallel()
		existing := []ctrlruntimeclient.Object{
			makeExistingAppInstallation(nodeExporterAppName, nodeExporterNamespace, customValues),
			makeExistingAppInstallation(kubeStateMetricsAppName, kubeStateMetricsNamespace, customValues),
		}
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, true), bothAppDefs(), existing)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertValueKey(t, assertInstalled(t, client, nodeExporterAppName, nodeExporterNamespace), "customKey")
		assertValueKey(t, assertInstalled(t, client, kubeStateMetricsAppName, kubeStateMetricsNamespace), "customKey")
	})

	t.Run("AppDef with defaultVersion uses that version", func(t *testing.T) {
		t.Parallel()
		neAd := makeAppDef(nodeExporterAppName, "1.0.0", neDefValues)
		neAd.Spec.DefaultVersion = "1.99.0"
		neAd.Spec.Versions = append(neAd.Spec.Versions, appskubermaticv1.ApplicationVersion{Version: "1.99.0"})
		seedObjs := []ctrlruntimeclient.Object{neAd, makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues)}

		client, err := runReconcile(t, makeHealthyCluster(testClusterName, true), seedObjs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		neApp := assertInstalled(t, client, nodeExporterAppName, nodeExporterNamespace)
		if neApp.Spec.ApplicationRef.Version != "1.99.0" {
			t.Errorf("expected defaultVersion 1.99.0, got %q", neApp.Spec.ApplicationRef.Version)
		}
	})
}

func TestUserClusterMonitoringReconcile_AddonMigration(t *testing.T) {
	t.Parallel()

	t.Run("existing Addon CR skips ApplicationInstallation for that app only", func(t *testing.T) {
		t.Parallel()
		seedObjs := append(bothAppDefs(), makeAddon(nodeExporterAppName, "cluster-"+testClusterName))
		client, err := runReconcile(t, makeHealthyCluster(testClusterName, true), seedObjs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertNotInstalled(t, client, nodeExporterAppName, nodeExporterNamespace)
		assertInstalled(t, client, kubeStateMetricsAppName, kubeStateMetricsNamespace)
	})
}
