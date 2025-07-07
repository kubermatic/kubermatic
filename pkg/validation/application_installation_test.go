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

package validation

import (
	"context"
	"fmt"
	"testing"
	"time"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAppName             = "app"
	defaultAppVersion          = "1.2.3"
	defaultAppSecondaryVersion = "1.2.4"
	invalidResource            = "invalid"
)

// TestValidateApplicationInstallationSpec tests the validation for ApplicationInstallation creation.
func TestValidateApplicationInstallationSpec(t *testing.T) {
	ad := getApplicationDefinition(defaultAppName, false, false, nil, nil)
	fakeClient := fake.
		NewClientBuilder().
		WithObjects(ad).
		Build()

	ai := getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, nil)

	testCases := []struct {
		name          string
		ai            *appskubermaticv1.ApplicationInstallation
		expectedError string
	}{
		{
			name:          "Create ApplicationInstallation Success",
			ai:            ai,
			expectedError: "[]",
		},
		{
			name: "Create ApplicationInstallation Success - DeployOpts helm is nil",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.DeployOptions = &appskubermaticv1.DeployOptions{}
					return *spec
				}(),
			}, expectedError: "[]",
		},
		{
			name: "Create ApplicationInstallation Success - DeployOpts helm={wait: true, timeout: 5, atomic: true}",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    true,
						Timeout: metav1.Duration{Duration: 5},
						Atomic:  true,
					}}
					return *spec
				}(),
			}, expectedError: "[]",
		},
		{
			name: "Create ApplicationInstallation Success - DeployOpts helm ={wait: true, timeout: 5, atomic: false}",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    true,
						Timeout: metav1.Duration{Duration: 5},
						Atomic:  false,
					}}
					return *spec
				}(),
			}, expectedError: "[]",
		},
		{
			name: "Create ApplicationInstallation Failure - DeployOpts helm (atomic=true but wait=false)",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    false,
						Timeout: metav1.Duration{Duration: 0},
						Atomic:  true,
					}}
					return *spec
				}(),
			}, expectedError: "[spec.deployOptions.helm: Forbidden: if atomic=true then wait must also be true]",
		},
		{
			name: "Create ApplicationInstallation Failure - DeployOpts helm (wait=true but timeout=0)",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    true,
						Timeout: metav1.Duration{Duration: 0},
						Atomic:  true,
					}}
					return *spec
				}(),
			}, expectedError: "[spec.deployOptions.helm: Forbidden: if wait = true then timeout must be greater than 0]",
		},
		{
			name: "Create ApplicationInstallation Failure - DeployOpts helm (wait false but timeout defined)",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    false,
						Timeout: metav1.Duration{Duration: 5},
						Atomic:  false,
					}}
					return *spec
				}(),
			}, expectedError: "[spec.deployOptions.helm: Forbidden: if timeout is defined then wait must be true]",
		},
		{
			name: "Create ApplicationInstallation Failure - ApplicationDefinitation doesn't exist",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ApplicationRef.Name = invalidResource
					return *spec
				}(),
			}, expectedError: `[spec.applicationRef.name: Not found: "invalid"]`,
		},
		{
			name: "Create ApplicationInstallation Failure - Invalid Version",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ApplicationRef.Version = "3.2.3"
					return *spec
				}(),
			}, expectedError: `[spec.applicationRef.version: Not found: "3.2.3"]`,
		},
		{
			name: "Create ApplicationInstallation Success - ReconciliationInterval equals 0",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ReconciliationInterval = metav1.Duration{Duration: 0}
					return *spec
				}(),
			}, expectedError: `[]`,
		},
		{
			name: "Create ApplicationInstallation Success - ReconciliationInterval greater than 0",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ReconciliationInterval = metav1.Duration{Duration: 10 * time.Minute}
					return *spec
				}(),
			}, expectedError: `[]`,
		},
		{
			name: "Create ApplicationInstallation Failure - Invalid ReconciliationInterval less than 0",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ReconciliationInterval = metav1.Duration{Duration: -10}
					return *spec
				}(),
			}, expectedError: `[spec.reconciliationInterval: Invalid value: "-10ns": should be a positive value, or zero to disable]`,
		},
		{
			name: "Create ApplicationInstallation Failure - Both values and valuesBlock are set",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Values = runtime.RawExtension{Raw: []byte("key: value")}
					spec.ValuesBlock = "key: value"
					return *spec
				}(),
			}, expectedError: `[spec.values: Forbidden: Only values or valuesBlock can be set, but not both simultaneously spec.valuesBlock: Forbidden: Only values or valuesBlock can be set, but not both simultaneously]`,
		},
		{
			name: "Create ApplicationInstallation Success - ValuesBlock set and Values in default empty",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Values = runtime.RawExtension{Raw: []byte("{}")} // Raw.Runtime Extension gets defaulted to '{}' when set through k8s-api
					spec.ValuesBlock = "key: value"
					return *spec
				}(),
			}, expectedError: `[]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateApplicationInstallationSpec(context.Background(), fakeClient, *testCase.ai)
			if fmt.Sprint(err) != testCase.expectedError {
				if testCase.expectedError == "[]" {
					testCase.expectedError = "nil"
				}
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, err)
			}
		})
	}
}

// TestValidateApplicationInstallationSpec tests the validation for ApplicationInstallation creation.
func TestValidateApplicationInstallationUpdate(t *testing.T) {
	ad := getApplicationDefinition(defaultAppName, false, false, nil, nil)
	updatedAD := getApplicationDefinition("updated-app", false, false, nil, nil)
	fakeClient := fake.
		NewClientBuilder().
		WithObjects(ad, updatedAD).
		Build()

	ai := getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, nil)

	aiVersionDoesExist := getApplicationInstallation(defaultAppName, defaultAppName, "0.0.0-does-not-exist", nil)
	testCases := []struct {
		name          string
		ai            *appskubermaticv1.ApplicationInstallation
		updatedAI     *appskubermaticv1.ApplicationInstallation
		expectedError string
	}{
		{
			name: "Update ApplicationInstallation Success",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Labels = map[string]string{"key": "value"}
					spec.ApplicationRef.Version = defaultAppSecondaryVersion
					return *spec
				}(),
			},
			expectedError: "[]",
		},
		{
			name: "Update deleting ApplicationInstallation Success (app def version does not exist)",
			ai:   aiVersionDoesExist,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: *aiVersionDoesExist.Spec.DeepCopy(),
			},
			expectedError: "[]",
		},
		{
			name: "Update ApplicationInstallation Failure - .Namespace.Name is immutable",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Name = invalidResource
					return *spec
				}(),
			},
			expectedError: `[spec.namespace.name: Invalid value: "invalid": field is immutable]`,
		},
		{
			name: "Update ApplicationInstallation Failure - .ApplicationRef.Name is immutable",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ApplicationRef.Name = "updated-app"
					return *spec
				}(),
			},
			expectedError: `[spec.applicationRef.name: Invalid value: "updated-app": field is immutable]`,
		},
		{
			name: "Update ApplicationInstallation Success - ReconciliationInterval equals 0",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ReconciliationInterval = metav1.Duration{Duration: 0}
					return *spec
				}(),
			},
			expectedError: "[]",
		},
		{
			name: "Update ApplicationInstallation Success - ReconciliationInterval greater than 0",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ReconciliationInterval = metav1.Duration{Duration: 10 * time.Minute}
					return *spec
				}(),
			},
			expectedError: "[]",
		},
		{
			name: "Update ApplicationInstallation Failure - Invalid ReconciliationInterval less than 0",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ReconciliationInterval = metav1.Duration{Duration: -10}
					return *spec
				}(),
			},
			expectedError: `[spec.reconciliationInterval: Invalid value: "-10ns": should be a positive value, or zero to disable]`,
		},
		{
			name: "Update ApplicationInstallation Failure - managed-by label is immutable",
			ai: getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
			}),
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						appskubermaticv1.ApplicationManagedByLabel: "somebody-else",
					},
				},
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Labels = map[string]string{"key": "value"}
					spec.ApplicationRef.Version = defaultAppSecondaryVersion
					return *spec
				}(),
			},
			expectedError: `[metadata.labels: Invalid value: map[string]string{"apps.kubermatic.k8c.io/managed-by":"somebody-else"}: label "apps.kubermatic.k8c.io/managed-by" is immutable]`, // TODO: change message
		},
		{
			name: "Update ApplicationInstallation Failure - type label is immutable if managed by kkp",
			ai: getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}),
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
						appskubermaticv1.ApplicationTypeLabel:      "something-else",
					},
				},
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Labels = map[string]string{"key": "value"}
					spec.ApplicationRef.Version = defaultAppSecondaryVersion
					return *spec
				}(),
			},
			expectedError: `[metadata.labels: Invalid value: map[string]string{"apps.kubermatic.k8c.io/managed-by":"kkp", "apps.kubermatic.k8c.io/type":"something-else"}: label "apps.kubermatic.k8c.io/type" is immutable]`,
		},
		{
			name: "Update ApplicationInstallation Failure - invalid values",
			ai: getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}),
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
						appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
					},
				},
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Labels = map[string]string{"key": "value"}
					spec.ApplicationRef.Version = defaultAppSecondaryVersion
					spec.Values = runtime.RawExtension{Raw: []byte("INVALID")}
					return *spec
				}(),
			},
			expectedError: `[spec.values: Invalid value: "INVALID": unable to unmarshal values: invalid character 'I' looking for beginning of value]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateApplicationInstallationUpdate(context.Background(), fakeClient, *testCase.updatedAI, *testCase.ai)
			if fmt.Sprint(err) != testCase.expectedError {
				if testCase.expectedError == "[]" {
					testCase.expectedError = "nil"
				}
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, err)
			}
		})
	}
}

