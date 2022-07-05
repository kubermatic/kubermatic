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

package validation

import (
	"context"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateApplicationInstallationSpec validates the ApplicationInstallation Spec.
func ValidateApplicationInstallationSpec(ctx context.Context, spec appskubermaticv1.ApplicationInstallationSpec) field.ErrorList {
	return field.ErrorList{}
}

// ValidateApplicationInstallationUpdate validates the new ApplicationInstallation for immutable fields.
func ValidateApplicationInstallationUpdate(ctx context.Context, newAI, oldAI appskubermaticv1.ApplicationInstallation) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := field.ErrorList{}

	// Validation for new ApplicationInstallation Spec
	if errs := ValidateApplicationInstallationSpec(ctx, newAI.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// Validate .Spec.Namespace.Create for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.Namespace.Create,
		oldAI.Spec.Namespace.Create,
		specPath.Child("namespace", "create"),
	)...)

	// Validate .Spec.Namespace.Name for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.Namespace.Name,
		oldAI.Spec.Namespace.Name,
		specPath.Child("namespace", "name"),
	)...)

	// Validate .Spec.ApplicationRef.Version for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.ApplicationRef.Version.Version.String(),
		oldAI.Spec.ApplicationRef.Version.Version.String(),
		specPath.Child("applicationRef", "version"),
	)...)

	// Validate .Spec.ApplicationRef.Name for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.ApplicationRef.Name,
		oldAI.Spec.ApplicationRef.Name,
		specPath.Child("applicationRef", "name"),
	)...)

	return allErrs
}
