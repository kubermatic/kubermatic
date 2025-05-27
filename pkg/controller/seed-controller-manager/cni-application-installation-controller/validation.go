/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package cniapplicationinstallationcontroller

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateValuesUpdate validates the update operation on provided Cilium Helm values.
func ValidateValuesUpdate(newValues, oldValues map[string]any, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// Validate immutability of specific top-level value subtrees, managed solely by KKP
	exclusions := []exclusion{
		{
			fullPath:  "cni.chainingMode",
			pathParts: []string{"cni", "chainingMode"},
		},
		{
			fullPath:  "ipam.operator.clusterPoolIPv4MaskSize",
			pathParts: []string{"ipam", "operator", "clusterPoolIPv4MaskSize"},
		},
		{
			fullPath:  "ipam.operator.clusterPoolIPv6MaskSize",
			pathParts: []string{"ipam", "operator", "clusterPoolIPv6MaskSize"},
		},
	}
	allErrs = append(allErrs, validateImmutableValues(newValues, oldValues, fieldPath, []string{
		"cni",
		"ipam",
		"ipv6",
	}, exclusions)...)

	// Validate that mandatory top-level values are present
	allErrs = append(allErrs, validateMandatoryValues(newValues, fieldPath, []string{
		"kubeProxyReplacement", // can be changed if switching KKP proxy mode, but must be present
	})...)

	// Validate config for strict kubeProxyReplacement (ebpf "proxy mode")
	if newValues["kubeProxyReplacement"] == "strict" {
		// Validate mandatory values for kube-proxy-replacement
		allErrs = append(allErrs, validateMandatoryValues(newValues, fieldPath, []string{
			"k8sServiceHost",
			"k8sServicePort",
		})...)
	}

	// Validate immutability of operator.securityContext
	operPath := fieldPath.Child("operator")
	newOperator, ok := newValues["operator"].(map[string]any)
	if !ok {
		allErrs = append(allErrs, field.Invalid(operPath, newValues["operator"], "value missing or incorrect"))
	}
	oldOperator, ok := oldValues["operator"].(map[string]any)
	if !ok {
		allErrs = append(allErrs, field.Invalid(operPath, oldValues["operator"], "value missing or incorrect"))
	}
	allErrs = append(allErrs, validateImmutableValues(newOperator, oldOperator, operPath, []string{
		"securityContext",
	}, []exclusion{})...)

	return allErrs
}

type exclusion struct {
	fullPath  string
	pathParts []string
}

func validateImmutableValues(newValues, oldValues map[string]any, fieldPath *field.Path, immutableValues []string, exclusions []exclusion) field.ErrorList {
	allErrs := field.ErrorList{}
	allowedValues := map[string]bool{}

	for _, v := range immutableValues {
		// Check if the value in `oldValues` is non-empty and missing in `newValues`
		if _, existsInOld := oldValues[v]; existsInOld {
			if _, existsInNew := newValues[v]; !existsInNew || newValues[v] == nil || fmt.Sprint(newValues[v]) == "" {
				allErrs = append(allErrs, field.Invalid(fieldPath.Child(v), newValues[v], "value is immutable"))
				continue
			}
		}
		for _, exclusion := range exclusions {
			if strings.HasPrefix(exclusion.fullPath, v) {
				if excludedKeyExists(newValues, exclusion.pathParts...) {
					allowedValues[v] = true
				}
				if excludedKeyExists(oldValues, exclusion.pathParts...) {
					allowedValues[v] = true
				}
			}
		}
		if fmt.Sprint(oldValues[v]) != fmt.Sprint(newValues[v]) && !allowedValues[v] {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child(v), newValues[v], "value is immutable"))
		}
	}
	return allErrs
}

func validateMandatoryValues(newValues map[string]any, fieldPath *field.Path, mandatoryValues []string) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, v := range mandatoryValues {
		if _, ok := newValues[v]; !ok {
			allErrs = append(allErrs, field.NotFound(fieldPath.Child(v), newValues[v]))
		}
	}
	return allErrs
}

func excludedKeyExists(values map[string]any, keys ...string) bool {
	if len(keys) == 0 || values == nil {
		return false
	}
	root := keys[0]
	value, ok := values[root]
	if !ok {
		return false
	}

	if len(keys) == 1 {
		return true
	}

	if innerMap, ok := value.(map[string]any); ok {
		return excludedKeyExists(innerMap, keys[1:]...)
	}

	return false
}
