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
	"context"
	"crypto/tls"
	"crypto/x509"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
)

func TestVSphereCA(t *testing.T) {
	// Set up vcsim simulator
	model := simulator.VPX()

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer model.Remove()

	// Create a TLS server by configuring TLS on the service before creating the server
	model.Service.TLS = new(tls.Config)
	server := model.Service.NewServer()
	defer server.Close()

	username := simulator.DefaultLogin.Username()
	password, _ := simulator.DefaultLogin.Password()

	serverURL := server.URL.String()
	serverHost := server.URL.Host

	dialer := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true}}
	conn, err := dialer.DialContext(context.Background(), "tcp", serverHost)
	if err != nil {
		t.Fatalf("failed to connect to get server cert: %v", err)
	}
	tlsConn := conn.(*tls.Conn)
	serverCert := tlsConn.ConnectionState().PeerCertificates[0]
	conn.Close()

	validCAPool := x509.NewCertPool()
	validCAPool.AddCert(serverCert)

	tests := []struct {
		name           string
		allowInsecure  bool
		caBundle       *x509.CertPool
		expectedError  bool
		errMsgContains string
	}{
		{
			name:          "succeed with AllowInsecure=true (skip cert validation)",
			allowInsecure: true,
			caBundle:      nil,
			expectedError: false,
		},
		{
			name:           "fail with fake certificate and AllowInsecure=false",
			allowInsecure:  false,
			caBundle:       certificates.NewFakeCABundle().CertPool(),
			expectedError:  true,
			errMsgContains: "certificate",
		},
		{
			name:          "succeed with valid server certificate",
			allowInsecure: false,
			caBundle:      validCAPool,
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dc := &kubermaticv1.DatacenterSpecVSphere{
				Datacenter:    "DC0",
				Endpoint:      strings.TrimSuffix(serverURL, "/sdk"),
				AllowInsecure: test.allowInsecure,
			}

			_, err := GetNetworks(context.Background(), dc, username, password, test.caBundle)

			if test.expectedError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), test.errMsgContains) {
					t.Fatalf("expected error message to contain %q, got: %v", test.errMsgContains, err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
