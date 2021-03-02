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
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
)

func init() {
	if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to add clusterv1alpha1 to scheme: %v", err))
	}
}

func TestCloudControllerManagerDeployment(t *testing.T) {
	// these tests use openstack as an example for a provider that has
	// a CCM; the logic tested here is independent of the provider itself

	testCases := []struct {
		name                string
		cluster             *kubermaticv1.Cluster
		kcmDeploymentConfig KCMDeploymentConfig
		wantCCMCreator      bool
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
			wantCCMCreator: true,
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
			wantCCMCreator: true,
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
			wantCCMCreator: false,
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
			wantCCMCreator: false,
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
			wantCCMCreator: false,
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
			wantCCMCreator: true,
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
			wantCCMCreator: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			caBundle := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tc.cluster.Status.NamespaceName,
					Name:      resources.CABundleConfigMapName,
				},
				Data: map[string]string{
					// this is just a dummy cert to make sure the bundle is not empty
					resources.CABundleConfigMapKey: `-----BEGIN CERTIFICATE-----
MIIDdTCCAl2gAwIBAgILBAAAAAABFUtaw5QwDQYJKoZIhvcNAQEFBQAwVzELMAkGA1UEBhMCQkUx
GTAXBgNVBAoTEEdsb2JhbFNpZ24gbnYtc2ExEDAOBgNVBAsTB1Jvb3QgQ0ExGzAZBgNVBAMTEkds
b2JhbFNpZ24gUm9vdCBDQTAeFw05ODA5MDExMjAwMDBaFw0yODAxMjgxMjAwMDBaMFcxCzAJBgNV
BAYTAkJFMRkwFwYDVQQKExBHbG9iYWxTaWduIG52LXNhMRAwDgYDVQQLEwdSb290IENBMRswGQYD
VQQDExJHbG9iYWxTaWduIFJvb3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDa
DuaZjc6j40+Kfvvxi4Mla+pIH/EqsLmVEQS98GPR4mdmzxzdzxtIK+6NiY6arymAZavpxy0Sy6sc
THAHoT0KMM0VjU/43dSMUBUc71DuxC73/OlS8pF94G3VNTCOXkNz8kHp1Wrjsok6Vjk4bwY8iGlb
Kk3Fp1S4bInMm/k8yuX9ifUSPJJ4ltbcdG6TRGHRjcdGsnUOhugZitVtbNV4FpWi6cgKOOvyJBNP
c1STE4U6G7weNLWLBYy5d4ux2x8gkasJU26Qzns3dLlwR5EiUWMWea6xrkEmCMgZK9FGqkjWZCrX
gzT/LCrBbBlDSgeF59N89iFo7+ryUp9/k5DPAgMBAAGjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNV
HRMBAf8EBTADAQH/MB0GA1UdDgQWBBRge2YaRQ2XyolQL30EzTSo//z9SzANBgkqhkiG9w0BAQUF
AAOCAQEA1nPnfE920I2/7LqivjTFKDK1fPxsnCwrvQmeU79rXqoRSLblCKOzyj1hTdNGCbM+w6Dj
Y1Ub8rrvrTnhQ7k4o+YviiY776BQVvnGCv04zcQLcFGUl5gE38NflNUVyRRBnMRddWQVDf9VMOyG
j/8N7yy5Y0b2qvzfvGn9LhJIZJrglfCm7ymPAbEVtQwdpf5pLGkkeB6zpxxxYu7KyJesF12KwvhH
hm4qxFYxldBniYUr+WymXUadDKqC5JlR3XC321Y9YeRq4VzW9v493kHMB65jUr9TU/Qr6cf9tveC
X4XSQRjbgbMEHMUfpIBvFSDJ3gyICh3WZlXi/EjJKSZp4A==
-----END CERTIFICATE-----`,
				},
			}

			fc := fake.NewClientBuilder().WithObjects(caBundle).Build()
			td := resources.NewTemplateDataBuilder().
				WithContext(ctx).
				WithClient(fc).
				WithCluster(tc.cluster).
				WithCABundleFile("../../../../charts/kubermatic-operator/static/ca-bundle.pem").
				Build()
			// Add the KCM deployment
			if err := fc.Create(ctx, tc.kcmDeploymentConfig.Create(td)); err != nil {
				t.Fatalf("error occurred while creating KCM deployment: %v", err)
			}
			creators := GetDeploymentCreators(td, false)
			var ccmDeploymentFound bool
			for _, c := range creators {
				name, _ := c()
				if name == cloudcontroller.OpenstackCCMDeploymentName {
					ccmDeploymentFound = true
				}
			}
			if a, e := tc.wantCCMCreator, ccmDeploymentFound; a != e {
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
							Image:   "my-registy.io/kube-controller-manager:v1.18",
							Command: []string{"/usr/local/bin/kube-controller-manager"},
							Args:    k.Flags,
						},
					},
				},
			},
		},
		Status: k.Status,
	}
	wrappedPodSpec, _ := apiserver.IsRunningWrapper(td, d.Spec.Template.Spec, sets.NewString(resources.ControllerManagerDeploymentName))
	d.Spec.Template.Spec = *wrappedPodSpec
	return &d
}
