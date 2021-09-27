//go:build integration

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"crypto/x509"
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources/certificates"
)

func TestVSphereCA(t *testing.T) {
	tests := []struct {
		name           string
		expectedError  bool
		caBundle       *x509.CertPool
		errMsgContains string
	}{
		{
			name:           "fail accessing vSphere with fake certificate",
			expectedError:  true,
			caBundle:       certificates.NewFakeCABundle().CertPool(),
			errMsgContains: "certificate",
		},
		{
			name:          "succeed accessing vSphere with root/host certificate",
			expectedError: false,
			caBundle:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GetNetworks(getTestDC(), vSphereUsername, vSpherePassword, test.caBundle)
			if test.expectedError {
				if err == nil {
					t.Fatal("expected err, got nil")
				}
				if !strings.Contains(err.Error(), test.errMsgContains) {
					t.Fatalf("expected err msg %q to contain %q", err.Error(), test.errMsgContains)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
