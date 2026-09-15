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

package validation

import (
	"fmt"
	"sort"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateAcceleratorQuota validates provider-specific accelerator quota entries.
func ValidateAcceleratorQuota(resourceDetails kubermaticv1.ResourceDetails) error {
	allErrs := field.ErrorList{}
	acceleratorsPath := field.NewPath("accelerators")
	seenProviders := map[string]struct{}{}

	for i, accelerator := range resourceDetails.Accelerators {
		_, duplicateProvider := seenProviders[accelerator.Provider]
		allErrs = validateAcceleratorQuotaEntry(allErrs, accelerator, acceleratorsPath.Index(i), duplicateProvider)
		seenProviders[accelerator.Provider] = struct{}{}
	}

	return allErrs.ToAggregate()
}

func validateAcceleratorQuotaEntry(
	allErrs field.ErrorList,
	accelerator kubermaticv1.AcceleratorQuota,
	acceleratorPath *field.Path,
	duplicateProvider bool,
) field.ErrorList {
	providerPath := acceleratorPath.Child("provider")
	resourcesPath := acceleratorPath.Child("resources")

	if accelerator.Provider == "" {
		allErrs = append(allErrs, field.Required(providerPath, "provider must not be empty"))
	} else if accelerator.Provider != string(kubermaticv1.KubevirtCloudProvider) {
		allErrs = append(allErrs, field.NotSupported(providerPath, accelerator.Provider, []string{string(kubermaticv1.KubevirtCloudProvider)}))
	}
	if duplicateProvider {
		allErrs = append(allErrs, field.Duplicate(providerPath, accelerator.Provider))
	}

	if len(accelerator.Resources) == 0 {
		allErrs = append(allErrs, field.Required(resourcesPath, fmt.Sprintf("accelerator quota for provider %q must contain at least one resource", accelerator.Provider)))
		return allErrs
	}

	resourceNames := make([]string, 0, len(accelerator.Resources))
	for resourceName := range accelerator.Resources {
		resourceNames = append(resourceNames, string(resourceName))
	}
	sort.Strings(resourceNames)

	for _, resourceName := range resourceNames {
		quantity := accelerator.Resources[corev1.ResourceName(resourceName)]
		resourcePath := resourcesPath.Key(resourceName)
		if strings.Count(resourceName, "/") != 1 {
			allErrs = append(allErrs, field.Invalid(resourcePath, resourceName, fmt.Sprintf("accelerator resource %q for provider %q must be a qualified name with exactly one '/'", resourceName, accelerator.Provider)))
		} else if validationErrors := k8svalidation.IsQualifiedName(resourceName); len(validationErrors) > 0 {
			allErrs = append(allErrs, field.Invalid(resourcePath, resourceName, fmt.Sprintf("accelerator resource %q for provider %q is not a valid qualified name: %s", resourceName, accelerator.Provider, strings.Join(validationErrors, "; "))))
		}

		if quantity.Sign() < 0 {
			allErrs = append(allErrs, field.Invalid(resourcePath, quantity.String(), fmt.Sprintf("accelerator resource %q for provider %q must not be negative", resourceName, accelerator.Provider)))
		}

		if _, exact := quantity.AsScale(0); !exact {
			allErrs = append(allErrs, field.Invalid(resourcePath, quantity.String(), fmt.Sprintf("accelerator resource %q for provider %q must be a whole number", resourceName, accelerator.Provider)))
		}
	}

	return allErrs
}
