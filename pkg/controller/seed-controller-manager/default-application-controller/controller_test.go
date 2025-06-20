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

package defaultapplicationcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermatictest "k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
	testScheme        = fake.NewScheme()
	noneCNISettings   = kubermaticv1.CNIPluginSettings{
		Type: kubermaticv1.CNIPluginTypeNone,
	}
	ciliumCNISettings = kubermaticv1.CNIPluginSettings{
		Type:    kubermaticv1.CNIPluginTypeCilium,
		Version: "1.15.0",
	}
)

const (
	defaultDatacenterName                   = "global"
	clusterName                             = "cluster1"
	defaultValue                            = "not-empty:\n  value"
	projectID                               = "testproject"
	applicationInstallationNamespace        = "applications"
	changedApplicationInstallationNamespace = "applications-changed"
	applicationName                         = "applicationName"
	appVersion                              = "v1.2.0"
)

func init() {
	utilruntime.Must(clusterv1alpha1.AddToScheme(testScheme))
}

//nolint:gocyclo
func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name                               string
		cluster                            *kubermaticv1.Cluster
		applications                       []appskubermaticv1.ApplicationDefinition
		defaultApplicationNamespace        string
		systemAppInstallationValues        map[string]any
		additionalApplicationInstallations []appskubermaticv1.ApplicationInstallation
		validate                           func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error
	}{
		{
			name:    "scenario 1: no default applications, no application installations",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", false, false, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				// cluster should now have its special condition
				name := kubermaticv1.ClusterConditionDefaultApplicationInstallationControllerReconcilingSuccess

				if cond := cluster.Status.Conditions[name]; cond.Status != corev1.ConditionTrue {
					return fmt.Errorf("cluster should have %v=%s condition, but does not", name, corev1.ConditionTrue)
				}

				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}
				return nil
			},
		},
		{
			name:    "scenario 2: default applications are installed with correct values",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 3: multiple default applications are installed with correct values",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", "", true, false, defaultValue, nil, nil),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.3", "", true, false, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 4: default applications are ignored if initial-application-installation condition exists on the cluster",
			cluster: genCluster(clusterName, defaultDatacenterName, true, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 0 {
					return errors.New("did not expect ApplicationInstallations in the user cluster after the reconciler finished")
				}

				return nil
			},
		},
		{
			name:    "scenario 5: enforced applications are installed",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", false, true, "", nil, nil),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.3", "", false, true, "test: value", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 5: enforced applications are installed even if initial-application-installation condition exists on the cluster",
			cluster: genCluster(clusterName, defaultDatacenterName, true, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", false, true, "", nil, nil),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", "", false, true, defaultValue, nil, nil),
				*genApplicationDefinition("applicationName3", "namespace3", appVersion, "", false, true, "test: value", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 6: enforced and default applications are installed",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", "", true, true, "", nil, nil),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.0", "", false, true, "test: value", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 7: enforced and default applications are installed for a certain datacenter",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", defaultDatacenterName, true, false, defaultValue, nil, nil),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", defaultDatacenterName, true, true, "", nil, nil),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.0", defaultDatacenterName, false, true, "test: value", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 8: enforced and default applications are not installed if cluster doesn't belong to target datacenter",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "wrongdc,invalid", true, false, "", nil, nil),
				*genApplicationDefinition("applicationName2", "namespace", "v1.0.0", "wrongdc,invalid", false, true, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 0 {
					return errors.New("did not expect ApplicationInstallations in the user cluster after the reconciler finished")
				}

				return nil
			},
		},
		{
			name:    "scenario 9: highest semver version is picked as the application version if defaultVersion is not specified",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "", "", true, false, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:    "scenario 10: application values are converted from defaultValues to defaultValuesBlock",
			cluster: genCluster(clusterName, defaultDatacenterName, false, noneCNISettings),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", false, true, "", &runtime.RawExtension{Raw: []byte(`{"test":"value"}`)}, nil),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.3", "", false, true, "", &runtime.RawExtension{Raw: []byte(`{ "commonLabels": {"owner": "somebody"}}`)}, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != len(applications) {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:                        "scenario 11: should create default application in cluster with ready Cilium system application",
			cluster:                     genCluster(clusterName, defaultDatacenterName, false, ciliumCNISettings),
			systemAppInstallationValues: map[string]any{"status": "ready"},
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 2 {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), 2)
				}

				return compareApplications(apps.Items, applications, "", false)
			},
		},
		{
			name:                        "scenario 12: should not create default application in cluster with not ready Cilium system application",
			cluster:                     genCluster(clusterName, defaultDatacenterName, false, ciliumCNISettings),
			systemAppInstallationValues: map[string]any{"status": "not-ready"},
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 1 && apps.Items[0].Name == kubermaticv1.CNIPluginTypeCilium.String() {
					return errors.New("did not expect ApplicationInstallations in the user cluster after the reconciler finished")
				}

				return nil
			},
		},
		{
			name:                        "scenario 13: should create default application in cluster with ready Cilium system application in configured default applicationinstallation namespace",
			defaultApplicationNamespace: applicationInstallationNamespace,
			cluster:                     genCluster(clusterName, defaultDatacenterName, false, ciliumCNISettings),
			systemAppInstallationValues: map[string]any{"status": "ready"},
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, applicationInstallationNamespace, "v1.0.0", "", true, false, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 2 {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), 2)
				}

				return compareApplications(apps.Items, applications, applicationInstallationNamespace, false)
			},
		},
		{
			name:                        "scenario 14: should not update default application in cluster with ready Cilium system application when the default applicationinstallation namespace has changed",
			defaultApplicationNamespace: changedApplicationInstallationNamespace,
			cluster:                     genCluster(clusterName, defaultDatacenterName, false, ciliumCNISettings),
			systemAppInstallationValues: map[string]any{"status": "ready"},
			additionalApplicationInstallations: []appskubermaticv1.ApplicationInstallation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        applicationName,
						Namespace:   applicationInstallationNamespace,
						Annotations: map[string]string{appskubermaticv1.ApplicationEnforcedAnnotation: "true", appskubermaticv1.ApplicationDefaultedAnnotation: "true"},
					},
					Spec: appskubermaticv1.ApplicationInstallationSpec{
						Namespace: &appskubermaticv1.AppNamespaceSpec{
							Name: "application",
						},
						ApplicationRef: appskubermaticv1.ApplicationRef{
							Name: applicationName,
						},
					},
					Status: appskubermaticv1.ApplicationInstallationStatus{
						ApplicationVersion: &appskubermaticv1.ApplicationVersion{
							Version: "v1.0.0",
						},
					},
				},
			},
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, applicationInstallationNamespace, "v1.0.0", "", true, true, defaultValue, nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 2 {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, changedApplicationInstallationNamespace, false)
			},
		},
		{
			name:                        "scenario 15: enforced annotation should be set to false when enforcing was disabled in the related application definition",
			defaultApplicationNamespace: changedApplicationInstallationNamespace,
			cluster:                     genCluster(clusterName, defaultDatacenterName, false, ciliumCNISettings),
			systemAppInstallationValues: map[string]any{"status": "ready"},
			additionalApplicationInstallations: []appskubermaticv1.ApplicationInstallation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        applicationName,
						Namespace:   applicationInstallationNamespace,
						Annotations: map[string]string{appskubermaticv1.ApplicationEnforcedAnnotation: "true"},
					},
					Spec: appskubermaticv1.ApplicationInstallationSpec{
						Namespace: &appskubermaticv1.AppNamespaceSpec{
							Name: applicationName,
						},
						ApplicationRef: appskubermaticv1.ApplicationRef{
							Name:    applicationName,
							Version: "v1.0.0",
						},
					},
				},
			},
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition(applicationName, applicationInstallationNamespace, "v1.0.0", "", false, false, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 2 {
					return fmt.Errorf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications))
				}

				return compareApplications(apps.Items, applications, changedApplicationInstallationNamespace, true)
			},
		},
	}
	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectID,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			objects := getSeedObjects(test.cluster, test.applications)
			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(objects...).
				WithObjects(project).
				Build()
			userClusterObjects := getUserClusterObjects(t, test.systemAppInstallationValues, test.additionalApplicationInstallations)
			userClusterClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(userClusterObjects...).
				Build()

			ctx := context.Background()
			config := createKubermaticConfiguration(test.defaultApplicationNamespace)

			r := &Reconciler{
				Client:       seedClient,
				recorder:     &record.FakeRecorder{},
				log:          log,
				versions:     kubermatic.GetFakeVersions(),
				configGetter: kubermatictest.NewConfigGetter(config),

				userClusterConnectionProvider: newFakeClientProvider(userClusterClient),

				// this dummy seedGetter returns the same dummy hetzner DC for all tests
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{
						Spec: kubermaticv1.SeedSpec{
							Datacenters: map[string]kubermaticv1.Datacenter{
								defaultDatacenterName: {
									Spec: kubermaticv1.DatacenterSpec{
										Hetzner: &kubermaticv1.DatacenterSpecHetzner{
											Datacenter: "hel1",
											Network:    "default",
										},
									},
								},
								"datacenter2": {
									Spec: kubermaticv1.DatacenterSpec{
										Hetzner: &kubermaticv1.DatacenterSpecHetzner{
											Datacenter: "hel1",
											Network:    "default",
										},
									},
								},
								"datacenter3": {
									Spec: kubermaticv1.DatacenterSpec{
										Hetzner: &kubermaticv1.DatacenterSpecHetzner{
											Datacenter: "hel1",
											Network:    "default",
										},
									},
								},
							},
						},
					}, nil
				},
			}

			nName := types.NamespacedName{Name: test.cluster.Name}

			// let the magic happen
			_, reconcileErr := r.Reconcile(ctx, reconcile.Request{NamespacedName: nName})

			// fetch potentially updated cluster object
			newCluster := &kubermaticv1.Cluster{}
			if err := r.Get(ctx, nName, newCluster); err != nil {
				t.Fatalf("Cluster object in seed cluster could not be found anymore: %v", err)
			}

			// validate the result
			if err := test.validate(newCluster, test.applications, userClusterClient, reconcileErr); err != nil {
				t.Fatalf("Test failed: %v", err)
			}
		})
	}
}
func getUserClusterObjects(t *testing.T, systemAppInstallationValues map[string]any, existingApplications []appskubermaticv1.ApplicationInstallation) []ctrlruntimeclient.Object {
	userClusterObjects := []ctrlruntimeclient.Object{}
	if systemAppInstallationValues != nil {
		appInst := &appskubermaticv1.ApplicationInstallation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubermaticv1.CNIPluginTypeCilium.String(),
				Namespace: metav1.NamespaceSystem,
			},
		}
		if systemAppInstallationValues["status"] == "ready" {
			appInst.Status = appskubermaticv1.ApplicationInstallationStatus{
				ApplicationVersion: &appskubermaticv1.ApplicationVersion{
					Version: "1.15.0",
				},
			}
		}
		rawValues, err := json.Marshal(systemAppInstallationValues)
		if err != nil {
			t.Fatalf("Test's systemAppInstallationValues marshalling failed: %v", err)
		}
		appInst.Spec.Values = runtime.RawExtension{Raw: rawValues}
		userClusterObjects = append(userClusterObjects, appInst)
	}

	for _, application := range existingApplications {
		userClusterObjects = append(userClusterObjects, &application)
	}
	return userClusterObjects
}

