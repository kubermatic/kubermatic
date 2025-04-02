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

package cniapplicationinstallationcontroller

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
)

const (
	datacenterName = "testdc"
	projectID      = "testproject"

	overrideCNIValue      = "kubeProxyReplacement"   // CNI value that should be always set if proxy mode == ebpf
	appDefDefaultCNIValue = "valueFromAppDefDefault" // CNI value to be present in the default values of the ApplicationDefinition in the tests
	annotationCNIValue    = "valueFromAnnotation"    // CNI value to be used in the initial-cni-values-request annotation by the tests
	existingCNIValue      = "existingValue"          // CNI value to be used by the tests if the CNI ApplicationInstallation already exists
)

var testScheme = fake.NewScheme()

func init() {
	utilruntime.Must(clusterv1alpha1.AddToScheme(testScheme))
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

func genCluster(annotation string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
			Annotations: map[string]string{
				kubermaticv1.InitialCNIValuesRequestAnnotation: annotation,
			},
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenterName,
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCilium),
			},
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Pods:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
				Services:             kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.16.0/20"}},
				NodeCIDRMaskSizeIPv4: ptr.To[int32](24),
				ProxyMode:            resources.EBPFProxyMode,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
		},
	}
}

//gocyclo:ignore
func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name                          string
		cluster                       *kubermaticv1.Cluster
		appDefinitionDefaultValues    map[string]any
		existingAppInstallationValues map[string]any
		validate                      func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error
	}{
		{
			name:                          "no existing ApplicationInstallation, no annotation, no ApplicationDefinition default values",
			cluster:                       genCluster(""),
			appDefinitionDefaultValues:    nil,
			existingAppInstallationValues: nil,
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}
				values, err := getApplicationInstallationValues(userClusterClient)
				if err != nil {
					return err
				}
				if values[overrideCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", overrideCNIValue)
				}
				if values[appDefDefaultCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", appDefDefaultCNIValue, values[appDefDefaultCNIValue])
				}
				if values[annotationCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", annotationCNIValue, values[annotationCNIValue])
				}
				if values[existingCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", existingCNIValue, values[existingCNIValue])
				}
				return nil
			},
		},
		{
			name:                          "no existing ApplicationInstallation, no annotation, existing ApplicationDefinition default values",
			cluster:                       genCluster(""),
			appDefinitionDefaultValues:    map[string]any{appDefDefaultCNIValue: "true"},
			existingAppInstallationValues: nil,
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}
				values, err := getApplicationInstallationValues(userClusterClient)
				if err != nil {
					return err
				}
				if values[overrideCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", overrideCNIValue)
				}
				if values[appDefDefaultCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", appDefDefaultCNIValue)
				}
				if values[annotationCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", annotationCNIValue, values[annotationCNIValue])
				}
				if values[existingCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", existingCNIValue, values[existingCNIValue])
				}
				return nil
			},
		},
		{
			name:                          "no existing ApplicationInstallation, existing annotation, existing ApplicationDefinition default values",
			cluster:                       genCluster(fmt.Sprintf("{\"%s\":true}", annotationCNIValue)),
			appDefinitionDefaultValues:    map[string]any{appDefDefaultCNIValue: "true"},
			existingAppInstallationValues: nil,
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}
				values, err := getApplicationInstallationValues(userClusterClient)
				if err != nil {
					return err
				}
				if values[overrideCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", overrideCNIValue)
				}
				if values[appDefDefaultCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", appDefDefaultCNIValue, values[appDefDefaultCNIValue])
				}
				if values[annotationCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", annotationCNIValue)
				}
				if values[existingCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", existingCNIValue, values[existingCNIValue])
				}
				if ann, ok := cluster.Annotations[kubermaticv1.InitialCNIValuesRequestAnnotation]; ok {
					return fmt.Errorf("annotation should be have been removed, but found %q on the cluster", ann)
				}
				return nil
			},
		},
		{
			name:                          "existing ApplicationInstallation, existing annotation, existing ApplicationDefinition default values",
			cluster:                       genCluster(fmt.Sprintf("{\"%s\":true}", annotationCNIValue)),
			appDefinitionDefaultValues:    map[string]any{appDefDefaultCNIValue: "true"},
			existingAppInstallationValues: map[string]any{existingCNIValue: "true"},
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}
				values, err := getApplicationInstallationValues(userClusterClient)
				if err != nil {
					return err
				}
				if values[overrideCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", overrideCNIValue)
				}
				if values[appDefDefaultCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", appDefDefaultCNIValue, values[appDefDefaultCNIValue])
				}
				if values[annotationCNIValue] != nil {
					return fmt.Errorf("%s CNI value should be nil, value: %s", annotationCNIValue, values[annotationCNIValue])
				}
				if values[existingCNIValue] == nil {
					return fmt.Errorf("CNI value %s is not present", existingCNIValue)
				}
				return nil
			},
		},
		{
			name:                          "invalid annotation",
			cluster:                       genCluster("INVALID"),
			appDefinitionDefaultValues:    map[string]any{appDefDefaultCNIValue: "true"},
			existingAppInstallationValues: nil,
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr == nil {
					return errors.New("reconciling a bad annotation should have produced an error, but got nil")
				}
				return nil
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
			appDef := &appskubermaticv1.ApplicationDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: kubermaticv1.CNIPluginTypeCilium.String(),
				},
			}
			if test.appDefinitionDefaultValues != nil {
				rawValues, err := yaml.Marshal(test.appDefinitionDefaultValues)
				if err != nil {
					t.Fatalf("Test's appDefinitionDefaultValuesBlock marshalling failed: %v", err)
				}
				appDef.Spec.DefaultValuesBlock = string(rawValues)
			}

			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(test.cluster, project, appDef).
				Build()

			userClusterObjects := []ctrlruntimeclient.Object{}
			if test.existingAppInstallationValues != nil {
				appInst := &appskubermaticv1.ApplicationInstallation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kubermaticv1.CNIPluginTypeCilium.String(),
						Namespace: cniPluginNamespace,
					},
				}
				rawValues, err := yaml.Marshal(test.existingAppInstallationValues)
				if err != nil {
					t.Fatalf("Test's existingAppInstallationValues marshalling failed: %v", err)
				}
				appInst.Spec.ValuesBlock = string(rawValues)
				userClusterObjects = append(userClusterObjects, appInst)
			}
			userClusterClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(userClusterObjects...).
				Build()

			ctx := context.Background()
			r := &Reconciler{
				Client:                        seedClient,
				recorder:                      &record.FakeRecorder{},
				log:                           log,
				versions:                      kubermatic.GetFakeVersions(),
				userClusterConnectionProvider: newFakeClientProvider(userClusterClient),
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
			if err := test.validate(newCluster, userClusterClient, reconcileErr); err != nil {
				t.Fatalf("Test failed: %v", err)
			}
		})
	}
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

func getApplicationInstallationValues(userClusterClient ctrlruntimeclient.Client) (map[string]any, error) {
	app := appskubermaticv1.ApplicationInstallation{}
	if err := userClusterClient.Get(context.Background(), types.NamespacedName{Namespace: cniPluginNamespace, Name: kubermaticv1.CNIPluginTypeCilium.String()}, &app); err != nil {
		return nil, fmt.Errorf("failed to get ApplicationInstallation in user cluster: %w", err)
	}
	return app.Spec.GetParsedValues()
}
