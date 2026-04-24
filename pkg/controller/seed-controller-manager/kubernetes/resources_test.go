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

package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCloudControllerManagerDeployment(t *testing.T) {
	// these tests use openstack as an example for a provider that has
	// a CCM; the logic tested here is independent of the provider itself

	testCases := []struct {
		name                string
		cluster             *kubermaticv1.Cluster
		kcmDeploymentConfig KCMDeploymentConfig
		wantCCMReconciler   bool
	}{
		{
			name: "KCM ready and cloud-provider disabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           1,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   1,
				Namespace:  "test",
			},
			wantCCMReconciler: true,
		},
		{
			name: "KCM ready and cloud controllers disabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{"--cloud-provider", "openstack", "--controllers", "-cloud-node-lifecycle,-route,-service"},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           1,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   1,
				Namespace:  "test",
			},
			wantCCMReconciler: true,
		},
		{
			name: "KCM ready and service controller not disabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{"--cloud-provider", "openstack", "--controllers", "-cloud-node-lifecycle,-route"},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           1,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   1,
				Namespace:  "test",
			},
			wantCCMReconciler: false,
		},
		{
			// If the KCM deployment rollout is not completed we do not deploy the
			// CCM as there could be old KCM pods with cloud controllers
			// running.
			name: "KCM not ready and cloud-provider disabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           2,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   2,
				Namespace:  "test",
			},
			wantCCMReconciler: false,
		},
		{
			name: "KCM ready and cloud-provider enabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{"--cloud-provider", "openstack"},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           1,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   1,
				Namespace:  "test",
			},
			wantCCMReconciler: false,
		},
		{
			name: "No CCM migration ongoing",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           2,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   2,
				Namespace:  "test",
			},
			wantCCMReconciler: true,
		},
		{
			name: "No external cloud-provider",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
			},
			kcmDeploymentConfig: KCMDeploymentConfig{
				Flags: []string{},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					Replicas:           1,
					AvailableReplicas:  1,
					UpdatedReplicas:    1,
				},
				Generation: 1,
				Replicas:   1,
				Namespace:  "test",
			},
			wantCCMReconciler: false,
		},
	}

	caBundle := certificates.NewFakeCABundle()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			caBundleConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tc.cluster.Status.NamespaceName,
					Name:      resources.CABundleConfigMapName,
				},
				Data: map[string]string{
					resources.CABundleConfigMapKey: caBundle.String(),
				},
			}

			fc := fake.NewClientBuilder().WithObjects(caBundleConfigMap).Build()
			td := resources.NewTemplateDataBuilder().
				WithContext(ctx).
				WithClient(fc).
				WithCluster(tc.cluster).
				WithCABundle(caBundle).
				Build()
			// Add the KCM deployment
			if err := fc.Create(ctx, tc.kcmDeploymentConfig.Create(td)); err != nil {
				t.Fatalf("error occurred while creating KCM deployment: %v", err)
			}
			creators := GetDeploymentReconcilers(td, Features{}, kubermatic.GetFakeVersions())
			var ccmDeploymentFound bool
			for _, c := range creators {
				name, _ := c()
				if name == cloudcontroller.OpenstackCCMDeploymentName {
					ccmDeploymentFound = true
				}
			}
			if a, e := tc.wantCCMReconciler, ccmDeploymentFound; a != e {
				t.Errorf("want CCM creator: %t got: %t", a, e)
			}
		})
	}
}

type KCMDeploymentConfig struct {
	Flags      []string
	Generation int64
	Namespace  string
	Replicas   int32
	Status     appsv1.DeploymentStatus
}

func (k KCMDeploymentConfig) Create(td *resources.TemplateData) *appsv1.Deployment {
	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       resources.ControllerManagerDeploymentName,
			Namespace:  k.Namespace,
			Generation: k.Generation,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &k.Replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    resources.ControllerManagerDeploymentName,
							Image:   "my-registry.io/kube-controller-manager:v1.18",
							Command: []string{"/usr/local/bin/kube-controller-manager"},
							Args:    k.Flags,
						},
					},
				},
			},
		},
		Status: k.Status,
	}
	d.Spec.Template, _ = apiserver.IsRunningWrapper(td, d.Spec.Template, sets.New(resources.ControllerManagerDeploymentName))
	return &d
}