func compareApplications(installedApps []appskubermaticv1.ApplicationInstallation, declaredApps []appskubermaticv1.ApplicationDefinition, defaultAppNamespace string, annotationsHasChanged bool) error {
	// Verify applications by comparing the apps in the cluster with the application definitions
	for _, appDef := range declaredApps {
		found := false
		for _, installedApp := range installedApps {
			if (installedApp.Name == appDef.Name && defaultAppNamespace == "" && installedApp.Namespace == appDef.Name) ||
				(installedApp.Name == appDef.Name && defaultAppNamespace != "" && installedApp.Namespace == appDef.Namespace) {
				found = true
				// Check if the installed app matches the definition
				if installedApp.Spec.ApplicationRef.Name != appDef.Name {
					return fmt.Errorf("installed app %s has incorrect ApplicationRef.Name: expected %s, got %s", installedApp.Name, appDef.Name, installedApp.Spec.ApplicationRef.Name)
				}

				if appDef.Spec.DefaultVersion != "" {
					if installedApp.Spec.ApplicationRef.Version != appDef.Spec.DefaultVersion {
						return fmt.Errorf("installed app %s has incorrect version: expected %s, got %s", installedApp.Name, appDef.Spec.DefaultVersion, installedApp.Spec.ApplicationRef.Version)
					}
				} else {
					if installedApp.Spec.ApplicationRef.Version != appVersion {
						return fmt.Errorf("installed app %s has incorrect version: expected %s, got %s", installedApp.Name, appVersion, installedApp.Spec.ApplicationRef.Version)
					}
				}

				// Compare namespace
				// when there is no default namespace configured in the related appdef, then the appdef name will be used for the applicationinstallation namespace
				if appDef.Spec.DefaultNamespace == nil && installedApp.Spec.Namespace.Name != appDef.Name {
					return fmt.Errorf("installed app %s has incorrect namespace: expected %s, got %s", installedApp.Name, appDef.Name, installedApp.Spec.Namespace.Name)
				}
				// when there is a default namespace configured in the related appdef, then this namespae will be used for the applicationinstallation namespace
				if appDef.Spec.DefaultNamespace != nil && installedApp.Spec.Namespace.Name != appDef.Spec.DefaultNamespace.Name {
					return fmt.Errorf("installed app %s has incorrect namespace: expected %s, got %s", installedApp.Name, appDef.Name, installedApp.Spec.Namespace.Name)
				}

				// Compare labels
				if !reflect.DeepEqual(installedApp.Labels, appDef.Labels) {
					return fmt.Errorf("installed app %s has incorrect labels: expected %v, got %v", installedApp.Name, appDef.Labels, installedApp.Labels)
				}

				// Compare annotations
				expectedAnnotations := map[string]string{}
				if appDef.Spec.Default {
					expectedAnnotations[appskubermaticv1.ApplicationDefaultedAnnotation] = "true"
				}
				if appDef.Spec.Enforced {
					expectedAnnotations[appskubermaticv1.ApplicationEnforcedAnnotation] = "true"
				} else if annotationsHasChanged {
					expectedAnnotations[appskubermaticv1.ApplicationEnforcedAnnotation] = "false"
				}

				if !reflect.DeepEqual(installedApp.Annotations, expectedAnnotations) {
					return fmt.Errorf("installed app %s has incorrect annotations: expected %v, got %v", installedApp.Name, appDef.Annotations, installedApp.Annotations)
				}

				// Compare values
				if installedApp.Spec.ValuesBlock != appDef.Spec.DefaultValuesBlock {
					// Values have been converted from defaultValues to defaultValuesBlock
					if appDef.Spec.DefaultValues != nil {
						_ = convertDefaultValuesToDefaultValuesBlock(&appDef)
						if installedApp.Spec.ValuesBlock != appDef.Spec.DefaultValuesBlock {
							return fmt.Errorf("installed app %s has incorrect values: expected %q, got %q", installedApp.Name, appDef.Spec.DefaultValuesBlock, installedApp.Spec.ValuesBlock)
						}
					} else {
						return fmt.Errorf("installed app %s has incorrect values: expected %q, got %q", installedApp.Name, appDef.Spec.DefaultValuesBlock, installedApp.Spec.ValuesBlock)
					}
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("application %s not found in installed applications", appDef.Name)
		}
	}
	return nil
}

func healthy() kubermaticv1.ExtendedClusterHealth {
	return kubermaticv1.ExtendedClusterHealth{
		Apiserver:                    kubermaticv1.HealthStatusUp,
		ApplicationController:        kubermaticv1.HealthStatusUp,
		Scheduler:                    kubermaticv1.HealthStatusUp,
		Controller:                   kubermaticv1.HealthStatusUp,
		MachineController:            kubermaticv1.HealthStatusUp,
		Etcd:                         kubermaticv1.HealthStatusUp,
		OpenVPN:                      kubermaticv1.HealthStatusUp,
		CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
		UserClusterControllerManager: kubermaticv1.HealthStatusUp,
	}
}

func genCluster(name, datacenter string, initialApplicationCondition bool, cniPluginSettings kubermaticv1.CNIPluginSettings) *kubermaticv1.Cluster {
	conditions := map[kubermaticv1.ClusterConditionType]kubermaticv1.ClusterCondition{}
	if initialApplicationCondition {
		conditions[kubermaticv1.ClusterConditionApplicationInstallationControllerReconcilingSuccess] = kubermaticv1.ClusterCondition{
			Status: corev1.ConditionTrue,
		}
	}
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenter,
			},
			CNIPlugin: &cniPluginSettings,
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
			Conditions:     conditions,
		},
	}
}

