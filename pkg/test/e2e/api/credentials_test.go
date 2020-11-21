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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestListCredentials(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		datacenter   string
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
		{
			name:         "test, get GCP credential names for the specific datacenter",
			provider:     "gcp",
			datacenter:   "gcp-westeurope",
			expectedList: []string{"e2e-gcp", "e2e-gcp-datacenter"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			credentialList, err := apiRunner.ListCredentials(tc.provider, tc.datacenter)
			if err != nil {
				t.Fatalf("failed to get credential names for provider %s: %v", tc.provider, err)
			}
			sort.Strings(tc.expectedList)
			sort.Strings(credentialList)
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

	endpoint, err := getAPIEndpoint()
	if err != nil {
		t.Fatalf("failed to determine Kubermatic API endpoint: %v", err)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			u.Path = tc.path

			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", masterToken))
			req.Header.Set("Credential", tc.credentialName)
			if len(tc.location) > 0 {
				req.Header.Set("Location", tc.location)
			}

			client := &http.Client{Timeout: time.Second * 5}
			backoff := wait.Backoff{
				// With those settings the cumulative sleep duration is ~ 8s
				// when all attempts are made.
				Duration: time.Second,
				Factor:   1.5,
				Steps:    4,
			}
			if err := wait.ExponentialBackoff(backoff, func() (bool, error) {
				resp, err := client.Do(req)
				if err != nil {
					if uerr, ok := err.(*url.Error); ok && (uerr.Temporary() || uerr.Timeout()) {
						return false, nil
					}
					t.Logf("unrecoverable error reading response: %v", err)
					return false, err
				}
				defer resp.Body.Close()

				if resp.StatusCode == tc.expectedCode {
					t.Logf("expected code %d, but got %d", tc.expectedCode, resp.StatusCode)
					return true, nil
				}
				// 5xx return codes may be associated to recoverable
				// conditions, with the exception of 501 (Not implemented)
				if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
					t.Logf("unexpected HTTP status %s", resp.Status)
					return false, nil
				}
				return false, fmt.Errorf("got response with an unexpected status code: %d", resp.StatusCode)

			}); err != nil {
				t.Errorf("failed to get expected response [%d] from %q endpoint: %v", tc.expectedCode, tc.location, err)
			}

		})
	}
}
