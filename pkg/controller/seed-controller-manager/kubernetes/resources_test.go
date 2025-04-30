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
			creators := GetDeploymentReconcilers(td, false, kubermatic.GetFakeVersions())
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
