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

func TestUserClusterMonitoringReconcile(t *testing.T) {
	const (
		clusterName  = "test-cluster"
		neVersion    = "1.10.2"
		ksmVersion   = "2.18.0"
		neDefValues  = "tolerations:\n- effect: NoExecute\n  operator: Exists\n"
		ksmDefValues = "resources:\n  limits:\n    cpu: 100m\n"
		customValues = "customKey: customValue\n"
	)

	testCases := []struct {
		name               string
		cluster            *kubermaticv1.Cluster
		seedObjects        []ctrlruntimeclient.Object
		userClusterObjects []ctrlruntimeclient.Object
		validate           func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error)
	}{
		{
			name:    "monitoring disabled: no applications installed",
			cluster: makeHealthyCluster(clusterName, false),
			seedObjects: []ctrlruntimeclient.Object{
				makeAppDef(nodeExporterAppName, neVersion, neDefValues),
				makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}
				list := &appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), list); err != nil {
					t.Fatalf("failed to list ApplicationInstallations: %v", err)
				}
				if len(list.Items) != 0 {
					t.Errorf("expected 0 ApplicationInstallations, got %d", len(list.Items))
				}
			},
		},
		{
			name:    "monitoring enabled, node-exporter AppDef missing: nothing installed",
			cluster: makeHealthyCluster(clusterName, true),
			seedObjects: []ctrlruntimeclient.Object{
				makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}
				list := &appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), list); err != nil {
					t.Fatalf("failed to list ApplicationInstallations: %v", err)
				}
				if len(list.Items) != 0 {
					t.Errorf("expected 0 ApplicationInstallations, got %d", len(list.Items))
				}
			},
		},
		{
			name:    "monitoring enabled, kube-state-metrics AppDef missing: nothing installed",
			cluster: makeHealthyCluster(clusterName, true),
			seedObjects: []ctrlruntimeclient.Object{
				makeAppDef(nodeExporterAppName, neVersion, neDefValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}
				list := &appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), list); err != nil {
					t.Fatalf("failed to list ApplicationInstallations: %v", err)
				}
				if len(list.Items) != 0 {
					t.Errorf("expected 0 ApplicationInstallations, got %d", len(list.Items))
				}
			},
		},
		{
			name:    "monitoring enabled: both applications installed with defaultValuesBlock",
			cluster: makeHealthyCluster(clusterName, true),
			seedObjects: []ctrlruntimeclient.Object{
				makeAppDef(nodeExporterAppName, neVersion, neDefValues),
				makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}

				neApp := &appskubermaticv1.ApplicationInstallation{}
				if err := userClusterClient.Get(context.Background(), types.NamespacedName{Name: nodeExporterAppName, Namespace: nodeExporterNamespace}, neApp); err != nil {
					t.Fatalf("node-exporter ApplicationInstallation not found: %v", err)
				}
				if neApp.Spec.ApplicationRef.Version != neVersion {
					t.Errorf("node-exporter version: got %q, want %q", neApp.Spec.ApplicationRef.Version, neVersion)
				}
				if neApp.Labels[appskubermaticv1.ApplicationManagedByLabel] != appskubermaticv1.ApplicationManagedByKKPValue {
					t.Errorf("node-exporter missing managed-by label")
				}
				// Values should be seeded from defaultValuesBlock on first install.
				neValues, err := neApp.Spec.GetParsedValues()
				if err != nil {
					t.Fatalf("failed to parse node-exporter values: %v", err)
				}
				if len(neValues) == 0 {
					t.Errorf("node-exporter ValuesBlock should be seeded from defaultValuesBlock, but is empty")
				}

				ksmApp := &appskubermaticv1.ApplicationInstallation{}
				if err := userClusterClient.Get(context.Background(), types.NamespacedName{Name: kubeStateMetricsAppName, Namespace: kubeStateMetricsNamespace}, ksmApp); err != nil {
					t.Fatalf("kube-state-metrics ApplicationInstallation not found: %v", err)
				}
				if ksmApp.Spec.ApplicationRef.Version != ksmVersion {
					t.Errorf("kube-state-metrics version: got %q, want %q", ksmApp.Spec.ApplicationRef.Version, ksmVersion)
				}
				if ksmApp.Labels[appskubermaticv1.ApplicationManagedByLabel] != appskubermaticv1.ApplicationManagedByKKPValue {
					t.Errorf("kube-state-metrics missing managed-by label")
				}
				ksmValues, err := ksmApp.Spec.GetParsedValues()
				if err != nil {
					t.Fatalf("failed to parse kube-state-metrics values: %v", err)
				}
				if len(ksmValues) == 0 {
					t.Errorf("kube-state-metrics ValuesBlock should be seeded from defaultValuesBlock, but is empty")
				}
			},
		},
		{
			name:    "monitoring enabled: existing installation preserves user-customized values",
			cluster: makeHealthyCluster(clusterName, true),
			seedObjects: []ctrlruntimeclient.Object{
				makeAppDef(nodeExporterAppName, neVersion, neDefValues),
				makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues),
			},
			userClusterObjects: []ctrlruntimeclient.Object{
				makeExistingAppInstallation(nodeExporterAppName, nodeExporterNamespace, customValues),
				makeExistingAppInstallation(kubeStateMetricsAppName, kubeStateMetricsNamespace, customValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}

				neApp := &appskubermaticv1.ApplicationInstallation{}
				if err := userClusterClient.Get(context.Background(), types.NamespacedName{Name: nodeExporterAppName, Namespace: nodeExporterNamespace}, neApp); err != nil {
					t.Fatalf("node-exporter ApplicationInstallation not found: %v", err)
				}
				neValues, err := neApp.Spec.GetParsedValues()
				if err != nil {
					t.Fatalf("failed to parse node-exporter values: %v", err)
				}
				if _, ok := neValues["customKey"]; !ok {
					t.Errorf("node-exporter: user-customized values should be preserved, but 'customKey' is missing")
				}

				ksmApp := &appskubermaticv1.ApplicationInstallation{}
				if err := userClusterClient.Get(context.Background(), types.NamespacedName{Name: kubeStateMetricsAppName, Namespace: kubeStateMetricsNamespace}, ksmApp); err != nil {
					t.Fatalf("kube-state-metrics ApplicationInstallation not found: %v", err)
				}
				ksmValues, err := ksmApp.Spec.GetParsedValues()
				if err != nil {
					t.Fatalf("failed to parse kube-state-metrics values: %v", err)
				}
				if _, ok := ksmValues["customKey"]; !ok {
					t.Errorf("kube-state-metrics: user-customized values should be preserved, but 'customKey' is missing")
				}
			},
		},
		{
			name:    "monitoring disabled: existing KKP-managed installations are removed",
			cluster: makeHealthyCluster(clusterName, false),
			userClusterObjects: []ctrlruntimeclient.Object{
				makeExistingAppInstallation(nodeExporterAppName, nodeExporterNamespace, neDefValues),
				makeExistingAppInstallation(kubeStateMetricsAppName, kubeStateMetricsNamespace, ksmDefValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}
				list := &appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), list); err != nil {
					t.Fatalf("failed to list ApplicationInstallations: %v", err)
				}
				if len(list.Items) != 0 {
					t.Errorf("expected 0 ApplicationInstallations after disable, got %d", len(list.Items))
				}
			},
		},
		{
			name:    "monitoring disabled: non-KKP-managed installation is left alone",
			cluster: makeHealthyCluster(clusterName, false),
			userClusterObjects: []ctrlruntimeclient.Object{
				// no managed-by label — user-owned install
				&appskubermaticv1.ApplicationInstallation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            nodeExporterAppName,
						Namespace:       nodeExporterNamespace,
						ResourceVersion: "1",
					},
				},
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}
				list := &appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), list); err != nil {
					t.Fatalf("failed to list ApplicationInstallations: %v", err)
				}
				if len(list.Items) != 1 {
					t.Errorf("expected user-owned ApplicationInstallation to remain, got %d items", len(list.Items))
				}
			},
		},
		{
			name:    "monitoring enabled: AppDef with defaultVersion uses that version",
			cluster: makeHealthyCluster(clusterName, true),
			seedObjects: []ctrlruntimeclient.Object{
				func() *appskubermaticv1.ApplicationDefinition {
					ad := makeAppDef(nodeExporterAppName, "1.0.0", neDefValues)
					ad.Spec.DefaultVersion = "1.99.0"
					ad.Spec.Versions = append(ad.Spec.Versions, appskubermaticv1.ApplicationVersion{Version: "1.99.0"})
					return ad
				}(),
				makeAppDef(kubeStateMetricsAppName, ksmVersion, ksmDefValues),
			},
			validate: func(t *testing.T, userClusterClient ctrlruntimeclient.Client, reconcileErr error) {
				t.Helper()
				if reconcileErr != nil {
					t.Fatalf("unexpected error: %v", reconcileErr)
				}
				neApp := &appskubermaticv1.ApplicationInstallation{}
				if err := userClusterClient.Get(context.Background(), types.NamespacedName{Name: nodeExporterAppName, Namespace: nodeExporterNamespace}, neApp); err != nil {
					t.Fatalf("node-exporter ApplicationInstallation not found: %v", err)
				}
				if neApp.Spec.ApplicationRef.Version != "1.99.0" {
					t.Errorf("expected defaultVersion 1.99.0, got %q", neApp.Spec.ApplicationRef.Version)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seedObjects := append(tc.seedObjects, tc.cluster)
			reconciler := newTestUserClusterMonitoringReconciler(seedObjects, tc.userClusterObjects)

			_, reconcileErr := reconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: tc.cluster.Name},
			})

			userClusterClient := reconciler.userClusterConnectionProvider.(*fakeUserClusterClientProvider).client
			tc.validate(t, userClusterClient, reconcileErr)
		})
	}
}
