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

package kubermatic

import (
	"testing"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestIngressCreatorKeepsAnnotations ensures that custom annotations
// are always kept when reconciling Ingresses.
func TestIngressCreatorKeepsAnnotations(t *testing.T) {
	cfg := &operatorv1alpha1.KubermaticConfiguration{}
	creatorGetter := IngressCreator(cfg)
	_, creator := creatorGetter()

	testcases := []struct {
		name    string
		ingress *networkingv1.Ingress
	}{
		{
			name:    "do not fail on nil map",
			ingress: &networkingv1.Ingress{},
		},
		{
			name: "keep existing annotations",
			ingress: &networkingv1.Ingress{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						"test": "value",
					},
				},
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			existingRules := test.ingress.DeepCopy().Annotations

			reconciled, err := creator(test.ingress)
			if err != nil {
				t.Fatalf("IngressCreator failed: %v", err)
			}

			for k, v := range existingRules {
				if reconciledValue := reconciled.Annotations[k]; reconciledValue != v {
					t.Errorf("Expected annotation %q with value %q, but got %q.", k, v, reconciledValue)
				}
			}
		})
	}
}
