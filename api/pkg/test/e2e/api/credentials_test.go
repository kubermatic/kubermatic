// +build e2e

package e2e

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestListDigitaloceanCredentials(t *testing.T) {
	tests := []struct {
		name         string
		expectedList []string
	}{
		{
			name:         "test, get DigitalOcean credential names",
			expectedList: []string{"digitalocean"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			credentialList, err := apiRunner.ListDigitaloceanCredentials()
			if err != nil {
				t.Fatalf("can not get credential names for DigitalOcean: %v", err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedList, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedList, credentialList)
			}
		})
	}
}

func TestListAzureCredentials(t *testing.T) {
	tests := []struct {
		name         string
		expectedList []string
	}{
		{
			name:         "test, get Azure credential names",
			expectedList: []string{"azure"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			credentialList, err := apiRunner.ListAzureCredentials()
			if err != nil {
				t.Fatalf("can not get credential names for Azure: %v", err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedList, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedList, credentialList)
			}
		})
	}
}

func TestListOpenstackCredentials(t *testing.T) {
	tests := []struct {
		name         string
		expectedList []string
	}{
		{
			name:         "test, get OpenStack credential names",
			expectedList: []string{"openstack"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			credentialList, err := apiRunner.ListOpenStackCredentials()
			if err != nil {
				t.Fatalf("can not get credential names for OpenStack: %v", err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedList, credentialList) {
				t.Fatalf("expected: %v, got %v", tc.expectedList, credentialList)
			}
		})
	}
}

func TestProviderEndpointsWithCredentials(t *testing.T) {
	tests := []struct {
		name           string
		credentialName string
		path           string
		location       string
		expectedCode   int
	}{
		{
			name:           "test, get DigitalOcean VM sizes",
			credentialName: "digitalocean",
			path:           "api/v1/providers/digitalocean/sizes",
			expectedCode:   http.StatusOK,
		},
		{
			name:           "test, get Azure VM sizes",
			credentialName: "azure",
			path:           "api/v1/providers/azure/sizes",
			location:       "westeurope",
			expectedCode:   http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			var u url.URL
			u.Host = getHost()
			u.Scheme = getScheme()
			u.Path = tc.path

			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				t.Fatalf("can not make GET call due error: %v", err)
			}

			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", masterToken))
			req.Header.Set("Credential", tc.credentialName)
			if len(tc.location) > 0 {
				req.Header.Set("Location", tc.location)
			}

			client := &http.Client{Timeout: time.Second * 10}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal("error reading response. ", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedCode {
				t.Fatalf("expected code %d, got %d", tc.expectedCode, resp.StatusCode)
			}

		})
	}
}
