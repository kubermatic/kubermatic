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

package resources

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetCSIMigrationFeatureGates(t *testing.T) {
	testCases := []struct {
		name             string
		cluster          *kubermaticv1.Cluster
		wantFeatureGates sets.String
	}{
		{
			name: "No CSI migration",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cluster-a",
					Annotations: map[string]string{},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			wantFeatureGates: sets.String{},
		},
		{
			name: "CSI migration",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			wantFeatureGates: sets.NewString("CSIMigration=true", "CSIMigrationOpenStack=true", "ExpandCSIVolumes=true"),
		},
		{
			name: "CSI migration completed",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
					Conditions: []kubermaticv1.ClusterCondition{
						{
							Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantFeatureGates: sets.NewString("CSIMigration=true", "CSIMigrationOpenStack=true", "ExpandCSIVolumes=true", "CSIMigrationOpenStackComplete=true"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			td := NewTemplateDataBuilder().
				WithCluster(tc.cluster).
				Build()
			if a, e := sets.NewString(td.GetCSIMigrationFeatureGates()...), tc.wantFeatureGates; !a.Equal(e) {
				t.Errorf("Want feature gates %v, but got %v", e, a)
			}
		})
	}
}

func TestKubermaticAPIImage(t *testing.T) {
	testCases := []struct {
		name         string
		templateData *TemplateData
		wantAPIImage string
	}{
		{
			name: "default image",
			templateData: &TemplateData{
				kubermaticImage: "quay.io/kubermatic/kubermatic",
			},
			wantAPIImage: "quay.io/kubermatic/kubermatic",
		},
		{
			name: "default image with overwrite registry",
			templateData: &TemplateData{
				kubermaticImage:   "quay.io/kubermatic/kubermatic",
				OverwriteRegistry: "custom-registry.kubermatic.io",
			},
			wantAPIImage: "custom-registry.kubermatic.io/kubermatic/kubermatic",
		},
		{
			name: "custom image with 2 parts",
			templateData: &TemplateData{
				kubermaticImage: "kubermatic/kubermatic",
			},
			wantAPIImage: "docker.io/kubermatic/kubermatic",
		},
		{
			name: "custom image with 2 parts with overwrite registry",
			templateData: &TemplateData{
				kubermaticImage:   "kubermatic/kubermatic",
				OverwriteRegistry: "custom-registry.kubermatic.io",
			},
			wantAPIImage: "custom-registry.kubermatic.io/kubermatic/kubermatic",
		},
		{
			name: "custom image with 4 parts",
			templateData: &TemplateData{
				kubermaticImage: "registry.kubermatic.io/images/kubermatic/kubermatic",
			},
			wantAPIImage: "registry.kubermatic.io/images/kubermatic/kubermatic",
		},
		{
			name: "custom image with 4 parts with overwrite registry",
			templateData: &TemplateData{
				kubermaticImage:   "registry.kubermatic.io/images/kubermatic/kubermatic",
				OverwriteRegistry: "custom-registry.kubermatic.io",
			},
			wantAPIImage: "custom-registry.kubermatic.io/images/kubermatic/kubermatic",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if img := tc.templateData.KubermaticAPIImage(); img != tc.wantAPIImage {
				t.Errorf("want kubermatic api image %q, but got %q", tc.wantAPIImage, img)
			}
		})
	}
}

func TestEtcdLauncherImage(t *testing.T) {
	testCases := []struct {
		name                  string
		templateData          *TemplateData
		wantEtcdLauncherImage string
	}{
		{
			name: "default image",
			templateData: &TemplateData{
				etcdLauncherImage: "quay.io/kubermatic/etcd-launcher",
			},
			wantEtcdLauncherImage: "quay.io/kubermatic/etcd-launcher",
		},
		{
			name: "default image with overwrite registry",
			templateData: &TemplateData{
				etcdLauncherImage: "quay.io/kubermatic/etcd-launcher",
				OverwriteRegistry: "custom-registry.kubermatic.io",
			},
			wantEtcdLauncherImage: "custom-registry.kubermatic.io/kubermatic/etcd-launcher",
		},
		{
			name: "custom image with 2 parts",
			templateData: &TemplateData{
				etcdLauncherImage: "kubermatic/etcd-launcher",
			},
			wantEtcdLauncherImage: "docker.io/kubermatic/etcd-launcher",
		},
		{
			name: "custom image with 2 parts with overwrite registry",
			templateData: &TemplateData{
				etcdLauncherImage: "kubermatic/etcd-launcher",
				OverwriteRegistry: "custom-registry.kubermatic.io",
			},
			wantEtcdLauncherImage: "custom-registry.kubermatic.io/kubermatic/etcd-launcher",
		},
		{
			name: "custom image with 4 parts",
			templateData: &TemplateData{
				etcdLauncherImage: "registry.kubermatic.io/images/kubermatic/etcd-launcher",
			},
			wantEtcdLauncherImage: "registry.kubermatic.io/images/kubermatic/etcd-launcher",
		},
		{
			name: "custom image with 4 parts with overwrite registry",
			templateData: &TemplateData{
				etcdLauncherImage: "registry.kubermatic.io/images/kubermatic/etcd-launcher",
				OverwriteRegistry: "custom-registry.kubermatic.io",
			},
			wantEtcdLauncherImage: "custom-registry.kubermatic.io/images/kubermatic/etcd-launcher",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if img := tc.templateData.EtcdLauncherImage(); img != tc.wantEtcdLauncherImage {
				t.Errorf("want etcd-launcher image %q, but got %q", tc.wantEtcdLauncherImage, img)
			}
		})
	}
}

