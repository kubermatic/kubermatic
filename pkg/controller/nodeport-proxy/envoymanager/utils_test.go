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
	"time"

	"github.com/go-test/deep"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestExtractExposeType(t *testing.T) {
	var testcases = []struct {
		name            string
		svc             *corev1.Service
		wantExposeTypes ExposeTypes
	}{
		{
			name:            "Legacy value",
			svc:             makeService("", "true"),
			wantExposeTypes: NewExposeTypes(NodePortType),
		},
		{
			name:            "New value",
			svc:             makeService("", "NodePort"),
			wantExposeTypes: NewExposeTypes(NodePortType),
		},
		{
			name:            "No value",
			svc:             makeService("", ""),
			wantExposeTypes: NewExposeTypes(),
		},
		{
			name:            "Both Tunneling and SNI",
			svc:             makeService("", "Tunneling, SNI"),
			wantExposeTypes: NewExposeTypes(TunnelingType, SNIType),
		},
		{
			name:            "Both Tunneling and SNI #2",
			svc:             makeService("", "Tunneling,SNI"),
			wantExposeTypes: NewExposeTypes(TunnelingType, SNIType),
		},
		{
			name:            "Malformed value",
			svc:             makeService("", "Tunneling SNI"),
			wantExposeTypes: NewExposeTypes(),
		},
		{
			name:            "Malformed value #2",
			svc:             makeService("", "True"),
			wantExposeTypes: NewExposeTypes(),
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
			p, err := portHostMappingFromAnnotation(tt.service)
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

func TestSortServicesByCreationTimestamp(t *testing.T) {
	mkSvc := func(uid string, creationTimestamp time.Time) corev1.Service {
		return corev1.Service{ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			Namespace:         "test",
			UID:               types.UID(uid),
			CreationTimestamp: metav1.NewTime(creationTimestamp),
		}}
	}
	timeRef := time.Date(2020, time.December, 0, 0, 0, 0, 0, time.UTC)
	var testcases = []struct {
		name             string
		items            []corev1.Service
		wantOrderedItems []corev1.Service
	}{
		{
			name:             "UID used to break ties",
			items:            []corev1.Service{mkSvc("b", timeRef), mkSvc("a", timeRef)},
			wantOrderedItems: []corev1.Service{mkSvc("a", timeRef), mkSvc("b", timeRef)},
		},
		{
			name: "Creation timestamp used as primary order criteria",
			items: []corev1.Service{
				mkSvc("a", timeRef.Add(2*time.Second)),
				mkSvc("b", timeRef),
				mkSvc("c", timeRef.Add(1*time.Second)),
				mkSvc("d", timeRef),
			},
			wantOrderedItems: []corev1.Service{
				mkSvc("b", timeRef),
				mkSvc("d", timeRef),
				mkSvc("c", timeRef.Add(1*time.Second)),
				mkSvc("a", timeRef.Add(2*time.Second)),
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			SortServicesByCreationTimestamp(tt.items)
			if diff := deep.Equal(tt.wantOrderedItems, tt.items); diff != nil {
				t.Errorf("Unexpected order of items. Diff to expected: %v", diff)
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