func genApplicationDefinition(name, namespace, defaultVersion, defaultDatacenterName string, defaultApp, enforced bool, defaultValues string, defaultRawValues *runtime.RawExtension, defaultNamespace *appskubermaticv1.AppNamespaceSpec) *appskubermaticv1.ApplicationDefinition {
	annotations := map[string]string{}
	selector := appskubermaticv1.DefaultingSelector{}
	if defaultDatacenterName != "" {
		selector.Datacenters = []string{defaultDatacenterName}
	}

	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description:      "Test application definition",
			Method:           appskubermaticv1.HelmTemplateMethod,
			DefaultNamespace: defaultNamespace,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "v1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com/test-chart",
								ChartName:    name,
								ChartVersion: "1.0.0",
							},
						},
					},
				},
				{
					Version: appVersion,
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com/test-chart",
								ChartName:    name,
								ChartVersion: appVersion,
							},
						},
					},
				},
				{
					Version: "v1.0.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com/test-chart",
								ChartName:    name,
								ChartVersion: "1.0.3",
							},
						},
					},
				},
			},
			DefaultValuesBlock: defaultValues,
			DefaultValues:      defaultRawValues,
			DefaultVersion:     defaultVersion,
			Default:            defaultApp,
			Enforced:           enforced,
			Selector:           selector,
		},
	}
}

func getSeedObjects(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition) []ctrlruntimeclient.Object {
	objects := []ctrlruntimeclient.Object{}
	objects = append(objects, cluster)
	for _, application := range applications {
		applicationCopy := application
		objects = append(objects, &applicationCopy)
	}
	return objects
}

type fakeClientProvider struct {
	client ctrlruntimeclient.Client
}

func newFakeClientProvider(client ctrlruntimeclient.Client) *fakeClientProvider {
	return &fakeClientProvider{
		client: client,
	}
}

func (f *fakeClientProvider) GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}

func createKubermaticConfiguration(defaultAppNamespace string) *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				Applications: kubermaticv1.ApplicationsConfiguration{
					Namespace: defaultAppNamespace,
				},
			},
		},
	}
}
