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
	ctx := context.Background()
	sim := vSphereTLSSimulator{t: t}
	sim.setUp()
	defer sim.tearDown()

	serverCert := sim.getServerCertificate()
	validCAPool := x509.NewCertPool()
	validCAPool.AddCert(serverCert)

	tests := []struct {
		name           string
		allowInsecure  bool
		caBundle       *x509.CertPool
		wantErr        bool
		errMsgContains string
	}{
		{
			name:          "succeed with AllowInsecure=true",
			allowInsecure: true,
			caBundle:      nil,
			wantErr:       false,
		},
		{
			name:           "fail with fake certificate",
			allowInsecure:  false,
			caBundle:       certificates.NewFakeCABundle().CertPool(),
			wantErr:        true,
			errMsgContains: "certificate",
		},
		{
			name:          "succeed with valid server certificate",
			allowInsecure: false,
			caBundle:      validCAPool,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &kubermaticv1.DatacenterSpecVSphere{
				Datacenter:    "DC0",
				Endpoint:      strings.TrimSuffix(sim.server.URL.String(), "/sdk"),
				AllowInsecure: tt.allowInsecure,
			}

			_, err := GetNetworks(ctx, dc, sim.username, sim.password, tt.caBundle)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsgContains) {
					t.Fatalf("expected error containing %q, got: %v", tt.errMsgContains, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

type vSphereTLSSimulator struct {
	t        *testing.T
	model    *simulator.Model
	server   *simulator.Server
	username string
	password string
}

func (v *vSphereTLSSimulator) setUp() {
	v.model = simulator.VPX()
	if err := v.model.Create(); err != nil {
		v.t.Fatal(err)
	}

	v.model.Service.TLS = new(tls.Config)
	v.server = v.model.Service.NewServer()

	v.username = simulator.DefaultLogin.Username()
	v.password, _ = simulator.DefaultLogin.Password()
}

func (v *vSphereTLSSimulator) tearDown() {
	v.server.Close()
	v.model.Remove()
}

func (v *vSphereTLSSimulator) getServerCertificate() *x509.Certificate {
	dialer := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true}}
	conn, err := dialer.DialContext(context.Background(), "tcp", v.server.URL.Host)
	if err != nil {
		v.t.Fatalf("failed to connect for server cert: %v", err)
	}
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	return tlsConn.ConnectionState().PeerCertificates[0]
}
