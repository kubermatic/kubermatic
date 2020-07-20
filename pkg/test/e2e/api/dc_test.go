// +build e2e

package api

import (
	"reflect"
	"testing"

	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)

			dcs, err := apiRunner.ListDCForProvider(tc.provider)
			if err != nil {
				t.Fatalf("can not get dcs list due to error: %v", GetErrorResponse(err))
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
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)

			dc, err := apiRunner.GetDCForProvider(tc.provider, tc.dc)
			if err != nil {
				t.Fatalf("can not get dc due to error: %v", GetErrorResponse(err))
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
			seed: "prow-build-cluster",
			dc: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "created-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)

			dc, err := adminAPIRunner.CreateDC(tc.seed, tc.dc)
			if err != nil {
				t.Fatalf("can not create dc due to error: %v", GetErrorResponse(err))
			}

			if !reflect.DeepEqual(tc.dc, dc) {
				t.Fatalf("Expected create result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.dc.Metadata, *tc.dc.Spec, *tc.dc.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}

			// user can't create DC with the same name in the same seed
			_, err = adminAPIRunner.CreateDC(tc.seed, tc.dc)
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
			seed: "prow-build-cluster",
			dc: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "dc-to-delete",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)

			_, err = adminAPIRunner.CreateDC(tc.seed, tc.dc)
			if err != nil {
				t.Fatalf("can not create dc due to error: %v", GetErrorResponse(err))
			}

			err = adminAPIRunner.DeleteDC(tc.seed, tc.dc.Metadata.Name)
			if err != nil {
				t.Fatalf("can not delete dc due to error: %v", GetErrorResponse(err))
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
			seed: "prow-build-cluster",
			originalDC: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "to-update-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "prow-build-cluster",
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
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)

			dc, err := adminAPIRunner.CreateDC(tc.seed, tc.originalDC)
			if err != nil {
				t.Fatalf("can not create dc due to error: %v", GetErrorResponse(err))
			}

			if !reflect.DeepEqual(tc.originalDC, dc) {
				t.Fatalf("Expected create result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.originalDC.Metadata, *tc.originalDC.Spec, *tc.originalDC.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}

			updatedDC, err := adminAPIRunner.UpdateDC(tc.seed, tc.originalDC.Metadata.Name, tc.updatedDC)
			if err != nil {
				t.Fatalf("can not update dc due to error: %v", GetErrorResponse(err))
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
			seed: "prow-build-cluster",
			originalDC: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "to-patch-dc",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "prow-build-cluster",
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
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)

			dc, err := adminAPIRunner.CreateDC(tc.seed, tc.originalDC)
			if err != nil {
				t.Fatalf("can not create dc due to error: %v", GetErrorResponse(err))
			}

			if !reflect.DeepEqual(tc.originalDC, dc) {
				t.Fatalf("Expected create result: [meta: %+v, spec:%+v, node: %+v] is not equal to the one received: [meta: %+v, spec:%+v, node: %+v]",
					*tc.originalDC.Metadata, *tc.originalDC.Spec, *tc.originalDC.Spec.Node, *dc.Metadata, *dc.Spec, *dc.Spec.Node)
			}

			patchedDC, err := adminAPIRunner.PatchDC(tc.seed, tc.originalDC.Metadata.Name, tc.patch)
			if err != nil {
				t.Fatalf("can not patch dc due to error: %v", GetErrorResponse(err))
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
			name: "get DC do-ams3 for seed prow-build-cluster",
			seed: "prow-build-cluster",
			dc:   "do-ams3",
			expected: &models.Datacenter{
				Metadata: &models.DatacenterMeta{
					Name: "do-ams3",
				},
				Spec: &models.DatacenterSpec{
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)

			dc, err := apiRunner.GetDCForSeed(tc.seed, tc.dc)
			if err != nil {
				t.Fatalf("can not get dc due to error: %v", GetErrorResponse(err))
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
			name:            "list DCs for seed prow-build-cluster",
			seed:            "prow-build-cluster",
			expectedDCNames: []string{"alibaba-eu-central-1a", "aws-eu-central-1a", "azure-westeurope", "byo-kubernetes", "do-ams3", "do-fra1", "gcp-westeurope", "hetzner-nbg1", "kubevirt-europe-west3-c", "packet-ewr1", "syseleven-dbl1", "vsphere-ger"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)

			dcs, err := apiRunner.ListDCForSeed(tc.seed)
			if err != nil {
				t.Fatalf("can not get dcs list due to error: %v", GetErrorResponse(err))
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
					Seed:     "prow-build-cluster",
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)

			dc, err := apiRunner.GetDC(tc.dc)
			if err != nil {
				t.Fatalf("can not get dc due to error: %v", GetErrorResponse(err))
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)

			dcs, err := apiRunner.ListDC()
			if err != nil {
				t.Fatalf("can not get dcs list due to error: %v", GetErrorResponse(err))
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
