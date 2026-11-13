/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package flatcar

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/utils/ptr"
)

func TestResolve(t *testing.T) {
	conf := func(disabled *bool) *kubermaticv1.FlatcarUpdateOperatorSettings {
		return &kubermaticv1.FlatcarUpdateOperatorSettings{Disabled: disabled}
	}

	testCases := []struct {
		name         string
		cluster      *kubermaticv1.FlatcarUpdateOperatorSettings
		dc           *kubermaticv1.FlatcarUpdateOperatorSettings
		seed         *kubermaticv1.FlatcarUpdateOperatorSettings
		global       *kubermaticv1.FlatcarUpdateOperatorSettings
		wantDisabled bool
		wantSource   Source
	}{
		{
			name:       "nothing declared deploys the operator",
			wantSource: SourceNone,
		},
		{
			name:         "cluster true wins over admin false",
			cluster:      conf(ptr.To(true)),
			dc:           conf(ptr.To(false)),
			seed:         conf(ptr.To(false)),
			global:       conf(ptr.To(false)),
			wantDisabled: true,
			wantSource:   SourceCluster,
		},
		{
			name:       "cluster false wins over admin true",
			cluster:    conf(ptr.To(false)),
			dc:         conf(ptr.To(true)),
			seed:       conf(ptr.To(true)),
			global:     conf(ptr.To(true)),
			wantSource: SourceCluster,
		},
		{
			name:         "datacenter decides when the cluster is unset",
			dc:           conf(ptr.To(true)),
			seed:         conf(ptr.To(false)),
			global:       conf(ptr.To(false)),
			wantDisabled: true,
			wantSource:   SourceDatacenter,
		},
		{
			name:       "explicit false at the datacenter stops the walk",
			dc:         conf(ptr.To(false)),
			seed:       conf(ptr.To(true)),
			global:     conf(ptr.To(true)),
			wantSource: SourceDatacenter,
		},
		{
			name:         "seed decides when cluster and datacenter are unset",
			dc:           conf(nil),
			seed:         conf(ptr.To(true)),
			global:       conf(ptr.To(false)),
			wantDisabled: true,
			wantSource:   SourceSeed,
		},
		{
			name:       "explicit false at the seed stops the walk",
			seed:       conf(ptr.To(false)),
			global:     conf(ptr.To(true)),
			wantSource: SourceSeed,
		},
		{
			name:         "global decides when nothing narrower is declared",
			global:       conf(ptr.To(true)),
			wantDisabled: true,
			wantSource:   SourceGlobal,
		},
		{
			name:       "global false is a declaration",
			global:     conf(ptr.To(false)),
			wantSource: SourceGlobal,
		},
		{
			name:       "empty settings are the same as unset",
			cluster:    conf(nil),
			dc:         conf(nil),
			seed:       conf(nil),
			global:     conf(nil),
			wantSource: SourceNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolution := Resolve(tc.cluster, tc.dc, tc.seed, tc.global)
			if resolution.Disabled != tc.wantDisabled {
				t.Errorf("expected Disabled=%t, got %t", tc.wantDisabled, resolution.Disabled)
			}
			if resolution.Source != tc.wantSource {
				t.Errorf("expected Source=%q, got %q", tc.wantSource, resolution.Source)
			}
		})
	}
}
