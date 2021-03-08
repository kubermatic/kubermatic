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
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/test"
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
		name           string
		cluster        *kubermaticv1.Cluster
		kcmDeployment  *appsv1.Deployment
		wantCCMCreator bool
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				AddContainer().
				WithGeneration(1).
				WithReplicas(1).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				WithArgs("--cloud-provider", "openstack", "--controllers", "-cloud-node-lifecycle,-route,-service").
				AddContainer().
				WithGeneration(1).
				WithReplicas(1).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				WithArgs("--cloud-provider", "openstack", "--controllers", "-cloud-node-lifecycle,-route").
				AddContainer().
				WithGeneration(1).
				WithReplicas(1).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				AddContainer().
				WithGeneration(1).
				WithReplicas(2).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 2, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				WithArgs("--cloud-provider", "openstack").
				AddContainer().
				WithGeneration(1).
				WithReplicas(1).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				WithArgs("--cloud-provider", "openstack").
				AddContainer().
				WithGeneration(1).
				WithReplicas(2).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 2, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
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
			kcmDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: resources.ControllerManagerDeploymentName, Namespace: "test"}).
				ContainerBuilder().
				WithName(resources.ControllerManagerDeploymentName).
				WithImage("my-registy.io/kube-controller-manager:v1.18").
				WithCommand("/usr/local/bin/kube-controller-manager").
				AddContainer().
				WithGeneration(1).
				WithReplicas(1).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}).Build(),
			wantCCMCreator: false,
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
			if err := fc.Create(ctx, wrapControllerManagerContainer(t, tc.kcmDeployment, td)); err != nil {
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

func wrapControllerManagerContainer(t *testing.T, d *appsv1.Deployment, td *resources.TemplateData) *appsv1.Deployment {
	wrappedPodSpec, _ := apiserver.IsRunningWrapper(td, d.Spec.Template.Spec, sets.NewString(resources.ControllerManagerDeploymentName))
	t.Logf("Deployment: %+v\n", d.Spec.Template)
	d.Spec.Template.Spec = *wrappedPodSpec
	return d
}
