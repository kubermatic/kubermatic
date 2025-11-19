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

package operatingsystemmanager

import (
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

// TestAppendProxyFlags tests the appendProxyFlags function with various inputs.
func TestAppendProxyFlags(t *testing.T) {
	const (
		sampleProxy        = "http://proxy.example.com"
		sampleProxyAnother = "http://another.proxy.com"
		noProxy            = "noproxy"
		noProxyAnother     = "another-noproxy"
	)

	tests := []struct {
		name         string
		flags        []string
		nodeSettings *kubermaticv1.NodeSettings
		cluster      *kubermaticv1.Cluster
		want         []string
	}{
		{
			name:         "nil cluster and node settings should return original flags",
			flags:        []string{"-flag1", "value1"},
			nodeSettings: nil,
			cluster:      nil,
			want:         []string{"-flag1", "value1"},
		},
		{
			name:         "nil nodeSettings with an empty OSM should not add flags",
			flags:        []string{"-flag1", "value1"},
			nodeSettings: nil,
			cluster:      &kubermaticv1.Cluster{},
			want:         []string{"-flag1", "value1"},
		},
		{
			name:         "empty nodeSettings with nil OSM should not add flags",
			flags:        []string{"-flag1", "value1"},
			nodeSettings: &kubermaticv1.NodeSettings{},
			cluster:      nil,
			want:         []string{"-flag1", "value1"},
		},
		{
			name:  "nil cluster should not prevent updating flags",
			flags: []string{"-flag1", "value1"},
			nodeSettings: &kubermaticv1.NodeSettings{
				ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: kubermaticv1.NewProxyValue(sampleProxy),
				},
			},
			cluster: nil,
			want:    []string{"-flag1", "value1", flagHTTPProxy, sampleProxy},
		},
		{
			name:  "empty flags with valid node settings should add proxy flags",
			flags: []string{},
			nodeSettings: &kubermaticv1.NodeSettings{
				ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: kubermaticv1.NewProxyValue(sampleProxy),
				},
			},
			cluster: &kubermaticv1.Cluster{},
			want:    []string{flagHTTPProxy, sampleProxy},
		},
		{
			name:  "cluster settings should override nodeSettings",
			flags: []string{"-flag1", "value1"},
			nodeSettings: &kubermaticv1.NodeSettings{
				ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: kubermaticv1.NewProxyValue(sampleProxy),
					NoProxy:   kubermaticv1.NewProxyValue(noProxy),
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						OperatingSystemManager: &kubermaticv1.OSMControllerSettings{
							Proxy: kubermaticv1.ProxySettings{
								HTTPProxy: kubermaticv1.NewProxyValue(sampleProxyAnother),
							},
						},
					},
				},
			},
			want: []string{
				"-flag1", "value1",
				flagHTTPProxy, sampleProxyAnother,
				flagNoProxy, noProxy,
			},
		},
		{
			name:  "empty proxy values should not be added",
			flags: []string{"-flag1", "value1"},
			nodeSettings: &kubermaticv1.NodeSettings{
				ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: kubermaticv1.NewProxyValue(""),
					NoProxy:   kubermaticv1.NewProxyValue(""),
				},
			},
			cluster: &kubermaticv1.Cluster{},
			want:    []string{"-flag1", "value1"},
		},
		{
			name:  "both proxy types from nodeSettings should be added",
			flags: []string{"-flag1", "value1"},
			nodeSettings: &kubermaticv1.NodeSettings{
				ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: kubermaticv1.NewProxyValue(sampleProxy),
					NoProxy:   kubermaticv1.NewProxyValue(noProxy),
				},
			},
			cluster: &kubermaticv1.Cluster{},
			want: []string{
				"-flag1", "value1",
				flagHTTPProxy, sampleProxy,
				flagNoProxy, noProxy,
			},
		},
		{
			name:         "both proxy types from cluster should be added",
			flags:        []string{"-flag1", "value1"},
			nodeSettings: nil,
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						OperatingSystemManager: &kubermaticv1.OSMControllerSettings{
							Proxy: kubermaticv1.ProxySettings{
								HTTPProxy: kubermaticv1.NewProxyValue(sampleProxy),
								NoProxy:   kubermaticv1.NewProxyValue(noProxy),
							},
						},
					},
				},
			},
			want: []string{
				"-flag1", "value1",
				flagHTTPProxy, sampleProxy,
				flagNoProxy, noProxy,
			},
		},
		{
			name:         "nil OSM should not add flags",
			flags:        []string{"-flag1", "value1"},
			nodeSettings: nil,
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						OperatingSystemManager: nil,
					},
				},
			},
			want: []string{"-flag1", "value1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origFlags := make([]string, len(tt.flags))
			copy(origFlags, tt.flags)

			got := appendProxyFlags(tt.flags, tt.nodeSettings, tt.cluster)

			if tt.cluster == nil && !reflect.DeepEqual(tt.flags, origFlags) {
				t.Errorf("appendProxyFlags modified original slice when cluster was nil")
			}

			if !containsSameFlags(got, tt.want) {
				t.Errorf("appendProxyFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendContainerRuntimeFlags(t *testing.T) {
	tests := []struct {
		name       string
		flags      []string
		datacenter *kubermaticv1.Datacenter
		cluster    *kubermaticv1.Cluster
		want       []string
	}{
		{
			name:       "nil datacenter and cluster",
			flags:      []string{"-existing", "value"},
			datacenter: nil,
			cluster:    nil,
			want:       []string{"-existing", "value"},
		},
		{
			name:  "datacenter with insecure registries only",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"registry1.example.com", "registry2.example.com"},
					},
				},
			},
			cluster: nil,
			want:    []string{"-existing", "value", "-node-insecure-registries", "registry1.example.com,registry2.example.com"},
		},
		{
			name:  "datacenter with registry mirrors only",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						RegistryMirrors: []string{"mirror1.example.com", "mirror2.example.com"},
					},
				},
			},
			cluster: nil,
			want:    []string{"-existing", "value", "-node-registry-mirrors", "mirror1.example.com,mirror2.example.com"},
		},
		{
			name:  "datacenter with pause image only",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						PauseImage: "custom-pause:v1.0.0",
					},
				},
			},
			cluster: nil,
			want:    []string{"-existing", "value", "-pause-image", "custom-pause:v1.0.0"},
		},
		{
			name:  "datacenter with containerd registry mirrors only",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"mirror1.docker.io", "mirror2.docker.io"},
								},
								"gcr.io": {
									Mirrors: []string{"mirror.gcr.io"},
								},
							},
						},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-existing", "value",
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
			},
		},
		{
			name:  "datacenter with all container runtime options",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"insecure1.example.com", "insecure2.example.com"},
						RegistryMirrors:    []string{"mirror1.example.com", "mirror2.example.com"},
						PauseImage:         "datacenter-pause:v1.0.0",
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"dc-mirror.docker.io"},
								},
							},
						},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-existing", "value",
				"-node-registry-mirrors", "mirror1.example.com,mirror2.example.com",
				"-pause-image", "datacenter-pause:v1.0.0",
				"-node-insecure-registries", "insecure1.example.com,insecure2.example.com",
				"-node-containerd-registry-mirrors=docker.io=dc-mirror.docker.io",
			},
		},
		{
			name:  "cluster overrides datacenter - insecure registries",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"dc-registry.example.com"},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"cluster-registry.example.com"},
					},
				},
			},
			want: []string{"-existing", "value", "-node-insecure-registries", "cluster-registry.example.com"},
		},
		{
			name:  "cluster overrides datacenter - registry mirrors",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						RegistryMirrors: []string{"dc-mirror.example.com"},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						RegistryMirrors: []string{"cluster-mirror.example.com"},
					},
				},
			},
			want: []string{"-existing", "value", "-node-registry-mirrors", "cluster-mirror.example.com"},
		},
		{
			name:  "cluster overrides datacenter - pause image",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						PauseImage: "dc-pause:v1.0.0",
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						PauseImage: "cluster-pause:v2.0.0",
					},
				},
			},
			want: []string{"-existing", "value", "-pause-image", "cluster-pause:v2.0.0"},
		},
		{
			name:  "cluster overrides datacenter - containerd registry mirrors",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"dc-mirror.docker.io"},
								},
							},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"cluster-mirror.docker.io"},
								},
								"quay.io": {
									Mirrors: []string{"cluster-mirror.quay.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-existing", "value",
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror.docker.io",
				"-node-containerd-registry-mirrors=quay.io=cluster-mirror.quay.io",
			},
		},
		{
			name:  "empty slices and strings are ignored - datacenter",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{},
						RegistryMirrors:    []string{},
						PauseImage:         "",
					},
				},
			},
			cluster: nil,
			want:    []string{"-existing", "value"},
		},
		{
			name:  "empty slices and strings are ignored - cluster",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"dc-registry.example.com"},
						PauseImage:         "dc-pause:v1.0.0",
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{},
						RegistryMirrors:    []string{},
						PauseImage:         "",
					},
				},
			},
			want: []string{
				"-existing", "value",
				"-node-insecure-registries", "dc-registry.example.com",
				"-pause-image", "dc-pause:v1.0.0",
			},
		},
		{
			name:  "nil containerd registry mirrors are handled gracefully",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries:        []string{"dc-registry.example.com"},
						ContainerdRegistryMirrors: nil,
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries:        []string{"cluster-registry.example.com"},
						ContainerdRegistryMirrors: nil,
					},
				},
			},
			want: []string{"-existing", "value", "-node-insecure-registries", "cluster-registry.example.com"},
		},
		{
			name:  "empty containerd registries map produces no flags",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{},
						},
					},
				},
			},
			cluster: nil,
			want:    []string{"-existing", "value"},
		},
		{
			name:  "registries with empty mirrors are excluded",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{},
								},
								"gcr.io": {
									Mirrors: []string{"mirror.gcr.io"},
								},
							},
						},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-existing", "value",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
			},
		},
		{
			name:  "single empty string in slice is ignored",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{""},
						RegistryMirrors:    []string{""},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-existing", "value",
				"-node-insecure-registries", "",
				"-node-registry-mirrors", "",
			},
		},
		{
			name:  "mixed empty and valid strings",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"", "valid-registry.example.com", ""},
						RegistryMirrors:    []string{"valid-mirror.example.com", ""},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-existing", "value",
				"-node-insecure-registries", ",valid-registry.example.com,",
				"-node-registry-mirrors", "valid-mirror.example.com,",
			},
		},
		{
			name:  "datacenter with nil node settings",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: nil,
			},
			cluster: nil,
			want:    []string{"-existing", "value"},
		},
		{
			name:  "partial cluster override - only some fields present",
			flags: []string{"-existing", "value"},
			datacenter: &kubermaticv1.Datacenter{
				Node: &kubermaticv1.NodeSettings{
					ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"dc-insecure.example.com"},
						RegistryMirrors:    []string{"dc-mirror.example.com"},
						PauseImage:         "dc-pause:v1.0.0",
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						InsecureRegistries: []string{"cluster-insecure.example.com"},
					},
				},
			},
			want: []string{
				"-existing", "value",
				"-node-insecure-registries", "cluster-insecure.example.com",
				"-node-registry-mirrors", "dc-mirror.example.com",
				"-pause-image", "dc-pause:v1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := resources.NewTemplateDataBuilder().
				WithCluster(tt.cluster).
				WithDatacenter(tt.datacenter).
				Build()
			got := appendContainerRuntimeFlags(tt.flags, td)
			if !containsSameFlags(got, tt.want) {
				t.Errorf("appendContainerRuntimeFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContainerdFlags(t *testing.T) {
	tests := []struct {
		name                         string
		containerRuntimeOpts         *kubermaticv1.ContainerRuntimeOpts
		enableNonRootDeviceOwnership bool
		expected                     []string
	}{
		{
			name:                 "nil input returns empty slice",
			containerRuntimeOpts: nil,
			expected:             []string{},
		},
		{
			name: "empty registries map returns empty slice",
			containerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
				ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
					Registries: map[string]kubermaticv1.ContainerdRegistry{},
				},
			},
			expected: []string{},
		},
		{
			name: "single registry with single mirror",
			containerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
				ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
					Registries: map[string]kubermaticv1.ContainerdRegistry{
						"docker.io": {
							Mirrors: []string{"mirror1.docker.io"},
						},
					},
				},
			},
			expected: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
			},
		},
		{
			name: "single registry with multiple mirrors",
			containerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
				ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
					Registries: map[string]kubermaticv1.ContainerdRegistry{
						"docker.io": {
							Mirrors: []string{"mirror1.docker.io", "mirror2.docker.io"},
						},
					},
				},
			},
			expected: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
			},
		},
		{
			name: "multiple registries with mirrors are sorted",
			containerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
				ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
					Registries: map[string]kubermaticv1.ContainerdRegistry{
						"quay.io": {
							Mirrors: []string{"mirror.quay.io"},
						},
						"docker.io": {
							Mirrors: []string{"mirror1.docker.io", "mirror2.docker.io"},
						},
						"gcr.io": {
							Mirrors: []string{"mirror.gcr.io"},
						},
					},
				},
			},
			expected: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
				"-node-containerd-registry-mirrors=quay.io=mirror.quay.io",
			},
		},
		{
			name: "registry with empty mirrors is excluded from output",
			containerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
				ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
					Registries: map[string]kubermaticv1.ContainerdRegistry{
						"docker.io": {
							Mirrors: []string{},
						},
						"gcr.io": {
							Mirrors: []string{"mirror.gcr.io"},
						},
					},
				},
			},
			expected: []string{
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
			},
		},
		{
			name: "enable non-root device ownership in containerd",
			containerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
				ContainerdRegistryMirrors:    &kubermaticv1.ContainerRuntimeContainerd{},
				EnableNonRootDeviceOwnership: true,
			},
			expected: []string{
				"-device-ownership-from-security-context",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContainerdFlags(tt.containerRuntimeOpts)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getContainerdFlags() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContainerdFlags(t *testing.T) {
	tests := []struct {
		name         string
		nodeSettings *kubermaticv1.NodeSettings
		cluster      *kubermaticv1.Cluster
		want         []string
	}{
		{
			name:         "nil inputs return empty slice",
			nodeSettings: nil,
			cluster:      nil,
			want:         []string{},
		},
		{
			name:         "nil node settings with nil cluster return empty slice",
			nodeSettings: nil,
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: nil,
				},
			},
			want: []string{},
		},
		{
			name: "datacenter only - single registry with single mirror",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {
								Mirrors: []string{"dc-mirror.docker.io"},
							},
						},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=dc-mirror.docker.io",
			},
		},
		{
			name: "datacenter only - multiple registries with multiple mirrors",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {
								Mirrors: []string{"dc-mirror1.docker.io", "dc-mirror2.docker.io"},
							},
							"quay.io": {
								Mirrors: []string{"dc-mirror.quay.io"},
							},
						},
					},
				},
			},
			cluster: nil,
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=dc-mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=dc-mirror2.docker.io",
				"-node-containerd-registry-mirrors=quay.io=dc-mirror.quay.io",
			},
		},
		{
			name:         "cluster only - single registry with single mirror",
			nodeSettings: nil,
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"gcr.io": {
									Mirrors: []string{"cluster-mirror.gcr.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=gcr.io=cluster-mirror.gcr.io",
			},
		},
		{
			name:         "cluster only - multiple registries with multiple mirrors",
			nodeSettings: nil,
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"cluster-mirror1.docker.io", "cluster-mirror2.docker.io"},
								},
								"quay.io": {
									Mirrors: []string{"cluster-mirror.quay.io"},
								},
								"gcr.io": {
									Mirrors: []string{"cluster-mirror.gcr.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror2.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=cluster-mirror.gcr.io",
				"-node-containerd-registry-mirrors=quay.io=cluster-mirror.quay.io",
			},
		},
		{
			name: "cluster overrides datacenter - complete replacement",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {
								Mirrors: []string{"dc-mirror.docker.io"},
							},
							"quay.io": {
								Mirrors: []string{"dc-mirror.quay.io"},
							},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"gcr.io": {
									Mirrors: []string{"cluster-mirror.gcr.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=gcr.io=cluster-mirror.gcr.io",
			},
		},
		{
			name: "cluster overrides datacenter - same registry different mirrors",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {
								Mirrors: []string{"dc-mirror.docker.io"},
							},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"cluster-mirror1.docker.io", "cluster-mirror2.docker.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror2.docker.io",
			},
		},
		{
			name: "empty datacenter registries map with cluster config",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"cluster-mirror.docker.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror.docker.io",
			},
		},
		{
			name: "datacenter config with empty cluster registries map",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {
								Mirrors: []string{"dc-mirror.docker.io"},
							},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{},
						},
					},
				},
			},
			want: []string{},
		},
		{
			name: "registries with empty mirrors are excluded from both levels",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {
								Mirrors: []string{},
							},
							"quay.io": {
								Mirrors: []string{"dc-mirror.quay.io"},
							},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"gcr.io": {
									Mirrors: []string{},
								},
								"docker.io": {
									Mirrors: []string{"cluster-mirror.docker.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror.docker.io",
			},
		},
		{
			name: "datacenter with nil containerd config and cluster with valid config",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: nil,
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: &kubermaticv1.ContainerRuntimeOpts{
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"cluster-mirror.docker.io"},
								},
							},
						},
					},
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=cluster-mirror.docker.io",
			},
		},
		{
			name: "cluster with nil container runtime opts falls back to datacenter",
			nodeSettings: &kubermaticv1.NodeSettings{
				ContainerRuntimeOpts: kubermaticv1.ContainerRuntimeOpts{
					ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
						Registries: map[string]kubermaticv1.ContainerdRegistry{
							"docker.io": {Mirrors: []string{"dc-mirror.docker.io"}},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ContainerRuntimeOpts: nil,
				},
			},
			want: []string{
				"-node-containerd-registry-mirrors=docker.io=dc-mirror.docker.io",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containerdFlags(tt.nodeSettings, tt.cluster)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("containerdFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func containsSameFlags(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	gotMap := make(map[string]string)
	wantMap := make(map[string]string)

	for i := 0; i < len(got); i += 2 {
		if i+1 < len(got) {
			gotMap[got[i]] = got[i+1]
		}
	}

	for i := 0; i < len(want); i += 2 {
		if i+1 < len(want) {
			wantMap[want[i]] = want[i+1]
		}
	}

	return reflect.DeepEqual(gotMap, wantMap)
}