func TestDNATControllerImage(t *testing.T) {
	testCases := []struct {
		name                    string
		templateData            *TemplateData
		wantDNATControllerImage string
	}{
		{
			name: "default image",
			templateData: &TemplateData{
				dnatControllerImage: "quay.io/kubermatic/kubeletdnat-controller",
			},
			wantDNATControllerImage: "quay.io/kubermatic/kubeletdnat-controller",
		},
		{
			name: "default image with overwrite registry",
			templateData: &TemplateData{
				dnatControllerImage: "quay.io/kubermatic/kubeletdnat-controller",
				OverwriteRegistry:   "custom-registry.kubermatic.io",
			},
			wantDNATControllerImage: "custom-registry.kubermatic.io/kubermatic/kubeletdnat-controller",
		},
		{
			name: "custom image with 2 parts",
			templateData: &TemplateData{
				dnatControllerImage: "kubermatic/kubeletdnat-controller",
			},
			wantDNATControllerImage: "docker.io/kubermatic/kubeletdnat-controller",
		},
		{
			name: "custom image with 2 parts with overwrite registry",
			templateData: &TemplateData{
				dnatControllerImage: "kubermatic/kubeletdnat-controller",
				OverwriteRegistry:   "custom-registry.kubermatic.io",
			},
			wantDNATControllerImage: "custom-registry.kubermatic.io/kubermatic/kubeletdnat-controller",
		},
		{
			name: "custom image with 4 parts",
			templateData: &TemplateData{
				dnatControllerImage: "registry.kubermatic.io/images/kubermatic/kubeletdnat-controller",
			},
			wantDNATControllerImage: "registry.kubermatic.io/images/kubermatic/kubeletdnat-controller",
		},
		{
			name: "custom image with 4 parts with overwrite registry",
			templateData: &TemplateData{
				dnatControllerImage: "registry.kubermatic.io/images/kubermatic/kubeletdnat-controller",
				OverwriteRegistry:   "custom-registry.kubermatic.io",
			},
			wantDNATControllerImage: "custom-registry.kubermatic.io/images/kubermatic/kubeletdnat-controller",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if img := tc.templateData.DNATControllerImage(); img != tc.wantDNATControllerImage {
				t.Errorf("want kubeletdnat-controller image %q, but got %q", tc.wantDNATControllerImage, img)
			}
		})
	}
}
