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

package resources

import (
	"context"
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestParseFluentBitRecords(t *testing.T) {
	tests := []struct {
		name           string
		cluster        *kubermaticv1.Cluster
		secrets        []runtime.Object
		configMaps     []runtime.Object
		expectedConfig *kubermaticv1.AuditSidecarConfiguration
		expectedError  bool
	}{
		{
			name:          "nil cluster returns error",
			cluster:       nil,
			expectedError: true,
		},
		{
			name: "no audit logging config should return an empty config",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-test-cluster",
				},
			},
			expectedConfig: &kubermaticv1.AuditSidecarConfiguration{},
		},
		{
			name: "config without variables passes through unchanged",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{
					AuditLogging: &kubermaticv1.AuditLoggingSettings{
						Enabled: true,
						SidecarSettings: &kubermaticv1.AuditSidecarSettings{
							Config: &kubermaticv1.AuditSidecarConfiguration{
								Filters: []map[string]string{
									{
										"Name":   "record_modifier",
										"Match":  "*",
										"Record": "cluster test-cluster",
									},
								},
							},
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-test-cluster",
				},
			},
			expectedConfig: &kubermaticv1.AuditSidecarConfiguration{
				Filters: []map[string]string{
					{
						"Name":   "record_modifier",
						"Match":  "*",
						"Record": "cluster test-cluster",
					},
				},
			},
		},
		{
			name: "config with environment variables gets expanded",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{
					AuditLogging: &kubermaticv1.AuditLoggingSettings{
						Enabled: true,
						SidecarSettings: &kubermaticv1.AuditSidecarSettings{
							Config: &kubermaticv1.AuditSidecarConfiguration{
								Filters: []map[string]string{
									{
										"Name":   "record_modifier",
										"Match":  "*",
										"Record": "cluster ${CLUSTER_ID}",
										"Extra":  "value with ${CUSTOM_VAR}",
									},
								},
							},
							ExtraEnvs: []corev1.EnvVar{
								{
									Name:  "CUSTOM_VAR",
									Value: "custom-value",
								},
							},
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-test-cluster",
				},
			},
			expectedConfig: &kubermaticv1.AuditSidecarConfiguration{
				Filters: []map[string]string{
					{
						"Name":   "record_modifier",
						"Match":  "*",
						"Record": "cluster test-cluster",
						"Extra":  "value with custom-value",
					},
				},
			},
		},
		{
			name: "multiple variables in one string get expanded",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{
					AuditLogging: &kubermaticv1.AuditLoggingSettings{
						Enabled: true,
						SidecarSettings: &kubermaticv1.AuditSidecarSettings{
							Config: &kubermaticv1.AuditSidecarConfiguration{
								Filters: []map[string]string{
									{
										"Name":   "record_modifier",
										"Match":  "*",
										"Record": "cluster ${CLUSTER_ID} in region ${REGION}",
									},
								},
							},
							ExtraEnvs: []corev1.EnvVar{
								{
									Name:  "REGION",
									Value: "eu-central-1",
								},
							},
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-test-cluster",
				},
			},
			expectedConfig: &kubermaticv1.AuditSidecarConfiguration{
				Filters: []map[string]string{
					{
						"Name":   "record_modifier",
						"Match":  "*",
						"Record": "cluster test-cluster in region eu-central-1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, 0)
			if tt.cluster != nil {
				objects = append(objects, tt.cluster)
			}
			objects = append(objects, tt.secrets...)
			objects = append(objects, tt.configMaps...)

			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(newTestScheme(t)).
				WithRuntimeObjects(objects...).
				Build()

			td := &TemplateData{
				ctx:     context.Background(),
				client:  client,
				cluster: tt.cluster,
			}

			config, err := td.ParseFluentBitRecords()

			if (err != nil) != tt.expectedError {
				t.Errorf("ParseFluentBitRecords() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !reflect.DeepEqual(config, tt.expectedConfig) {
				t.Errorf("ParseFluentBitRecords() = %v, want %v", config, tt.expectedConfig)
			}
		})
	}
}

func TestExpandVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:     "no variables",
			input:    "plain text",
			vars:     nil,
			expected: "plain text",
		},
		{
			name:     "single variable",
			input:    "Hello, ${NAME}!",
			vars:     map[string]string{"NAME": "World"},
			expected: "Hello, World!",
		},
		{
			name:     "multiple variables",
			input:    "${GREETING}, ${NAME}!",
			vars:     map[string]string{"GREETING": "Hello", "NAME": "World"},
			expected: "Hello, World!",
		},
		{
			name:     "undefined variable stays as is",
			input:    "Hello, ${UNDEFINED}!",
			vars:     map[string]string{"NAME": "World"},
			expected: "Hello, ${UNDEFINED}!",
		},
		{
			name:     "empty value for variable",
			input:    "Hello, ${EMPTY}!",
			vars:     map[string]string{"EMPTY": ""},
			expected: "Hello, !",
		},
		{
			name:     "malformed variable",
			input:    "Hello, $MALFORMED}!",
			vars:     map[string]string{"MALFORMED": "Wrong"},
			expected: "Hello, $MALFORMED}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandVariables(tt.input, tt.vars)
			if result != tt.expected {
				t.Errorf("expandVariables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func newTestScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}

	err = kubermaticv1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}

	return scheme
}
