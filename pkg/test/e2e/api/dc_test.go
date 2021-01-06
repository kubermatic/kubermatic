// +build e2e

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

package api

import (
	"context"
	"reflect"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

func TestListDCForProvider(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		expectedDCNames []string
	}{
		{
			name:            "list DCs for Digital Ocean",
			provider:        "digitalocean",
			expectedDCNames: []string{"do-ams3", "do-fra1"},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)

			dcs, err := testClient.ListDCForProvider(tc.provider)
			if err != nil {
				t.Fatalf("failed to get dcs list: %v", err)
			}

			var resultNames []string
			for _, dc := range dcs {
				resultNames = append(resultNames, dc.Metadata.Name)
			}

			if !reflect.DeepEqual(tc.expectedDCNames, resultNames) {
				t.Fatalf("Expected list result: %v is not equal to the one received: %v", tc.expectedDCNames, resultNames)
			}
		})
	}
}

func TestGetDCForProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		dc       string
		expected *models.Datacenter
	}{
		{
			name:     "get DC do-ams3 for provider DO",
			provider: "digitalocean",
			dc:       "do-ams3",
			expected: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "do-ams3",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Amsterdam",
					Country:  "NL",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ams3",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			apiRunner := utils.NewTestClient(masterToken, t)

			dc, err := apiRunner.GetDCForProvider(tc.provider, tc.dc)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			if !reflect.DeepEqual(tc.expected, dc) {
				t.Fatalf("Expected get result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.expected.Metadata, *tc.expected.Spec, *tc.expected.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}
		})
	}
}

func TestCreateDC(t *testing.T) {
	tests := []struct {
		name string
		seed string
		dc   *models.Datacenter
	}{
		{
			name: "create DC",
			seed: "kubermatic",
			dc: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "created-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Hamburg",
					Country:  "DE",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ham2",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			dc, err := adminTestClient.CreateDC(tc.seed, tc.dc)
			if err != nil {
				t.Fatalf("failed to create dc: %v", err)
			}

			if !reflect.DeepEqual(tc.dc, dc) {
				t.Fatalf("Expected create result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.dc.Metadata, *tc.dc.Spec, *tc.dc.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}

			_, err = adminTestClient.GetDCForSeed(tc.seed, tc.dc.Metadata.Name)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			// user can't create DC with the same name in the same seed
			_, err = adminTestClient.CreateDC(tc.seed, tc.dc)
			if err == nil {
				t.Fatalf("expected error, shouldn't be able to create DC with existing name in the same seed")
			}
		})
	}
}

func TestDeleteDC(t *testing.T) {
	tests := []struct {
		name string
		seed string
		dc   *models.Datacenter
	}{
		{
			name: "delete DC",
			seed: "kubermatic",
			dc: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "dc-to-delete",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Hamburg",
					Country:  "DE",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ham2",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			_, err = adminTestClient.CreateDC(tc.seed, tc.dc)
			if err != nil {
				t.Fatalf("failed to create dc: %v", err)
			}

			_, err = adminTestClient.GetDCForSeed(tc.seed, tc.dc.Metadata.Name)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			err = adminTestClient.DeleteDC(tc.seed, tc.dc.Metadata.Name)
			if err != nil {
				t.Fatalf("failed to delete dc: %v", err)
			}
		})
	}
}

func TestUpdateDC(t *testing.T) {
	tests := []struct {
		name       string
		seed       string
		originalDC *models.Datacenter
		updatedDC  *models.Datacenter
	}{
		{
			name: "update DC",
			seed: "kubermatic",
			originalDC: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "to-update-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Hamburg",
					Country:  "DE",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ham2",
					},
					Node: &models.NodeSettings{},
				},
			},
			updatedDC: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "updated-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "aws",
					Location: "Frankfurt",
					Country:  "DE",
					Aws: &models.DatacenterSpecAWS{
						Region: "fra2",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			dc, err := adminTestClient.CreateDC(tc.seed, tc.originalDC)
			if err != nil {
				t.Fatalf("failed to create dc: %v", err)
			}

			if !reflect.DeepEqual(tc.originalDC, dc) {
				t.Fatalf("Expected create result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.originalDC.Metadata, *tc.originalDC.Spec, *tc.originalDC.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}

			_, err = adminTestClient.GetDCForSeed(tc.seed, tc.originalDC.Metadata.Name)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			updatedDC, err := adminTestClient.UpdateDC(tc.seed, tc.originalDC.Metadata.Name, tc.updatedDC)
			if err != nil {
				t.Fatalf("failed to update dc: %v", err)
			}

			if !reflect.DeepEqual(tc.updatedDC, updatedDC) {
				t.Fatalf("Expected update result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.updatedDC.Metadata, *tc.updatedDC.Spec, *tc.updatedDC.Spec.Node, *updatedDC.Metadata, *updatedDC.Spec, *updatedDC.Spec.Node)
			}
		})
	}
}

