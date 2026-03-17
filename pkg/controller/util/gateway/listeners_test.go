/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package gateway

import (
	"testing"

	"k8s.io/utils/ptr"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestMergeListeners(t *testing.T) {
	tests := []struct {
		name      string
		core      []gatewayapiv1.Listener
		existing  []gatewayapiv1.Listener
		want      []string                  // expected listener names in order
		wantPorts []gatewayapiv1.PortNumber // expected ports in same order as want
	}{
		{
			name: "core only, no existing",
			core: []gatewayapiv1.Listener{
				{Name: "http", Port: 80},
				{Name: "https", Port: 443},
			},
			existing:  nil,
			want:      []string{"http", "https"},
			wantPorts: []gatewayapiv1.PortNumber{80, 443},
		},
		{
			name: "preserves non-core from existing",
			core: []gatewayapiv1.Listener{
				{Name: "http", Port: 80},
				{Name: "https", Port: 443},
			},
			existing: []gatewayapiv1.Listener{
				{Name: "http", Port: 80},
				{Name: "https", Port: 443},
				{Name: "dex-example-com", Port: 8443},
			},
			want:      []string{"dex-example-com", "http", "https"},
			wantPorts: []gatewayapiv1.PortNumber{8443, 80, 443},
		},
		{
			name: "sorts alphabetically",
			core: []gatewayapiv1.Listener{
				{Name: "https", Port: 443},
				{Name: "http", Port: 80},
			},
			existing: []gatewayapiv1.Listener{
				{Name: "zebra-host", Port: 9443},
				{Name: "alpha-host", Port: 8443},
			},
			want:      []string{"alpha-host", "http", "https", "zebra-host"},
			wantPorts: []gatewayapiv1.PortNumber{8443, 80, 443, 9443},
		},
		{
			name: "uses core version of http/https, not existing",
			core: []gatewayapiv1.Listener{
				{Name: "http", Port: 8080},
				{Name: "https", Port: 8443},
			},
			existing: []gatewayapiv1.Listener{
				{Name: "http", Port: 80},
				{Name: "https", Port: 443},
			},
			want:      []string{"http", "https"},
			wantPorts: []gatewayapiv1.PortNumber{8080, 8443},
		},
		{
			name:      "empty core and existing",
			core:      nil,
			existing:  nil,
			want:      []string{},
			wantPorts: []gatewayapiv1.PortNumber{},
		},
		{
			name: "only non-core in existing",
			core: []gatewayapiv1.Listener{
				{Name: "http", Port: 80},
			},
			existing: []gatewayapiv1.Listener{
				{Name: "custom-listener", Port: 9000},
			},
			want:      []string{"custom-listener", "http"},
			wantPorts: []gatewayapiv1.PortNumber{9000, 80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeListeners(tt.core, tt.existing)

			if len(got) != len(tt.want) {
				t.Fatalf("MergeListeners() returned %d listeners, want %d", len(got), len(tt.want))
			}

			for i, name := range tt.want {
				if string(got[i].Name) != name {
					t.Errorf("MergeListeners()[%d].Name = %q, want %q", i, got[i].Name, name)
				}
				if got[i].Port != tt.wantPorts[i] {
					t.Errorf("MergeListeners()[%d].Port = %d, want %d", i, got[i].Port, tt.wantPorts[i])
				}
			}
		})
	}
}

func TestMergeListenersPreservesFields(t *testing.T) {
	core := []gatewayapiv1.Listener{
		{
			Name:     "http",
			Port:     80,
			Protocol: gatewayapiv1.HTTPProtocolType,
		},
	}

	const (
		dexListenerName = "dex-example-com"
		dexHostname     = "dex.example.com"
	)

	existing := []gatewayapiv1.Listener{
		{
			Name:     dexListenerName,
			Port:     443,
			Protocol: gatewayapiv1.HTTPSProtocolType,
			Hostname: ptr.To(gatewayapiv1.Hostname(dexHostname)),
			TLS: &gatewayapiv1.ListenerTLSConfig{
				Mode: ptr.To(gatewayapiv1.TLSModeTerminate),
			},
		},
	}

	got := MergeListeners(core, existing)
	if len(got) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(got))
	}

	var dexListener *gatewayapiv1.Listener
	for i := range got {
		if got[i].Name == dexListenerName {
			dexListener = &got[i]
			break
		}
	}

	if dexListener == nil {
		t.Fatalf("%q listener not found", dexListenerName)
	}

	if dexListener.Port != 443 {
		t.Errorf("expected port 443, got %d", dexListener.Port)
	}

	if dexListener.Hostname == nil || *dexListener.Hostname != dexHostname {
		t.Errorf("expected hostname %q, got %v", dexHostname, dexListener.Hostname)
	}

	if dexListener.TLS == nil {
		t.Error("expected TLS config to be preserved")
	}
}

func TestSortListenersByName(t *testing.T) {
	listeners := []gatewayapiv1.Listener{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "http"},
		{Name: "https"},
		{Name: "beta"},
	}

	SortListenersByName(listeners)

	expected := []string{"alpha", "beta", "http", "https", "zebra"}
	for i, name := range expected {
		if string(listeners[i].Name) != name {
			t.Errorf("SortListenersByName()[%d].Name = %q, want %q", i, listeners[i].Name, name)
		}
	}
}

func TestSortListenersByNameDeterministic(t *testing.T) {
	for run := 0; run < 10; run++ {
		listeners := []gatewayapiv1.Listener{
			{Name: "c"},
			{Name: "a"},
			{Name: "b"},
		}

		SortListenersByName(listeners)

		if listeners[0].Name != "a" || listeners[1].Name != "b" || listeners[2].Name != "c" {
			t.Errorf("run %d: non-deterministic sort result", run)
		}
	}
}
