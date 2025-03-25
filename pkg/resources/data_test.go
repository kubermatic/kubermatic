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
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetCSIMigrationFeatureGates(t *testing.T) {
	testCases := []struct {
		name             string
		cluster          *kubermaticv1.Cluster
		wantFeatureGates sets.Set[string]
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
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *semver.NewSemverOrDie("v1.1.1"),
					},
				},
			},
			wantFeatureGates: sets.Set[string]{},
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
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *semver.NewSemverOrDie("v1.1.1"),
					},
				},
			},
			wantFeatureGates: sets.Set[string]{},
		},
		{
			name: "CSI migration completed with k8s >= 1.23",
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
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Version: *semver.NewSemverOrDie("1.23.5"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *semver.NewSemverOrDie("1.23.5"),
					},
					Conditions: map[kubermaticv1.ClusterConditionType]kubermaticv1.ClusterCondition{
						kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted: {
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantFeatureGates: sets.New("InTreePluginOpenStackUnregister=true"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			td := NewTemplateDataBuilder().
				WithCluster(tc.cluster).
				Build()
			if a, e := sets.New(td.GetCSIMigrationFeatureGates(nil)...), tc.wantFeatureGates; !a.Equal(e) {
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

func TestNetworkInterfaceManagerImage(t *testing.T) {
	testCases := []struct {
		name                    string
		templateData            *TemplateData
		wantNetworkIntfMgrImage string
	}{
		{
			name: "default image",
			templateData: &TemplateData{
				networkIntfMgrImage: "quay.io/kubermatic/network-interface-manager",
			},
			wantNetworkIntfMgrImage: "quay.io/kubermatic/network-interface-manager",
		},
		{
			name: "default image with overwrite registry",
			templateData: &TemplateData{
				networkIntfMgrImage: "quay.io/kubermatic/network-interface-manager",
				OverwriteRegistry:   "custom-registry.kubermatic.io",
			},
			wantNetworkIntfMgrImage: "custom-registry.kubermatic.io/kubermatic/network-interface-manager",
		},
		{
			name: "custom image with 2 parts",
			templateData: &TemplateData{
				networkIntfMgrImage: "kubermatic/network-interface-manager",
			},
			wantNetworkIntfMgrImage: "docker.io/kubermatic/network-interface-manager",
		},
		{
			name: "custom image with 2 parts with overwrite registry",
			templateData: &TemplateData{
				networkIntfMgrImage: "kubermatic/network-interface-manager",
				OverwriteRegistry:   "custom-registry.kubermatic.io",
			},
			wantNetworkIntfMgrImage: "custom-registry.kubermatic.io/kubermatic/network-interface-manager",
		},
		{
			name: "custom image with 4 parts",
			templateData: &TemplateData{
				networkIntfMgrImage: "registry.kubermatic.io/images/kubermatic/network-interface-manager",
			},
			wantNetworkIntfMgrImage: "registry.kubermatic.io/images/kubermatic/network-interface-manager",
		},
		{
			name: "custom image with 4 parts with overwrite registry",
			templateData: &TemplateData{
				networkIntfMgrImage: "registry.kubermatic.io/images/kubermatic/network-interface-manager",
				OverwriteRegistry:   "custom-registry.kubermatic.io",
			},
			wantNetworkIntfMgrImage: "custom-registry.kubermatic.io/images/kubermatic/network-interface-manager",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if img := tc.templateData.NetworkIntfMgrImage(); img != tc.wantNetworkIntfMgrImage {
				t.Errorf("want network-interface-manager image %q, but got %q", tc.wantNetworkIntfMgrImage, img)
			}
		})
	}
}

func TestGetKonnectivityAgentArgs(t *testing.T) {
	testCases := []struct {
		name         string
		templateData *TemplateData
		want         []string
		wantErr      bool
		errMsg       string
	}{
		{
			name: "nil cluster returns error",
			templateData: &TemplateData{
				cluster: nil,
			},
			want:    nil,
			wantErr: true,
			errMsg:  "invalid cluster template, user cluster template is nil",
		},
		{
			name: "valid cluster with KonnectivityProxy args",
			templateData: &TemplateData{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: []string{
									"--arg1=value1",
									"--arg2=value2",
								},
							},
						},
					},
				},
			},
			want:    []string{"--arg1=value1", "--arg2=value2"},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "valid cluster with empty KonnectivityProxy args",
			templateData: &TemplateData{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: []string{},
							},
						},
					},
				},
			},
			want:    []string{},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "valid cluster with nil KonnectivityProxy args",
			templateData: &TemplateData{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: nil,
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "valid cluster with no ComponentsOverride",
			templateData: &TemplateData{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{},
				},
			},
			want:    nil,
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.templateData.GetKonnectivityAgentArgs()

			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error, got = %v, want = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("unexpected error message, got = %v, want = %v", err.Error(), tt.errMsg)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unexpected result, got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestGetKonnectivityServerArgs(t *testing.T) {
	tests := []struct {
		name           string
		seed           *kubermaticv1.Seed
		objects        []ctrlruntimeclient.Object
		expectedArgs   []string
		expectedErrMsg string
	}{
		{
			name:           "nil seed returns error",
			seed:           nil,
			expectedErrMsg: "invalid cluster template, seed cluster template is nil",
		},
		{
			name: "args directly from seed",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: []string{"--arg1=value1", "--arg2=value2"},
						},
					},
				},
			},
			expectedArgs: []string{"--arg1=value1", "--arg2=value2"},
		},
		{
			name: "empty default cluster template returns nil args",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "",
				},
			},
			expectedArgs: nil,
		},
		{
			name: "prefer direct args over template",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: []string{"--direct-arg1=value1", "--direct-arg2=value2"},
						},
					},
					DefaultClusterTemplate: "test-template",
				},
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.ClusterTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"scope": kubermaticv1.SeedTemplateScope,
						},
					},
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: []string{"--template-arg1=value1", "--template-arg2=value2"},
							},
						},
					},
				},
			},
			expectedArgs: []string{"--direct-arg1=value1", "--direct-arg2=value2"},
		},
		{
			name: "client error when getting template",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "test-template",
				},
			},
			expectedErrMsg: "failed to get ClusterTemplate for konnectivity",
		},
		{
			name: "invalid template scope",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "test-template",
				},
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.ClusterTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"scope": "invalid-scope",
						},
					},
				},
			},
			expectedErrMsg: fmt.Sprintf(
				"invalid scope of default cluster template, is %q but must be %q",
				"invalid-scope",
				kubermaticv1.SeedTemplateScope,
			),
		},
		{
			name: "success from template",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "test-template",
				},
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.ClusterTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"scope": kubermaticv1.SeedTemplateScope,
						},
					},
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: []string{"--template-arg1=value1", "--template-arg2=value2"},
							},
						},
					},
				},
			},
			expectedArgs: []string{"--template-arg1=value1", "--template-arg2=value2"},
		},
		{
			name: "nil args from template results in nil",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "test-template",
				},
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.ClusterTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"scope": kubermaticv1.SeedTemplateScope,
						},
					},
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: nil,
							},
						},
					},
				},
			},
			expectedArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []ctrlruntimeclient.Object
			if tt.seed != nil {
				objects = append(objects, tt.seed)
			}

			objects = append(objects, tt.objects...)

			cl := fake.NewClientBuilder().
				WithObjects(objects...).
				Build()

			templateData := &TemplateData{
				ctx:    context.Background(),
				client: cl,
				seed:   tt.seed,
			}

			args, err := templateData.GetKonnectivityServerArgs()

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedArgs, args)
			}
		})
	}
}
