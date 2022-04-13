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

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateApplicationInstallationSpec validates the ApplicationInstallation Spec.
func ValidateApplicationInstallationSpec(ctx context.Context, client ctrlruntimeclient.Client, spec appkubermaticv1.ApplicationInstallationSpec) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := field.ErrorList{}

	// Ensure that the referenced ApplicationDefinition exists
	ad := &appkubermaticv1.ApplicationDefinition{}
	err := client.Get(ctx, types.NamespacedName{Name: spec.ApplicationRef.Name}, ad)
	if err != nil {
		if kerrors.IsNotFound(err) {
			allErrs = append(allErrs, field.NotFound(specPath.Child("applicationRef", "name"), spec.ApplicationRef.Name))
		} else {
			allErrs = append(allErrs, field.InternalError(specPath.Child("applicationRef", "name"), err))
		}
		return allErrs
	}

	// Ensure that there is matching version defined in ApplicationDefinition
	exists := false
	desiredVersion := spec.ApplicationRef.Version.String()
	for _, version := range ad.Spec.Versions {
		if version.Version == desiredVersion {
			exists = true
		}
	}

	if !exists {
		allErrs = append(allErrs, field.NotFound(specPath.Child("applicationRef", "version"), spec.ApplicationRef.Version))
	}

	return allErrs
}

// ValidateApplicationInstallationUpdate validates the new ApplicationInstallation for immutable fields.
func ValidateApplicationInstallationUpdate(ctx context.Context, client ctrlruntimeclient.Client, newAI, oldAI appkubermaticv1.ApplicationInstallation) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := field.ErrorList{}

	// Validation for new ApplicationInstallation Spec
	if errs := ValidateApplicationInstallationSpec(ctx, client, newAI.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// Validate .Spec.Namespace.Create for immutability
	if oldAI.Spec.Namespace.Create != newAI.Spec.Namespace.Create {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			newAI.Spec.Namespace.Create,
			oldAI.Spec.Namespace.Create,
			specPath.Child("namespace", "create"),
		)...)
	}

	// Validate .Spec.Namespace.Name for immutability
	if oldAI.Spec.Namespace.Name != newAI.Spec.Namespace.Name {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			newAI.Spec.Namespace.Name,
			oldAI.Spec.Namespace.Name,
			specPath.Child("namespace", "name"),
		)...)
	}

	// Validate .Spec.ApplicationRef for immutability
	if oldAI.Spec.ApplicationRef != newAI.Spec.ApplicationRef {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			newAI.Spec.ApplicationRef,
			oldAI.Spec.ApplicationRef,
			specPath.Child("applicationRef"),
		)...)
	}

	return allErrs
}
