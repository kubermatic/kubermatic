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

package validation_test

import (
	"slices"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateAcceleratorQuota(t *testing.T) {
	testCases := []struct {
		name             string
		accelerators     []kubermaticv1.AcceleratorQuota
		expectedMessages []string
	}{
		{name: "omitted accelerator quota"},
		{
			name:         "empty accelerator quota",
			accelerators: []kubermaticv1.AcceleratorQuota{},
		},
		{
			name: "valid kubevirt provider",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("4"),
				},
			}},
		},
		{
			name: "explicit zero",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("0"),
				},
			}},
		},
		{
			name: "multiple resource names",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("4"),
					"nvidia.com/A100_80GB":      resource.MustParse("2"),
				},
			}},
		},
		{
			name: "large whole quantity",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("9223372036854775808"),
				},
			}},
		},
		{
			name: "negative quantity",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("-1"),
				},
			}},
			expectedMessages: []string{`accelerator resource "nvidia.com/GH100_H200_NVL" for provider "kubevirt" must not be negative`},
		},
		{
			name: "fractional quantity",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("500m"),
				},
			}},
			expectedMessages: []string{`accelerator resource "nvidia.com/GH100_H200_NVL" for provider "kubevirt" must be a whole number`},
		},
		{
			name: "unqualified resource name",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"GH100_H200_NVL": resource.MustParse("1"),
				},
			}},
			expectedMessages: []string{`accelerator resource "GH100_H200_NVL" for provider "kubevirt" must be a qualified name with exactly one '/'`},
		},
		{
			name: "resource name with multiple slashes",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/products/GH100_H200_NVL": resource.MustParse("1"),
				},
			}},
			expectedMessages: []string{`accelerator resource "nvidia.com/products/GH100_H200_NVL" for provider "kubevirt" must be a qualified name with exactly one '/'`},
		},
		{
			name: "malformed qualified resource name",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"NVIDIA!.com/GH100": resource.MustParse("1"),
				},
			}},
			expectedMessages: []string{`accelerator resource "NVIDIA!.com/GH100" for provider "kubevirt" is not a valid qualified name`},
		},
		{
			name: "nil resources map",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "kubevirt",
			}},
			expectedMessages: []string{`accelerator quota for provider "kubevirt" must contain at least one resource`},
		},
		{
			name: "empty resources map",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider:  "kubevirt",
				Resources: corev1.ResourceList{},
			}},
			expectedMessages: []string{`accelerator quota for provider "kubevirt" must contain at least one resource`},
		},
		{
			name: "empty provider",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Resources: corev1.ResourceList{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("1"),
				},
			}},
			expectedMessages: []string{"provider must not be empty"},
		},
		{
			name: "unsupported provider",
			accelerators: []kubermaticv1.AcceleratorQuota{{
				Provider: "aws",
				Resources: corev1.ResourceList{
					"accelerators.kubermatic.io/h100": resource.MustParse("1"),
				},
			}},
			expectedMessages: []string{`accelerators[0].provider: Unsupported value: "aws": supported values: "kubevirt"`},
		},
		{
			name: "duplicate provider",
			accelerators: []kubermaticv1.AcceleratorQuota{
				{
					Provider: "kubevirt",
					Resources: corev1.ResourceList{
						"nvidia.com/GH100_H200_NVL": resource.MustParse("1"),
					},
				},
				{
					Provider: "kubevirt",
					Resources: corev1.ResourceList{
						"nvidia.com/A100_80GB": resource.MustParse("1"),
					},
				},
			},
			expectedMessages: []string{`accelerators[1].provider: Duplicate value: "kubevirt"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validation.ValidateAcceleratorQuota(kubermaticv1.ResourceDetails{Accelerators: tc.accelerators})
			if len(tc.expectedMessages) == 0 {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			for _, expectedMessage := range tc.expectedMessages {
				if !strings.Contains(err.Error(), expectedMessage) {
					t.Errorf("expected error %q to contain %q", err, expectedMessage)
				}
			}
		})
	}
}

func TestValidateAcceleratorQuotaAggregatesErrorsDeterministically(t *testing.T) {
	resourceDetails := kubermaticv1.ResourceDetails{
		Accelerators: []kubermaticv1.AcceleratorQuota{
			{
				Provider: "aws",
				Resources: corev1.ResourceList{
					"z.example/GPU": resource.MustParse("-1"),
					"bad":           resource.MustParse("500m"),
					"a.example/GPU": resource.MustParse("-500m"),
				},
			},
			{
				Provider: "kubevirt",
			},
			{
				Provider: "kubevirt",
				Resources: corev1.ResourceList{
					"nvidia.com/GPU": resource.MustParse("1"),
				},
			},
		},
	}

	expectedFields := []string{
		"accelerators[0].provider",
		"accelerators[0].resources[a.example/GPU]",
		"accelerators[0].resources[a.example/GPU]",
		"accelerators[0].resources[bad]",
		"accelerators[0].resources[bad]",
		"accelerators[0].resources[z.example/GPU]",
		"accelerators[1].resources",
		"accelerators[2].provider",
	}

	var expectedError string
	for i := 0; i < 50; i++ {
		err := validation.ValidateAcceleratorQuota(resourceDetails)
		if err == nil {
			t.Fatal("expected validation errors, got nil")
		}

		aggregate, ok := err.(interface{ Errors() []error })
		if !ok {
			t.Fatalf("expected an aggregate error, got %T", err)
		}

		actualFields := make([]string, 0, len(aggregate.Errors()))
		for _, validationError := range aggregate.Errors() {
			fieldName, _, _ := strings.Cut(validationError.Error(), ":")
			actualFields = append(actualFields, fieldName)
		}
		if !slices.Equal(actualFields, expectedFields) {
			t.Fatalf("expected fields %v, got %v", expectedFields, actualFields)
		}

		if i == 0 {
			expectedError = err.Error()
			continue
		}
		if err.Error() != expectedError {
			t.Fatalf("validation errors are not deterministic:\nfirst: %s\nnext:  %s", expectedError, err)
		}
	}

	for _, identity := range []string{`provider "aws"`, `resource "a.example/GPU"`, `resource "bad"`, `resource "z.example/GPU"`, `provider "kubevirt"`} {
		if !strings.Contains(expectedError, identity) {
			t.Errorf("expected aggregate error to identify %s, got: %s", identity, expectedError)
		}
	}
}
