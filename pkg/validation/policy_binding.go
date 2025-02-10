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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidatePolicyBinding validates the PolicyBinding spec.
func ValidatePolicyBinding(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")

	// Validate scope value
	if binding.Spec.Scope != "global" && binding.Spec.Scope != "project" {
		allErrs = append(allErrs, field.Invalid(specPath.Child("scope"), binding.Spec.Scope, "scope must be one of: global, project"))
		return allErrs
	}

	// Validate PolicyTemplateRef exists
	policyTemplate := &kubermaticv1.PolicyTemplate{}
	if err := client.Get(ctx, types.NamespacedName{Name: binding.Spec.PolicyTemplateRef.Name}, policyTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.NotFound(specPath.Child("policyTemplateRef", "name"), binding.Spec.PolicyTemplateRef.Name))
		} else {
			allErrs = append(allErrs, field.InternalError(specPath.Child("policyTemplateRef", "name"), err))
		}
		return allErrs
	}

	// Validate template visibility and binding scope compatibility
	switch binding.Spec.Scope {
	case "global":
		// Global bindings can ONLY use global templates
		if policyTemplate.Spec.Visibility != "global" {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("policyTemplateRef"),
				"global scope policy bindings can only reference policy templates with global visibility"))
		}

		// Validate global binding target selectors
		if err := validateGlobalBindingSelectors(binding, specPath); err != nil {
			allErrs = append(allErrs, err...)
		}

	case "project":
		// Project bindings can ONLY use project templates
		if policyTemplate.Spec.Visibility != "project" {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("policyTemplateRef"),
				"project scope policy bindings can only reference policy templates with project visibility"))
		}

		// Project templates MUST have a project ID
		if policyTemplate.Spec.ProjectID == "" {
			allErrs = append(allErrs, field.Required(specPath.Child("policyTemplateRef"),
				"project-visible policy templates must specify a ProjectID"))
		}

		// Validate project binding target selectors
		if err := validateProjectBindingSelectors(binding, specPath); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	return allErrs
}

// validateGlobalBindingSelectors validates selectors for global-scoped bindings
func validateGlobalBindingSelectors(binding *kubermaticv1.PolicyBinding, specPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// For global bindings, projects selector must be set
	if isEmptyResourceSelector(&binding.Spec.Target.Projects) {
		allErrs = append(allErrs, field.Required(specPath.Child("target", "projects"),
			"global scope policy bindings must specify target projects"))
	}

	// If selectAll is true, no specific names or labels should be set
	if binding.Spec.Target.Projects.SelectAll {
		if len(binding.Spec.Target.Projects.Name) > 0 {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "projects", "name"),
				"cannot specify project names when selectAll is true"))
		}
		if binding.Spec.Target.Projects.LabelSelector != nil {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "projects", "labelSelector"),
				"cannot specify project labels when selectAll is true"))
		}
	}

	// Clusters selector must be empty for global bindings
	if !isEmptyResourceSelector(&binding.Spec.Target.Clusters) {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "clusters"),
			"global scope policy bindings must not specify target clusters"))
	}

	return allErrs
}

// validateProjectBindingSelectors validates selectors for project-scoped bindings
func validateProjectBindingSelectors(binding *kubermaticv1.PolicyBinding, specPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Project bindings must not have project selectors
	if !isEmptyResourceSelector(&binding.Spec.Target.Projects) {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "projects"),
			"project scope policy bindings must not specify target projects"))
	}

	// For project bindings, clusters selector must be set
	if isEmptyResourceSelector(&binding.Spec.Target.Clusters) {
		allErrs = append(allErrs, field.Required(specPath.Child("target", "clusters"),
			"project scope policy bindings must specify target clusters"))
	}

	// If selectAll is true, no specific names or labels should be set
	if binding.Spec.Target.Clusters.SelectAll {
		if len(binding.Spec.Target.Clusters.Name) > 0 {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "clusters", "name"),
				"cannot specify cluster names when selectAll is true"))
		}
		if binding.Spec.Target.Clusters.LabelSelector != nil {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "clusters", "labelSelector"),
				"cannot specify cluster labels when selectAll is true"))
		}
	}

	return allErrs
}

// ValidatePolicyBindingCreate validates creation of a PolicyBinding
func ValidatePolicyBindingCreate(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) field.ErrorList {
	return ValidatePolicyBinding(ctx, client, binding)
}

// ValidatePolicyBindingUpdate validates updates to a PolicyBinding
func ValidatePolicyBindingUpdate(ctx context.Context, client ctrlruntimeclient.Client, oldBinding, newBinding *kubermaticv1.PolicyBinding) field.ErrorList {
	allErrs := ValidatePolicyBinding(ctx, client, newBinding)

	// Validate immutable fields
	specPath := field.NewPath("spec")

	// Scope is immutable
	if oldBinding.Spec.Scope != newBinding.Spec.Scope {
		allErrs = append(allErrs, field.Invalid(specPath.Child("scope"), newBinding.Spec.Scope, "scope is immutable"))
	}

	return allErrs
}

// isEmptyResourceSelector checks if a ResourceSelector is empty
func isEmptyResourceSelector(selector *kubermaticv1.ResourceSelector) bool {
	return !selector.SelectAll && len(selector.Name) == 0 && selector.LabelSelector == nil
}
