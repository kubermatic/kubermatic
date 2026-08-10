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
		if _, exists := seenProviders[accelerator.Provider]; exists {
			allErrs = validateAcceleratorQuotaEntry(allErrs, accelerator, acceleratorsPath.Index(i), true, true, true, nil, true)
		} else {
			allErrs = validateAcceleratorQuotaEntry(allErrs, accelerator, acceleratorsPath.Index(i), true, false, true, nil, true)
		}
		seenProviders[accelerator.Provider] = struct{}{}
	}

	return allErrs.ToAggregate()
}

// ValidateAcceleratorQuotaUpdate validates newly introduced or changed accelerator
// quota data while allowing unchanged invalid persisted components to remain.
func ValidateAcceleratorQuotaUpdate(oldResourceDetails, newResourceDetails kubermaticv1.ResourceDetails) error {
	allErrs := field.ErrorList{}
	acceleratorsPath := field.NewPath("accelerators")
	oldEntryMatches := matchOldAcceleratorQuotaEntries(oldResourceDetails.Accelerators, newResourceDetails.Accelerators)
	duplicateProviders := newlyIntroducedDuplicateProviders(oldResourceDetails.Accelerators, newResourceDetails.Accelerators, oldEntryMatches)

	for i, accelerator := range newResourceDetails.Accelerators {
		oldEntryIndex := oldEntryMatches[i]
		if oldEntryIndex < 0 {
			allErrs = validateAcceleratorQuotaEntry(allErrs, accelerator, acceleratorsPath.Index(i), true, duplicateProviders[i], true, nil, true)
			continue
		}

		oldResources := oldResourceDetails.Accelerators[oldEntryIndex].Resources
		validateEmptyResources := len(accelerator.Resources) == 0 && len(oldResources) != 0
		allErrs = validateAcceleratorQuotaEntry(allErrs, accelerator, acceleratorsPath.Index(i), false, duplicateProviders[i], validateEmptyResources, oldResources, false)
	}

	return allErrs.ToAggregate()
}

func matchOldAcceleratorQuotaEntries(oldAccelerators, newAccelerators []kubermaticv1.AcceleratorQuota) []int {
	matches := make([]int, len(newAccelerators))
	for i := range matches {
		matches[i] = -1
	}
	usedOldEntries := make([]bool, len(oldAccelerators))

	// Preserve exact entries first so moving an unchanged invalid entry does
	// not cause it to be paired with a different duplicate-provider entry.
	for newIndex, newAccelerator := range newAccelerators {
		for oldIndex, oldAccelerator := range oldAccelerators {
			if usedOldEntries[oldIndex] || oldAccelerator.Provider != newAccelerator.Provider || !resourceListsEqual(oldAccelerator.Resources, newAccelerator.Resources) {
				continue
			}
			matches[newIndex] = oldIndex
			usedOldEntries[oldIndex] = true
			break
		}
	}

	// Then correlate changed entries by provider. This treats the provider
	// component as unchanged while still validating every changed resource.
	for newIndex, newAccelerator := range newAccelerators {
		if matches[newIndex] >= 0 {
			continue
		}
		for oldIndex, oldAccelerator := range oldAccelerators {
			if usedOldEntries[oldIndex] || oldAccelerator.Provider != newAccelerator.Provider {
				continue
			}
			matches[newIndex] = oldIndex
			usedOldEntries[oldIndex] = true
			break
		}
	}

	return matches
}

func resourceListsEqual(left, right corev1.ResourceList) bool {
	if len(left) != len(right) {
		return false
	}
	for resourceName, leftQuantity := range left {
		rightQuantity, exists := right[resourceName]
		if !exists || leftQuantity.Cmp(rightQuantity) != 0 {
			return false
		}
	}
	return true
}

func newlyIntroducedDuplicateProviders(oldAccelerators, newAccelerators []kubermaticv1.AcceleratorQuota, oldEntryMatches []int) map[int]bool {
	oldProviderCounts := map[string]int{}
	for _, accelerator := range oldAccelerators {
		oldProviderCounts[accelerator.Provider]++
	}

	// A valid object may contain one entry per provider. Existing duplicate
	// occurrences are grandfathered, but updates may not increase their count.
	remainingAllowedOccurrences := map[string]int{}
	for provider, count := range oldProviderCounts {
		remainingAllowedOccurrences[provider] = max(1, count)
	}
	for _, accelerator := range newAccelerators {
		if _, exists := remainingAllowedOccurrences[accelerator.Provider]; !exists {
			remainingAllowedOccurrences[accelerator.Provider] = 1
		}
	}

	// Reserve allowance for matched old entries before considering newly
	// introduced entries, regardless of how list positions shifted.
	for newIndex, oldIndex := range oldEntryMatches {
		if oldIndex >= 0 {
			remainingAllowedOccurrences[newAccelerators[newIndex].Provider]--
		}
	}

	duplicates := map[int]bool{}
	for i, accelerator := range newAccelerators {
		if oldEntryMatches[i] >= 0 {
			continue
		}
		if remainingAllowedOccurrences[accelerator.Provider] > 0 {
			remainingAllowedOccurrences[accelerator.Provider]--
			continue
		}
		duplicates[i] = true
	}

	return duplicates
}

func validateAcceleratorQuotaEntry(
	allErrs field.ErrorList,
	accelerator kubermaticv1.AcceleratorQuota,
	acceleratorPath *field.Path,
	validateProvider bool,
	validateDuplicateProvider bool,
	validateEmptyResources bool,
	oldResources corev1.ResourceList,
	validateAllResources bool,
) field.ErrorList {
	providerPath := acceleratorPath.Child("provider")
	resourcesPath := acceleratorPath.Child("resources")

	if validateProvider {
		if accelerator.Provider == "" {
			allErrs = append(allErrs, field.Required(providerPath, "provider must not be empty"))
		} else if accelerator.Provider != string(kubermaticv1.KubevirtCloudProvider) {
			allErrs = append(allErrs, field.NotSupported(providerPath, accelerator.Provider, []string{string(kubermaticv1.KubevirtCloudProvider)}))
		}
	}
	if validateDuplicateProvider {
		allErrs = append(allErrs, field.Duplicate(providerPath, accelerator.Provider))
	}

	if len(accelerator.Resources) == 0 {
		if validateEmptyResources {
			allErrs = append(allErrs, field.Required(resourcesPath, fmt.Sprintf("accelerator quota for provider %q must contain at least one resource", accelerator.Provider)))
		}
		return allErrs
	}

	resourceNames := make([]string, 0, len(accelerator.Resources))
	for resourceName := range accelerator.Resources {
		resourceNames = append(resourceNames, string(resourceName))
	}
	sort.Strings(resourceNames)

	for _, resourceName := range resourceNames {
		quantity := accelerator.Resources[corev1.ResourceName(resourceName)]
		if !validateAllResources {
			oldQuantity, exists := oldResources[corev1.ResourceName(resourceName)]
			if exists && quantity.Cmp(oldQuantity) == 0 {
				continue
			}
		}

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
