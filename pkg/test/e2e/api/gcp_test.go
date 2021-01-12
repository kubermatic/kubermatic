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
	"sort"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGCPZones(t *testing.T) {
	tests := []struct {
		name                string
		provider            string
		expectedCredentials []string
		datacenter          string
		resultList          []string
	}{
		{
			name:                "test, get GCP zones",
			provider:            "gcp",
			expectedCredentials: []string{"e2e-gcp"},
			datacenter:          "gcp-westeurope",
			resultList:          []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"},
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
			credentialList, err := testClient.ListCredentials(tc.provider, "")
			if err != nil {
				t.Fatalf("failed to get credential names for provider %s: %v", tc.provider, err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedCredentials, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedCredentials, credentialList)
			}

			zoneList, err := testClient.ListGCPZones(credentialList[0], tc.datacenter)
			if err != nil {
				t.Fatalf("failed to get zones %v", err)
			}

			sort.Strings(zoneList)
			sort.Strings(tc.resultList)

			if !equality.Semantic.DeepEqual(tc.resultList, zoneList) {
				t.Fatalf("expected: %v, but got %v", tc.resultList, zoneList)
			}
		})
	}
}

func TestGCPDiskTypes(t *testing.T) {
	tests := []struct {
		name                string
		provider            string
		expectedCredentials []string
		zone                string
		resultList          []string
	}{
		{
			name:                "test, get GCP disk types",
			provider:            "gcp",
			expectedCredentials: []string{"e2e-gcp"},
			zone:                "europe-west3-c",
			resultList:          []string{"pd-ssd", "pd-standard"},
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
			credentialList, err := testClient.ListCredentials(tc.provider, "")
			if err != nil {
				t.Fatalf("failed to get credential names for provider %s: %v", tc.provider, err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedCredentials, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedCredentials, credentialList)
			}

			diskTypeList, err := testClient.ListGCPDiskTypes(credentialList[0], tc.zone)
			if err != nil {
				t.Fatalf("failed to get disk types: %v", err)
			}

			expectedDiskTypeList := sets.NewString(diskTypeList...)

			if !expectedDiskTypeList.HasAll(tc.resultList...) {
				t.Fatalf("expected: %v, but got %v", tc.resultList, diskTypeList)
			}
		})
	}
}

func TestGCPSizes(t *testing.T) {
	tests := []struct {
		name                string
		provider            string
		expectedCredentials []string
		zone                string
	}{
		{
			name:                "test, get GCP sizes",
			provider:            "gcp",
			expectedCredentials: []string{"e2e-gcp"},
			zone:                "europe-west3-c",
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
			credentialList, err := testClient.ListCredentials(tc.provider, "")
			if err != nil {
				t.Fatalf("failed to get credential names for provider %s: %v", tc.provider, err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedCredentials, credentialList) {
				t.Fatalf("expected: %v, but got %v", tc.expectedCredentials, credentialList)
			}

			_, err = testClient.ListGCPSizes(credentialList[0], tc.zone)
			if err != nil {
				t.Fatalf("failed to get sizes: %v", err)
			}
		})
	}
}
