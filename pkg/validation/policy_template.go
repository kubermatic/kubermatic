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

package validation

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidatePolicyTemplate validates the PolicyTemplate spec.
func ValidatePolicyTemplate(ctx context.Context, client ctrlruntimeclient.Client, template *kubermaticv1.PolicyTemplate) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")

	// Validate visibility value
	if template.Spec.Visibility != kubermaticv1.PolicyTemplateGlobalVisibility && template.Spec.Visibility != kubermaticv1.PolicyTemplateProjectVisibility {
		allErrs = append(allErrs, field.Invalid(specPath.Child("visibility"), template.Spec.Visibility, "visibility must be one of: global, project"))
		return allErrs
	}

	// Validate ProjectID based on visibility
	switch template.Spec.Visibility {
	case kubermaticv1.PolicyTemplateGlobalVisibility:
		// Global templates must not have a ProjectID
		if template.Spec.ProjectID != "" {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("projectID"),
				"global visibility policy templates must not specify a ProjectID"))
		}

	case kubermaticv1.PolicyTemplateProjectVisibility:
		// Project templates must have a ProjectID
		if template.Spec.ProjectID == "" {
			allErrs = append(allErrs, field.Required(specPath.Child("projectID"),
				"project visibility policy templates must specify a ProjectID"))
		}
	}

	// Validate Rules are set and not empty
	if len(template.Spec.KyvernoPolicySpec.Rules) == 0 {
		allErrs = append(allErrs, field.Required(specPath.Child("kyvernoPolicySpec", "rules"),
			"at least one rule must be specified in kyvernoPolicySpec.rules"))
	}

	return allErrs
}

// ValidatePolicyTemplateCreate validates creation of a PolicyTemplate.
func ValidatePolicyTemplateCreate(ctx context.Context, client ctrlruntimeclient.Client, template *kubermaticv1.PolicyTemplate) field.ErrorList {
	return ValidatePolicyTemplate(ctx, client, template)
}

// ValidatePolicyTemplateUpdate validates updates to a PolicyTemplate.
func ValidatePolicyTemplateUpdate(ctx context.Context, client ctrlruntimeclient.Client, oldTemplate, newTemplate *kubermaticv1.PolicyTemplate) field.ErrorList {
	allErrs := ValidatePolicyTemplate(ctx, client, newTemplate)

	// Validate immutable fields
	specPath := field.NewPath("spec")

	// Visibility is immutable
	if oldTemplate.Spec.Visibility != newTemplate.Spec.Visibility {
		allErrs = append(allErrs, field.Invalid(specPath.Child("visibility"), newTemplate.Spec.Visibility, "visibility is immutable"))
	}

	// ProjectID is immutable
	if oldTemplate.Spec.ProjectID != newTemplate.Spec.ProjectID {
		allErrs = append(allErrs, field.Invalid(specPath.Child("projectID"), newTemplate.Spec.ProjectID, "projectID is immutable"))
	}

	return allErrs
}
