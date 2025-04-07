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
