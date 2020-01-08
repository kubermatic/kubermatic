// +build e2e

package api

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestListCredentials(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		expectedList []string
	}{
		{
			name:         "test, get DigitalOcean credential names",
			provider:     "digitalocean",
			expectedList: []string{"e2e-digitalocean"},
		},
		{
			name:         "test, get Azure credential names",
			provider:     "azure",
			expectedList: []string{"e2e-azure"},
		},
		{
			name:         "test, get OpenStack credential names",
			provider:     "openstack",
			expectedList: []string{"e2e-openstack"},
		},
		{
			name:         "test, get GCP credential names",
			provider:     "gcp",
			expectedList: []string{"e2e-gcp"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			credentialList, err := apiRunner.ListCredentials(tc.provider)
			if err != nil {
				t.Fatalf("can not get credential names for provider %s: %v", tc.provider, err)
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
			credentialName: "e2e-digitalocean",
			path:           "api/v1/providers/digitalocean/sizes",
			expectedCode:   http.StatusOK,
		},
		{
			name:           "test, get Azure VM sizes",
			credentialName: "e2e-azure",
			path:           "api/v1/providers/azure/sizes",
			location:       "westeurope",
			expectedCode:   http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
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
