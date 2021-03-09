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

package provider_test

import (
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileBinding(t *testing.T) {
	tests := []struct {
		name               string
		userInfo           *provider.UserInfo
		datacenterName     string
		expectedError      bool
		expectedDatacenter *kubermaticv1.Datacenter
	}{
		{
			name:          "scenario 1: regular user can't get datacenter with restricted domain",
			expectedError: true,
			userInfo: &provider.UserInfo{
				Email:   "test@test.com",
				Group:   "",
				IsAdmin: false,
			},
			datacenterName: "restricted-fake-dc",
		},
		{
			name: "scenario 2: admin should get restricted datacenter with any domain",
			userInfo: &provider.UserInfo{
				Email:   "test@test.com",
				Group:   "",
				IsAdmin: true,
			},
			datacenterName: "restricted-fake-dc",
			expectedDatacenter: &kubermaticv1.Datacenter{
				Country:  "NL",
				Location: "Amsterdam",
				Spec: kubermaticv1.DatacenterSpec{
					Fake:                &kubermaticv1.DatacenterSpecFake{},
					RequiredEmailDomain: "example.com",
				},
			},
		},
		{
			name: "scenario 3: user can get datacenter without restriction",
			userInfo: &provider.UserInfo{
				Email:   "test@test.com",
				Group:   "",
				IsAdmin: false,
			},
			datacenterName: "fake-dc",
			expectedDatacenter: &kubermaticv1.Datacenter{
				Location: "Henriks basement",
				Country:  "Germany",
				Spec: kubermaticv1.DatacenterSpec{
					Fake: &kubermaticv1.DatacenterSpecFake{},
				},
			},
		},
		{
			name: "scenario 4: user can get restricted datacenter with the matching domain",
			userInfo: &provider.UserInfo{
				Email:   "test@example.com",
				Group:   "",
				IsAdmin: false,
			},
			datacenterName: "node-dc",
			expectedDatacenter: &kubermaticv1.Datacenter{
				Location: "Santiago",
				Country:  "Chile",
				Spec: kubermaticv1.DatacenterSpec{
					Fake:                 &kubermaticv1.DatacenterSpecFake{},
					RequiredEmailDomains: []string{"abc.com", "example.com", "cde.org"},
				},
				Node: &kubermaticv1.NodeSettings{
					ProxySettings: kubermaticv1.ProxySettings{
						HTTPProxy: kubermaticv1.NewProxyValue("HTTPProxy"),
					},
					InsecureRegistries: []string{"incsecure-registry"},
					RegistryMirrors:    []string{"http://127.0.0.1:5001"},
					PauseImage:         "pause-image",
					HyperkubeImage:     "hyperkube-image",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, datacenter, err := provider.DatacenterFromSeedMap(test.userInfo, buildSeeds(), test.datacenterName)
			if !test.expectedError {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(datacenter, test.expectedDatacenter) {
					t.Fatalf("expected %v got %v", test.expectedDatacenter, datacenter)
				}
			}
			if test.expectedError && err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func genTestUSCentalSeed() *kubermaticv1.Seed {
	return &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "us-central1",
		},
		Spec: kubermaticv1.SeedSpec{
			Location: "us-central",
			Country:  "US",
			Datacenters: map[string]kubermaticv1.Datacenter{
				"private-do1": {
					Country:  "NL",
					Location: "US ",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
						EnforcePodSecurityPolicy: true,
					},
					Node: &kubermaticv1.NodeSettings{
						PauseImage: "image-pause",
					},
				},
				"regular-do1": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
				"restricted-fake-dc": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                &kubermaticv1.DatacenterSpecFake{},
						RequiredEmailDomain: "example.com",
					},
				},
				"restricted-fake-dc2": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                 &kubermaticv1.DatacenterSpecFake{},
						RequiredEmailDomains: []string{"abc.com", "example.com", "cde.org"},
					},
				},
			},
		}}
}

func genTestEuropeWestSeed() *kubermaticv1.Seed {
	return &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "europe-west",
		},
		Spec: kubermaticv1.SeedSpec{
			Location: "europe-west",
			Country:  "US",
			Datacenters: map[string]kubermaticv1.Datacenter{
				"fake-dc": {
					Location: "Henriks basement",
					Country:  "Germany",
					Spec: kubermaticv1.DatacenterSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
				"audited-dc": {
					Location: "Finanzamt Castle",
					Country:  "Germany",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                &kubermaticv1.DatacenterSpecFake{},
						EnforceAuditLogging: true,
					},
				},
				"psp-dc": {
					Location: "Alexandria",
					Country:  "Egypt",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                     &kubermaticv1.DatacenterSpecFake{},
						EnforcePodSecurityPolicy: true,
					},
				},
				"node-dc": {
					Location: "Santiago",
					Country:  "Chile",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                 &kubermaticv1.DatacenterSpecFake{},
						RequiredEmailDomains: []string{"abc.com", "example.com", "cde.org"},
					},
					Node: &kubermaticv1.NodeSettings{
						ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: kubermaticv1.NewProxyValue("HTTPProxy"),
						},
						InsecureRegistries: []string{"incsecure-registry"},
						RegistryMirrors:    []string{"http://127.0.0.1:5001"},
						PauseImage:         "pause-image",
						HyperkubeImage:     "hyperkube-image",
					},
				},
			},
		}}
}

func buildSeeds() provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		seeds := make(map[string]*kubermaticv1.Seed)
		seeds["us-central1"] = genTestUSCentalSeed()
		seeds["europe-west"] = genTestEuropeWestSeed()
		return seeds, nil
	}
}
