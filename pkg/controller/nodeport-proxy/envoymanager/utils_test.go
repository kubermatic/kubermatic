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

package envoymanager

import (
	"testing"

	"github.com/go-test/deep"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractExposeType(t *testing.T) {
	var testcases = []struct {
		name            string
		svc             *corev1.Service
		wantExposeTypes []ExposeType
	}{
		{
			name:            "Legacy value",
			svc:             makeService("", "true"),
			wantExposeTypes: []ExposeType{NodePortType},
		},
		{
			name:            "New value",
			svc:             makeService("", "NodePort"),
			wantExposeTypes: []ExposeType{NodePortType},
		},
		{
			name:            "No value",
			svc:             makeService("", ""),
			wantExposeTypes: nil,
		},
		{
			name:            "Both HTTP2 Connet and SNI",
			svc:             makeService("", "HTTP2Connect, SNI"),
			wantExposeTypes: []ExposeType{HTTP2ConnectType, SNIType},
		},
		{
			name:            "Both HTTP2 Connet and SNI #2",
			svc:             makeService("", "HTTP2Connect,SNI"),
			wantExposeTypes: []ExposeType{HTTP2ConnectType, SNIType},
		},
		{
			name:            "Malformed value",
			svc:             makeService("", "HTTP2Connect SNI"),
			wantExposeTypes: nil,
		},
		{
			name:            "Malformed value #2",
			svc:             makeService("", "True"),
			wantExposeTypes: nil,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			e := extractExposeTypes(tt.svc, DefaultExposeAnnotationKey)

			if diff := deep.Equal(tt.wantExposeTypes, e); diff != nil {
				t.Errorf("Got export types. Diff to expected: %v", diff)
			}
		})
	}
}

func TestExtractPortHostMappingFromService(t *testing.T) {
	var testcases = []struct {
		name        string
		service     *corev1.Service
		wantMapping portHostMapping
		wantErr     bool
	}{
		{
			name:        "Default port only",
			service:     makeService(`{"": "host.com"}`, ""),
			wantMapping: portHostMapping{"": "host.com"},
		},
		{
			name:        "Multiple port mappings",
			service:     makeService(`{"port-a": "admin.host.com", "port-b": "host.com"}`, ""),
			wantMapping: portHostMapping{"port-a": "admin.host.com", "port-b": "host.com"},
		},
		{
			name:        "Missing annotations",
			service:     &corev1.Service{},
			wantMapping: portHostMapping{},
		},
		{
			name: "Missing port host mapping annotation",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			wantMapping: portHostMapping{},
		},
		{
			name:        "Annotation contains malformed json",
			service:     makeService("{sdf: a}", ""),
			wantErr:     true,
			wantMapping: portHostMapping{},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			p, err := portHostMappingFromService(tt.service)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr: %t, got %v", tt.wantErr, err)
			}

			if diff := deep.Equal(tt.wantMapping, p); diff != nil {
				t.Errorf("Got unexpected host port mapping. Diff to expected: %v", diff)
			}
		})
	}
}

func TestPortHostMappingValidate(t *testing.T) {
	var testcases = []struct {
		name    string
		svc     *corev1.Service
		mapping portHostMapping
		wantErr bool
	}{
		{
			name: "Default port",
			svc: makeService("", "",
				corev1.ServicePort{Name: "", Protocol: corev1.ProtocolTCP}),
			mapping: portHostMapping{"": "host.com"},
		},
		{
			name: "Multiple ports",
			svc: makeService("", "",
				corev1.ServicePort{Name: "port-a", Protocol: corev1.ProtocolTCP},
				corev1.ServicePort{Name: "port-b", Protocol: corev1.ProtocolTCP},
				corev1.ServicePort{Name: "port-c", Protocol: corev1.ProtocolTCP}),
			mapping: portHostMapping{"port-a": "admin.host.com", "port-b": "host.com"},
		},
		{
			name: "Duplicated ports",
			svc: makeService("", "",
				corev1.ServicePort{Name: "port-a", Protocol: corev1.ProtocolTCP},
				corev1.ServicePort{Name: "port-b", Protocol: corev1.ProtocolTCP},
				corev1.ServicePort{Name: "port-c", Protocol: corev1.ProtocolTCP}),
			mapping: portHostMapping{"port-a": "admin.host.com", "port-b": "host.com", "port-c": "host.com"},
			wantErr: true,
		},
		{
			name: "Mapping reference missing ports",
			svc: makeService("", "",
				corev1.ServicePort{Name: "port-a", Protocol: corev1.ProtocolTCP}),
			mapping: portHostMapping{"port-a": "admin.host.com", "port-b": "host.com"},
			wantErr: true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mapping.validate(tt.svc); (err != nil) != tt.wantErr {
				t.Fatalf("wantErr: %t, got: %v", tt.wantErr, err)
			}
		})
	}
}

func makeService(portHostMappingVal string, exposeTypeVal string, ports ...corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
			Annotations: map[string]string{
				PortHostMappingAnnotationKey: portHostMappingVal,
				DefaultExposeAnnotationKey:   exposeTypeVal,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	}
}