func TestPatchDC(t *testing.T) {
	tests := []struct {
		name       string
		seed       string
		originalDC *models.Datacenter
		patch      string
		expectedDC *models.Datacenter
	}{
		{
			name: "patch DC",
			seed: "kubermatic",
			originalDC: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "to-patch-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Hamburg",
					Country:  "DE",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ham2",
					},
					Node: &models.NodeSettings{},
				},
			},
			patch: `{"metadata":{"name":"patched-dc"},"spec":{"location":"Frankfurt","aws":{"region":"fra2"},"digitalocean":null}}`,
			expectedDC: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "patched-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "aws",
					Location: "Frankfurt",
					Country:  "DE",
					Aws: &models.DatacenterSpecAWS{
						Region: "fra2",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			dc, err := adminTestClient.CreateDC(tc.seed, tc.originalDC)
			if err != nil {
				t.Fatalf("failed to create dc: %v", err)
			}

			if !reflect.DeepEqual(tc.originalDC, dc) {
				t.Fatalf("Expected create result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.originalDC.Metadata, *tc.originalDC.Spec, *tc.originalDC.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}

			_, err = adminTestClient.GetDCForSeed(tc.seed, tc.originalDC.Metadata.Name)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			patchedDC, err := adminTestClient.PatchDC(tc.seed, tc.originalDC.Metadata.Name, tc.patch)
			if err != nil {
				t.Fatalf("failed to patch dc: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedDC, patchedDC) {
				t.Fatalf("Expected patch result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.expectedDC.Metadata, *tc.expectedDC.Spec, *tc.expectedDC.Spec.Node, *patchedDC.Metadata, *patchedDC.Spec, *patchedDC.Spec.Node)
			}
		})
	}
}

func TestGetDCForSeed(t *testing.T) {
	tests := []struct {
		name     string
		seed     string
		dc       string
		expected *models.Datacenter
	}{
		{
			name: "get DC do-ams3 for seed kubermatic",
			seed: "kubermatic",
			dc:   "do-ams3",
			expected: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "do-ams3",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Amsterdam",
					Country:  "NL",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ams3",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)

			dc, err := testClient.GetDCForSeed(tc.seed, tc.dc)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			if !reflect.DeepEqual(tc.expected, dc) {
				t.Fatalf("Expected get result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.expected.Metadata, *tc.expected.Spec, *tc.expected.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}
		})
	}
}

func TestListDCForSeed(t *testing.T) {
	tests := []struct {
		name            string
		seed            string
		expectedDCNames []string
	}{
		{
			name:            "list DCs for seed kubermatic",
			seed:            "kubermatic",
			expectedDCNames: []string{"alibaba-eu-central-1a", "aws-eu-central-1a", "azure-westeurope", "byo-kubernetes", "do-ams3", "do-fra1", "gcp-westeurope", "hetzner-nbg1", "kubevirt-europe-west3-c", "packet-ewr1", "syseleven-dbl1", "vsphere-ger"},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)

			dcs, err := testClient.ListDCForSeed(tc.seed)
			if err != nil {
				t.Fatalf("failed to get dcs list: %v", err)
			}

			resultNames := make(map[string]bool)
			for _, dc := range dcs {
				resultNames[dc.Metadata.Name] = true
			}

			for _, dcName := range tc.expectedDCNames {
				if _, ok := resultNames[dcName]; !ok {
					t.Fatalf("Expected list result: %v does not contail all expected dcs: %v", resultNames, tc.expectedDCNames)
				}
			}
		})
	}
}

func TestGetDC(t *testing.T) {
	tests := []struct {
		name     string
		dc       string
		expected *models.Datacenter
	}{
		{
			name: "get DC do-ams3",
			dc:   "do-ams3",
			expected: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "do-ams3",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "kubermatic",
					Provider: "digitalocean",
					Location: "Amsterdam",
					Country:  "NL",
					Digitalocean: &models.DatacenterSpecDigitalocean{
						Region: "ams3",
					},
					Node: &models.NodeSettings{},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)

			dc, err := testClient.GetDC(tc.dc)
			if err != nil {
				t.Fatalf("failed to get dc: %v", err)
			}

			if !reflect.DeepEqual(tc.expected, dc) {
				t.Fatalf("Expected get result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.expected.Metadata, *tc.expected.Spec, *tc.expected.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}
		})
	}
}

func TestListDC(t *testing.T) {
	tests := []struct {
		name            string
		expectedDCNames []string
	}{
		{
			name:            "list DCs",
			expectedDCNames: []string{"alibaba-eu-central-1a", "aws-eu-central-1a", "azure-westeurope", "byo-kubernetes", "do-ams3", "do-fra1", "gcp-westeurope", "hetzner-nbg1", "kubevirt-europe-west3-c", "packet-ewr1", "syseleven-dbl1", "vsphere-ger"},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)

			dcs, err := testClient.ListDC()
			if err != nil {
				t.Fatalf("failed to get dcs list: %v", err)
			}

			resultNames := make(map[string]bool)
			for _, dc := range dcs {
				resultNames[dc.Metadata.Name] = true
			}

			for _, dcName := range tc.expectedDCNames {
				if _, ok := resultNames[dcName]; !ok {
					t.Fatalf("Expected list result: %v does not contail all expected dcs: %v", resultNames, tc.expectedDCNames)
				}
			}
		})
	}
}
