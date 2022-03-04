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

package openapi

import (
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidatorFromCRD(t *testing.T) {
	const v1crdMultiversion = `
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: samples.kubermatic.k8c.io
spec:
  group: kubermatic.k8c.io
  names:
    kind: Sample
    listKind: SampleList
    plural: Samples
    singular: Sample
  scope: Cluster
  versions:
    - name: v1
      schema:
        openAPIV3Schema:
          description: ""
          properties:
            apiVersion:
              description: ""
              type: string
            kind:
              description: ""
              type: string
            metadata:
              type: object
            spec:
              description: spec
              properties:
                val1:
                  description: val1
                  type: string
                  enum:
                    - valid1
          type: object
      served: true
      storage: true
    - name: v2
      schema:
        openAPIV3Schema:
          description: ""
          properties:
            apiVersion:
              description: ""
              type: string
            kind:
              description: ""
              type: string
            metadata:
              type: object
            spec:
              description: spec
              properties:
                val1:
                  description: val1
                  type: string
                  enum:
                    - valid1
                    - valid2
          type: object
      served: true
      storage: true
`
	const v1crdSingleversion = `
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: samples.kubermatic.k8c.io
spec:
  group: kubermatic.k8c.io
  names:
    kind: Sample
    listKind: SampleList
    plural: Samples
    singular: Sample
  scope: Cluster
  versions:
    - name: v1
      schema:
        openAPIV3Schema:
          description: ""
          properties:
            apiVersion:
              description: ""
              type: string
            kind:
              description: ""
              type: string
            metadata:
              type: object
            spec:
              description: spec
              properties:
                val1:
                  description: val1
                  type: string
                  enum:
                    - valid1
          type: object
      served: true
      storage: true
`

	const v1beta1crd = `
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: samples.kubermatic.k8c.io
spec:
  group: kubermatic.k8c.io
  names:
    kind: Sample
    listKind: SampleList
    plural: Samples
    singular: Sample
  scope: Cluster
  validation:
    openAPIV3Schema:
      description: ""
      properties:
        apiVersion:
          description: ""
          type: string
        kind:
          description: ""
          type: string
        metadata:
          type: object
        spec:
          description: spec
          properties:
            val1:
              description: val1
              type: string
              enum:
                - valid1
    type: object
  versions:
    - name: v1
      served: true
      storage: true
`

	type sampleSpec struct {
		Val1 string `json:"val1"`
	}
	type sample struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec sampleSpec `json:"spec,omitempty"`
	}

	tt := map[string]struct {
		crd          string
		saKind       string
		saAPIVersion string
		saVal        string
		expValErrs   int
		expErr       bool
	}{
		"valid v1 sample": {
			v1crdMultiversion,
			"sample",
			"kubermatic.k8c.io/v1",
			"valid1",
			0,
			false,
		},
		"valid v1 sample singleVersion": {
			v1crdSingleversion,
			"sample",
			"kubermatic.k8c.io/v1",
			"valid1",
			0,
			false,
		},
		"invalid v1 sample": {
			v1crdMultiversion,
			"sample",
			"kubermatic.k8c.io/v1",
			"valid2",
			1,
			false,
		},
		"invalid v1 sample singleVersion": {
			v1crdSingleversion,
			"sample",
			"kubermatic.k8c.io/v1",
			"valid2",
			1,
			false,
		},
		"valid v2 sample": {
			v1crdMultiversion,
			"sample",
			"kubermatic.k8c.io/v2",
			"valid2",
			0,
			false,
		},
		"unsupported APIVersion": {
			v1crdMultiversion,
			"sample",
			"kubermatic.k8c.io/vInvalid",
			"valid1",
			0,
			true,
		},
		"valid global validation sample": {
			v1beta1crd,
			"sample",
			"kubermatic.k8c.io/v1",
			"valid1",
			0,
			false,
		},
		"invalid global validation sample": {
			v1beta1crd,
			"sample",
			"kubermatic.k8c.io/v1",
			"valid2",
			1,
			false,
		},
		"empty desired version": {
			v1crdMultiversion,
			"sample",
			"",
			"",
			0,
			true,
		},
		"unsupported crd version": {
			"apiVersion: apiextensions.k8s.io/vinvalid\nkind: CustomResourceDefinition",
			"sample",
			"kubermatic.k8c.io/v1",
			"valid1",
			0,
			true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			s := &sample{
				TypeMeta: metav1.TypeMeta{Kind: tc.saKind, APIVersion: tc.saAPIVersion},
				Spec:     sampleSpec{Val1: tc.saVal},
			}

			v, err := ValidatorFromCRD(strings.NewReader(tc.crd), s.GetObjectKind().GroupVersionKind().Version)
			res := validation.ValidateCustomResource(nil, s, v)

			if tc.expValErrs != len(res) {
				t.Errorf("Exp Errorlist length to be %d, got %d", tc.expValErrs, len(res))
			}

			if err != nil && !tc.expErr {
				t.Errorf("Exp err to be nil, but got %q", err)
			}
		})
	}
}

func TestValidatorForType(t *testing.T) {
	tt := map[string]struct {
		in           *metav1.TypeMeta
		expValidator bool
		expErr       bool
	}{
		"k8c.io crd": {
			&metav1.TypeMeta{Kind: "Cluster", APIVersion: "kubermatic.k8c.io/v1"},
			true,
			false,
		},
		"apps.k8c.io crd": {
			&metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"},
			true,
			false,
		},
		"invalid kind": {
			&metav1.TypeMeta{Kind: "Invalid", APIVersion: "kubermatic.k8c.io/v1"},
			false,
			true,
		},
		"invalid apiversion": {
			&metav1.TypeMeta{Kind: "Cluster", APIVersion: "Invalid"},
			false,
			true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			res, err := ValidatorForType(tc.in)

			if res != nil {
				if tc.expValidator && res.Schema == nil {
					t.Errorf("Root Schema is empty, when they should not be")
				}
			}

			if !tc.expErr && err != nil {
				t.Errorf("Exp error to be nil, but got %q", err)
			}
		})
	}
}
