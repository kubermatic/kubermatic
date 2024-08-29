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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
	testScheme        = fake.NewScheme()
)

const (
	defaultDatacenterName = "global"
	clusterName           = "cluster1"
	defaultValue          = "not-empty:\n  value"
	projectID             = "testproject"
	applicationName       = "katana"
	appVersion            = "v1.2.0"
)

func init() {
	utilruntime.Must(clusterv1alpha1.AddToScheme(testScheme))
}

func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name         string
		cluster      *kubermaticv1.Cluster
		applications []appskubermaticv1.ApplicationDefinition
		validate     func(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error
	}{
		{
			name:    "scenario 1: no default applications, no application installations",
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", false, false, defaultValue),
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
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
			},
		},
		{
			name:    "scenario 3: multiple default applications are installed with correct values",
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", "", true, false, defaultValue),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.3", "", true, false, ""),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
			},
		},
		{
			name:    "scenario 4: default applications are ignored if initial-application-installation condition exists on the cluster",
			cluster: genCluster(clusterName, defaultDatacenterName, true),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue),
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
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", false, true, ""),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.3", "", false, true, "test: value"),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
			},
		},
		{
			name:    "scenario 5: enforced applications are installed even if initial-application-installation condition exists on the cluster",
			cluster: genCluster(clusterName, defaultDatacenterName, true),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", false, true, ""),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", "", false, true, defaultValue),
				*genApplicationDefinition("applicationName3", "namespace3", appVersion, "", false, true, "test: value"),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
			},
		},
		{
			name:    "scenario 6: enforced and default applications are installed",
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", "", true, true, ""),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.0", "", false, true, "test: value"),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
			},
		},
		{
			name:    "scenario 7: enforced and default applications are installed for a certain datacenter",
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", defaultDatacenterName, true, false, defaultValue),
				*genApplicationDefinition("applicationName2", "namespace2", "v1.0.3", defaultDatacenterName, true, true, ""),
				*genApplicationDefinition("applicationName3", "namespace3", "v1.0.0", defaultDatacenterName, false, true, "test: value"),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
			},
		},
		{
			name:    "scenario 8: enforced and default applications are not installed if cluster doesn't belong to target datacenter",
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "v1.0.0", "wrongdc,invalid", true, false, ""),
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
			cluster: genCluster(clusterName, defaultDatacenterName, false),
			applications: []appskubermaticv1.ApplicationDefinition{
				*genApplicationDefinition("applicationName", "namespace", "", "", true, false, ""),
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
					return errors.New(fmt.Sprintf("installed applications count %d doesn't match the expected couunt %d", len(apps.Items), len(applications)))
				}

				return compareApplications(apps.Items, applications)
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
			userClusterClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build()

			ctx := context.Background()
			r := &Reconciler{
				Client:   seedClient,
				recorder: &record.FakeRecorder{},
				log:      log,
				versions: kubermatic.NewFakeVersions(),

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
			if err := r.Client.Get(ctx, nName, newCluster); err != nil {
				t.Fatalf("Cluster object in seed cluster could not be found anymore: %v", err)
			}

			// validate the result
			if err := test.validate(newCluster, test.applications, userClusterClient, reconcileErr); err != nil {
				t.Fatalf("Test failed: %v", err)
			}
		})
	}
}

func compareApplications(installedApps []appskubermaticv1.ApplicationInstallation, declaredApps []appskubermaticv1.ApplicationDefinition) error {
	// Verify applications by comparing the apps in the cluster with the application definitions
	for _, appDef := range declaredApps {
		found := false
		for _, installedApp := range installedApps {
			if installedApp.Name == appDef.Name && installedApp.Namespace == appDef.Name {
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
				if installedApp.Spec.Namespace.Name != appDef.Name {
					return fmt.Errorf("installed app %s has incorrect namespace: expected %s, got %s", installedApp.Name, appDef.Name, installedApp.Spec.Namespace.Name)
				}

				// Compare labels
				if !reflect.DeepEqual(installedApp.Labels, appDef.Labels) {
					return fmt.Errorf("installed app %s has incorrect labels: expected %v, got %v", installedApp.Name, appDef.Labels, installedApp.Labels)
				}

				// Compare annotations
				delete(appDef.Annotations, appskubermaticv1.ApplicationTargetDatacenterAnnotation)
				if !reflect.DeepEqual(installedApp.Annotations, appDef.Annotations) {
					return fmt.Errorf("installed app %s has incorrect annotations: expected %v, got %v", installedApp.Name, appDef.Annotations, installedApp.Annotations)
				}

				// Compare values
				if installedApp.Spec.ValuesBlock != appDef.Spec.DefaultValuesBlock {
					return fmt.Errorf("installed app %s has incorrect values: expected %v, got %v", installedApp.Name, installedApp.Spec.ValuesBlock, appDef.Spec.DefaultValuesBlock)
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

func genCluster(name, datacenter string, initialApplicationCondition bool) *kubermaticv1.Cluster {
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
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
			Conditions:     conditions,
		},
	}
}

func genApplicationDefinition(name, namespace, defaultVersion, defaultDatacenterName string, defaultApp, enforced bool, defaultValues string) *appskubermaticv1.ApplicationDefinition {
	annotations := map[string]string{}
	if defaultApp {
		annotations[appskubermaticv1.ApplicationDefaultAnnotation] = "true"
	}
	if enforced {
		annotations[appskubermaticv1.ApplicationEnforcedAnnotation] = "true"
	}

	if defaultDatacenterName != "" {
		annotations[appskubermaticv1.ApplicationTargetDatacenterAnnotation] = defaultDatacenterName
	}

	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "Test application definition",
			Method:      appskubermaticv1.HelmTemplateMethod,
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
			DefaultVersion:     defaultVersion,
		},
	}
}

func getSeedObjects(cluster *kubermaticv1.Cluster, applications []appskubermaticv1.ApplicationDefinition) []ctrlruntimeclient.Object {
	objects := []ctrlruntimeclient.Object{}
	objects = append(objects, cluster)
	for _, application := range applications {
		objects = append(objects, &application)
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
