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

package seed

import (
	"context"
	"errors"
	"testing"

	logrtesting "github.com/go-logr/logr/testing"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		valid       bool
		wantSeed    kubermaticv1.Seed
		wantAllowed bool
	}{
		{
			name: "Delete seed succeess",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Seed",
					},
					Name:      "Seed1",
					Namespace: "Kubermatic",
				},
			},
			valid:       true,
			wantSeed:    kubermaticv1.Seed{ObjectMeta: metav1.ObjectMeta{Name: "Seed1", Namespace: "Kubermatic"}},
			wantAllowed: true,
		},
		{
			name: "Create seed succeess",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Seed",
					},
					Name:      "Seed1",
					Namespace: "Kubermatic",
					Object: runtime.RawExtension{
						Raw: []byte(`{"apiVersion":"kubermatic.k8s.io/v1","kind":"Seed","metadata":{"name":"Seed","namespace":"Kubermatic"}}`),
					},
				},
			},
			valid: true,
			wantSeed: kubermaticv1.Seed{
				TypeMeta:   metav1.TypeMeta{Kind: "Seed", APIVersion: "kubermatic.k8s.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "Seed", Namespace: "Kubermatic"},
			},
			wantAllowed: true,
		},
		{
			name: "Update seed succeess",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Seed",
					},
					Name:      "Seed1",
					Namespace: "Kubermatic",
					Object: runtime.RawExtension{
						Raw: []byte(`{"apiVersion":"kubermatic.k8s.io/v1","kind":"Seed","metadata":{"name":"Seed","namespace":"Kubermatic"}}`),
					},
				},
			},
			valid: true,
			wantSeed: kubermaticv1.Seed{
				TypeMeta:   metav1.TypeMeta{Kind: "Seed", APIVersion: "kubermatic.k8s.io/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "Seed", Namespace: "Kubermatic"},
			},
			wantAllowed: true,
		},
		{
			name: "Unsupported operation",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Connect,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Seed",
					},
					Name:      "Seed1",
					Namespace: "Kubermatic",
				},
			},
			valid:       true,
			wantAllowed: false,
		},
		{
			name: "Malformed payload",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Seed",
					},
					Name:      "Seed1",
					Namespace: "Kubermatic",
					Object: runtime.RawExtension{
						Raw: []byte(`{"Seed`),
					},
				},
			},
			valid:       true,
			wantAllowed: false,
		},
		{
			name: "Validation function failed",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Seed",
					},
					Name:      "Seed1",
					Namespace: "Kubermatic",
				},
			},
			valid:       false,
			wantSeed:    kubermaticv1.Seed{ObjectMeta: metav1.ObjectMeta{Name: "Seed1", Namespace: "Kubermatic"}},
			wantAllowed: false,
		},
	}
	for _, tt := range tests {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}
		handler := seedAdmissionHandler{
			log:     &logrtesting.NullLogger{},
			decoder: d,
			validateFunc: func(_ context.Context, s *kubermaticv1.Seed, op admissionv1.Operation) error {
				if !equality.Semantic.DeepEqual(*s, tt.wantSeed) {
					t.Errorf("expected seed %+v but got %+v", s, tt.wantSeed)
				}
				if tt.valid {
					return nil
				}
				return errors.New("validation failed")
			},
		}
		t.Run(tt.name, func(t *testing.T) {
			if res := handler.Handle(context.TODO(), tt.req); res.Allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
				t.Logf("Response: %v", res)
			}
		})
	}
}
