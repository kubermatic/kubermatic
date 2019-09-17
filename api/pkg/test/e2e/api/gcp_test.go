// +build e2e

package e2e

import (
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
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
			expectedCredentials: []string{"loodse"},
			datacenter:          "gcp-westeurope",
			resultList:          []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			credentialList, err := apiRunner.ListCredentials(tc.provider)
			if err != nil {
				t.Fatalf("can not get credential names for provider %s: %v", tc.provider, err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedCredentials, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedCredentials, credentialList)
			}

			zoneList, err := apiRunner.ListGCPZones(credentialList[0], tc.datacenter)
			if err != nil {
				t.Fatalf("can not get zones %v", err)
			}

			sort.Strings(zoneList)
			sort.Strings(tc.resultList)

			if !equality.Semantic.DeepEqual(tc.resultList, zoneList) {
				t.Fatalf("expected: %v, got %v", tc.resultList, zoneList)
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
			expectedCredentials: []string{"loodse"},
			zone:                "europe-west3-c",
			resultList:          []string{"pd-ssd", "pd-standard"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			credentialList, err := apiRunner.ListCredentials(tc.provider)
			if err != nil {
				t.Fatalf("can not get credential names for provider %s: %v", tc.provider, err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedCredentials, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedCredentials, credentialList)
			}

			diskTypeList, err := apiRunner.ListGCPDiskTypes(credentialList[0], tc.zone)
			if err != nil {
				t.Fatalf("can not get disk types %v", err)
			}

			sort.Strings(diskTypeList)
			sort.Strings(tc.resultList)

			if !equality.Semantic.DeepEqual(tc.resultList, diskTypeList) {
				t.Fatalf("expected: %v, got %v", tc.resultList, diskTypeList)
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
			expectedCredentials: []string{"loodse"},
			zone:                "europe-west3-c",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			credentialList, err := apiRunner.ListCredentials(tc.provider)
			if err != nil {
				t.Fatalf("can not get credential names for provider %s: %v", tc.provider, err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedCredentials, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedCredentials, credentialList)
			}

			_, err = apiRunner.ListGCPSizes(credentialList[0], tc.zone)
			if err != nil {
				t.Fatalf("can not get sizes %v", err)
			}

		})
	}
}
