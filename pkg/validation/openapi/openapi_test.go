/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"strings"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd"

	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
)

func TestValidatorFromCRD(t *testing.T) {
	const v1crdMultiversion = `
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
			crd:          v1crdMultiversion,
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/v1",
			saVal:        "valid1",
			expValErrs:   0,
			expErr:       false,
		},
		"valid v1 sample singleVersion": {
			crd:          v1crdSingleversion,
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/v1",
			saVal:        "valid1",
			expValErrs:   0,
			expErr:       false,
		},
		"invalid v1 sample": {
			crd:          v1crdMultiversion,
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/v1",
			saVal:        "valid2",
			expValErrs:   1,
			expErr:       false,
		},
		"invalid v1 sample singleVersion": {
			crd:          v1crdSingleversion,
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/v1",
			saVal:        "valid2",
			expValErrs:   1,
			expErr:       false,
		},
		"valid v2 sample": {
			crd:          v1crdMultiversion,
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/v2",
			saVal:        "valid2",
			expValErrs:   0,
			expErr:       false,
		},
		"unsupported APIVersion": {
			crd:          v1crdMultiversion,
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/vInvalid",
			saVal:        "valid1",
			expValErrs:   0,
			expErr:       true,
		},
		"empty desired version": {
			crd:          v1crdMultiversion,
			saKind:       "sample",
			saAPIVersion: "",
			saVal:        "",
			expValErrs:   0,
			expErr:       true,
		},
		"unsupported crd version": {
			crd:          "apiVersion: apiextensions.k8s.io/vinvalid\nkind: CustomResourceDefinition",
			saKind:       "sample",
			saAPIVersion: "kubermatic.k8c.io/v1",
			saVal:        "valid1",
			expValErrs:   0,
			expErr:       true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			s := &sample{
				TypeMeta: metav1.TypeMeta{Kind: tc.saKind, APIVersion: tc.saAPIVersion},
				Spec:     sampleSpec{Val1: tc.saVal},
			}

			u := &unstructured.Unstructured{}
			dec := yaml.NewYAMLOrJSONDecoder(strings.NewReader(tc.crd), 1024)
			if err := dec.Decode(u); err != nil {
				t.Fatal(err)
			}

			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), crd); err != nil {
				t.Fatal(err)
			}

			v, err := NewValidatorForCRD(crd, s.GetObjectKind().GroupVersionKind().Version)
			if err != nil && !tc.expErr {
				t.Fatalf("Exp err to be nil, but got %q", err)
			}

			res := validation.ValidateCustomResource(nil, s, v)
			if tc.expValErrs != len(res) {
				t.Errorf("Exp Errorlist length to be %d, got %d", tc.expValErrs, len(res))
			}
		})
	}
}

func TestValidatorForObject(t *testing.T) {
	tt := map[string]struct {
		in     runtime.Object
		expErr bool
	}{
		"k8c.io crd": {
			in: &kubermaticv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubermatic.k8c.io/v1",
					Kind:       "Cluster",
				},
			},
		},
		"apps.k8c.io crd": {
			in: &appskubermaticv1.ApplicationDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps.kubermatic.k8c.io/v1",
					Kind:       "ApplicationDefinition",
				},
			},
		},
		"invalid kind": {
			in: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
			},
			expErr: true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			validator, err := NewValidatorForObject(tc.in)
			if err != nil {
				if tc.expErr {
					return
				}

				t.Fatalf("Received unexpected error: %v", err)
			}

			if tc.expErr {
				t.Fatalf("Should have errored, but got validator: %+v", validator)
			}

			if validator == nil {
				t.Fatal("Returned no error, but also no validator.")
			}
		})
	}
}

func TestDefaultProjectResourceQuotaAcceleratorCELValidation(t *testing.T) {
	const validationMessage = "accelerator quotas are not supported in default project resource quotas"

	validAccelerators := []interface{}{
		map[string]interface{}{
			"provider": "kubevirt",
			"resources": map[string]interface{}{
				"nvidia.com/GH100_H200_NVL": "1",
			},
		},
	}

	settingValidator, settingSchema := celValidatorForKubermaticCRD(t, "KubermaticSetting")
	tests := []struct {
		name       string
		quota      map[string]interface{}
		wantDetail string
	}{
		{
			name:  "omitted quota",
			quota: nil,
		},
		{
			name: "CPU only",
			quota: map[string]interface{}{
				"cpu": "1",
			},
		},
		{
			name: "explicit empty accelerator list",
			quota: map[string]interface{}{
				"accelerators": []interface{}{},
			},
		},
		{
			name: "non-empty accelerator list",
			quota: map[string]interface{}{
				"accelerators": validAccelerators,
			},
			wantDetail: validationMessage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setting := map[string]interface{}{
				"apiVersion": kubermaticv1.SchemeGroupVersion.String(),
				"kind":       "KubermaticSetting",
				"metadata": map[string]interface{}{
					"name": kubermaticv1.GlobalSettingsName,
				},
				"spec": map[string]interface{}{},
			}
			if tc.quota != nil {
				setting["spec"].(map[string]interface{})["defaultQuota"] = map[string]interface{}{
					"quota": tc.quota,
				}
			}

			errs, _ := settingValidator.Validate(context.Background(), field.NewPath("root"), settingSchema, setting, nil, celconfig.RuntimeCELCostBudget)
			if tc.wantDetail == "" {
				if len(errs) != 0 {
					t.Fatalf("expected no CEL validation errors, got %v", errs)
				}
				return
			}

			if len(errs) != 1 {
				t.Fatalf("expected one CEL validation error, got %v", errs)
			}
			if errs[0].Detail != tc.wantDetail {
				t.Fatalf("expected CEL validation detail %q, got %q", tc.wantDetail, errs[0].Detail)
			}
		})
	}

	// ResourceDetails is also used by ResourceQuota. Ensure the default-quota
	// restriction remains scoped to its occurrence in KubermaticSetting.
	resourceQuotaValidator, resourceQuotaSchema := celValidatorForKubermaticCRD(t, "ResourceQuota")
	resourceQuota := map[string]interface{}{
		"apiVersion": kubermaticv1.SchemeGroupVersion.String(),
		"kind":       "ResourceQuota",
		"metadata": map[string]interface{}{
			"name": "project-quota",
		},
		"spec": map[string]interface{}{
			"subject": map[string]interface{}{
				"kind": "project",
				"name": "project-id",
			},
			"quota": map[string]interface{}{
				"accelerators": validAccelerators,
			},
		},
	}
	errs, _ := resourceQuotaValidator.Validate(context.Background(), field.NewPath("root"), resourceQuotaSchema, resourceQuota, nil, celconfig.RuntimeCELCostBudget)
	if len(errs) != 0 {
		t.Fatalf("expected project ResourceQuota accelerators to remain valid, got %v", errs)
	}
}

func celValidatorForKubermaticCRD(t *testing.T, kind string) (*cel.Validator, *structuralschema.Structural) {
	t.Helper()

	customResourceDefinition, err := crd.CRDForGVK(kubermaticv1.SchemeGroupVersion.WithKind(kind))
	if err != nil {
		t.Fatalf("failed to load %s CRD: %v", kind, err)
	}

	var versionSchema *apiextensionsv1.JSONSchemaProps
	for i := range customResourceDefinition.Spec.Versions {
		version := &customResourceDefinition.Spec.Versions[i]
		if version.Name == kubermaticv1.SchemeGroupVersion.Version && version.Schema != nil {
			versionSchema = version.Schema.OpenAPIV3Schema
			break
		}
	}
	if versionSchema == nil {
		t.Fatalf("failed to find schema for %s version %q", kind, kubermaticv1.SchemeGroupVersion.Version)
	}

	internalSchema := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(versionSchema, internalSchema, nil); err != nil {
		t.Fatalf("failed to convert %s CRD schema: %v", kind, err)
	}
	structuralSchema, err := structuralschema.NewStructural(internalSchema)
	if err != nil {
		t.Fatalf("failed to create structural schema for %s CRD: %v", kind, err)
	}

	validator := cel.NewValidator(structuralSchema, true, celconfig.PerCallLimit)
	if validator == nil {
		t.Fatalf("expected %s CRD to have CEL validation rules", kind)
	}

	return validator, structuralSchema
}