func TestResolveAuthenticationConfigurationYAML(t *testing.T) {
	seedLevelYAML := []byte("seed-level-auth-config")
	dcLevelYAML := []byte("datacenter-level-auth-config")
	fakeSecrets := []ctrlruntimeclient.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "seed-auth-secret",
				Namespace: resources.KubermaticNamespace,
			},
			Data: map[string][]byte{
				"config.yaml": seedLevelYAML,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dc-auth-secret",
				Namespace: resources.KubermaticNamespace,
			},
			Data: map[string][]byte{
				"config.yaml": dcLevelYAML,
			},
		},
	}

	tests := []struct {
		name           string
		datacenterConf *kubermaticv1.AuthenticationConfiguration
		seedConf       *kubermaticv1.AuthenticationConfiguration
		expectedResult []byte
		expectError    string
	}{
		{
			name: "no datacenter config and no seed config returns nil",
		},
		{
			name: "seed-level config is returned when no datacenter config specified",
			seedConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "seed-auth-secret",
				SecretKey:  "config.yaml",
			},
			expectedResult: seedLevelYAML,
		},
		{
			name: "datacenter config takes precedence over seed-level",
			datacenterConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "dc-auth-secret",
				SecretKey:  "config.yaml",
			},
			seedConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "seed-auth-secret",
				SecretKey:  "config.yaml",
			},
			expectedResult: dcLevelYAML,
		},
		{
			name: "datacenter config with missing secret returns error",
			datacenterConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "nonexistent-secret",
				SecretKey:  "config.yaml",
			},
			seedConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "seed-auth-secret",
				SecretKey:  "config.yaml",
			},
			expectError: "failed to read authentication configuration secret",
		},
		{
			name: "datacenter config with missing key in secret returns error",
			datacenterConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "dc-auth-secret",
				SecretKey:  "missing-key",
			},
			seedConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "seed-auth-secret",
				SecretKey:  "config.yaml",
			},
			expectError: "does not contain key",
		},
		{
			name: "seed config with missing secret returns error",
			seedConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "nonexistent-secret",
				SecretKey:  "config.yaml",
			},
			expectError: "failed to read authentication configuration secret",
		},
		{
			name: "seed config with missing key in secret returns error",
			seedConf: &kubermaticv1.AuthenticationConfiguration{
				SecretName: "seed-auth-secret",
				SecretKey:  "missing-key",
			},
			expectError: "does not contain key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			client := fake.NewClientBuilder().WithObjects(fakeSecrets...).Build()

			r := &Reconciler{
				Client: client,
			}

			datacenter := &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Fake:                        &kubermaticv1.DatacenterSpecFake{},
					AuthenticationConfiguration: tt.datacenterConf,
				},
			}
			seed := &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					AuthenticationConfiguration: tt.seedConf,
				},
			}

			result, err := r.resolveAuthenticationConfigurationYAML(ctx, datacenter, seed)
			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestIssuerURLsFromAuthenticationConfigurationYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        []byte
		expected    []string
		expectError string
	}{
		{
			name: "v1 extracts issuer URLs",
			yaml: []byte(`apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
jwt:
  - issuer:
      url: https://issuer-1.example
  - issuer:
      url: https://issuer-2.example
`),
			expected: []string{"https://issuer-1.example", "https://issuer-2.example"},
		},
		{
			name: "v1beta1 extracts issuer URLs",
			yaml: []byte(`apiVersion: apiserver.config.k8s.io/v1beta1
kind: AuthenticationConfiguration
jwt:
  - issuer:
      url: https://issuer-beta.example
`),
			expected: []string{"https://issuer-beta.example"},
		},
		{
			name: "missing apiVersion returns error",
			yaml: []byte(`kind: AuthenticationConfiguration
jwt:
  - issuer:
      url: https://issuer.example
`),
			expectError: "failed to decode AuthenticationConfiguration YAML",
		},
		{
			name: "unsupported apiVersion returns error",
			yaml: []byte(`apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthenticationConfiguration
jwt:
  - issuer:
      url: https://issuer.example
`),
			expectError: "failed to decode AuthenticationConfiguration YAML",
		},
		{
			name:        "invalid yaml returns error",
			yaml:        []byte(`jwt: [bad`),
			expectError: "failed to decode AuthenticationConfiguration YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuerURLs, err := issuerURLsFromAuthenticationConfigurationYAML(tt.yaml)
			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, issuerURLs)
		})
	}
}

func TestURLsToIPList(t *testing.T) {
	tests := []struct {
		name        string
		urls        []string
		expectedIPs []string
		expectError string
	}{
		{
			name: "deduplicates repeated hosts",
			urls: []string{
				"https://10.10.10.10/path-a",
				"https://10.10.10.10/path-b",
				"https://20.20.20.20",
			},
			expectedIPs: []string{"10.10.10.10", "20.20.20.20"},
		},
		{
			name: "supports ipv6 hosts",
			urls: []string{
				"https://[2001:db8::1]",
				"https://[2001:db8::2]",
			},
			expectedIPs: []string{"2001:db8::1", "2001:db8::2"},
		},
		{
			name:        "invalid URL returns error",
			urls:        []string{"://not-a-url"},
			expectError: "failed to parse URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips, err := urlsToIPList(context.Background(), tt.urls)
			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)
				return
			}

			require.NoError(t, err)

			actual := make([]string, 0, len(ips))
			for _, ip := range ips {
				actual = append(actual, ip.String())
			}

			require.Equal(t, tt.expectedIPs, actual)
		})
	}
}
