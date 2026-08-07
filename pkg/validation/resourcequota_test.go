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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateAcceleratorQuota(t *testing.T) {
	testCases := []struct {
		name         string
		accelerators map[string]resource.Quantity
		expectError  bool
	}{
		{name: "unset accelerator quota"},
		{
			name: "whole accelerator quantities",
			accelerators: map[string]resource.Quantity{
				"kubevirt/nvidia.com/GH100_H200_NVL": resource.MustParse("2"),
				"kubevirt/nvidia.com/A100_80GB":      resource.MustParse("0"),
			},
		},
		{
			name: "empty accounting key",
			accelerators: map[string]resource.Quantity{
				"": resource.MustParse("1"),
			},
			expectError: true,
		},
		{
			name: "negative quantity",
			accelerators: map[string]resource.Quantity{
				"kubevirt/nvidia.com/GH100_H200_NVL": resource.MustParse("-1"),
			},
			expectError: true,
		},
		{
			name: "fractional quantity",
			accelerators: map[string]resource.Quantity{
				"kubevirt/nvidia.com/GH100_H200_NVL": resource.MustParse("500m"),
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validation.ValidateAcceleratorQuota(kubermaticv1.ResourceDetails{Accelerators: tc.accelerators})
			if (err != nil) != tc.expectError {
				t.Fatalf("expected error: %t, got: %v", tc.expectError, err)
			}
		})
	}
}