func TestValidateApplicationInstallationDelete(t *testing.T) {
	enforcedAppName := "enforced"
	enforcedAppDCName := "enforced-dc"
	enforcedWrongDCName := "enforced-wrong-dc"
	defaultAppName := "default"
	appName := "app"
	dcName := "dc-1"
	clusterName := "cluster-1"

	applicationDefinitions := []ctrlruntimeclient.Object{
		getApplicationDefinition(appName, false, false, nil, nil),
		getApplicationDefinition(defaultAppName, true, false, nil, nil),
		getApplicationDefinition(enforcedAppName, false, true, nil, nil),
		getApplicationDefinition(enforcedAppDCName, false, true, []string{dcName}, nil),
		getApplicationDefinition(enforcedWrongDCName, false, true, []string{"dc-2"}, nil),
	}

	fakeClient := fake.
		NewClientBuilder().
		WithObjects(genCluster(clusterName, dcName), genCluster("cluster-2", "dc-2")).
		WithObjects(applicationDefinitions...).
		Build()

	testCases := []struct {
		name          string
		ai            *appskubermaticv1.ApplicationInstallation
		clusterName   string
		expectedError string
	}{
		{
			name:          "scenario 1: application deletion is allowed for non-default/non-enforced application installation",
			ai:            getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, nil),
			expectedError: "[]",
		},
		{
			name:          "scenario 2: application deletion is allowed if referenced application definition is not found",
			ai:            getApplicationInstallation(defaultAppName, "non-existent-app", defaultAppVersion, nil),
			expectedError: "[]",
		},
		{
			name:          "scenario 3: application deletion is allowed if referenced application definition is default application",
			ai:            getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion, nil),
			expectedError: "[]",
		},
		{
			name:          "scenario 4: application deletion is not allowed if referenced application definition is enforced application",
			ai:            getApplicationInstallation(enforcedAppName, enforcedAppName, defaultAppVersion, nil),
			expectedError: `[spec.applicationRef.name: Forbidden: application "enforced" is enforced and cannot be deleted. Please contact your administrator.]`,
		},
		{
			name:          "scenario 5: application deletion is not allowed if referenced application definition is enforced for the datacenter",
			ai:            getApplicationInstallation(enforcedAppDCName, enforcedAppDCName, defaultAppVersion, nil),
			expectedError: `[spec.applicationRef.name: Forbidden: application "enforced-dc" is enforced and cannot be deleted. Please contact your administrator.]`,
			clusterName:   clusterName,
		},
		{
			name:          "scenario 6: application deletion is allowed if referenced application definition is not enforced for the datacenter",
			ai:            getApplicationInstallation(enforcedWrongDCName, enforcedWrongDCName, defaultAppVersion, nil),
			expectedError: `[]`,
			clusterName:   clusterName,
		},
		{
			name:          "scenario 7: application deletion is allowed if the installed application name/namespace is different than the application definition name, even if application definition is marked as enforced",
			ai:            getApplicationInstallation(defaultAppName, enforcedAppName, defaultAppVersion, nil),
			expectedError: `[]`,
			clusterName:   clusterName,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateApplicationInstallationDelete(context.Background(), fakeClient, testCase.clusterName, *testCase.ai)
			if fmt.Sprint(err) != testCase.expectedError {
				if testCase.expectedError == "[]" {
					testCase.expectedError = "nil"
				}
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, err)
			}
		})
	}
}

func getApplicationDefinition(name string, defaulted, enforced bool, datacenters []string, labels map[string]string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationDefinition",
			APIVersion: "apps.kubermatic.k8c.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "Description",
			Method:      appskubermaticv1.HelmTemplateMethod,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: defaultAppVersion,
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "http://example.com/charts",
								ChartName:    "test-chart",
								ChartVersion: defaultAppVersion,
							},
						},
					},
				},
				{
					Version: defaultAppSecondaryVersion,
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "http://example.com/charts",
								ChartName:    "test-chart",
								ChartVersion: defaultAppSecondaryVersion,
							},
						},
					},
				},
			},
			Enforced: enforced,
			Default:  defaulted,
			Selector: appskubermaticv1.DefaultingSelector{
				Datacenters: datacenters,
			},
		},
	}
}

func getApplicationInstallation(name string, appName string, appVersion string, labels map[string]string) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
			Labels:    labels,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &appskubermaticv1.AppNamespaceSpec{
				Name:   name,
				Create: true,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    appName,
				Version: appVersion,
			},
		},
	}
}

func genCluster(name, datacenter string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenter,
			},
		},
	}
}
