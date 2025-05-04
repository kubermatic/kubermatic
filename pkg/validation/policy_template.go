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
	"encoding/json"
	"fmt"

	kyvernoapiv1 "github.com/kyverno/kyverno/api/kyverno/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	validVisibilities = sets.New(kubermaticv1.PolicyTemplateVisibilityGlobal, kubermaticv1.PolicyTemplateVisibilityProject, kubermaticv1.PolicyTemplateVisibilityCluster)
)

// ValidatePolicyTemplate validates the PolicyTemplate resource.
func ValidatePolicyTemplate(template *kubermaticv1.PolicyTemplate) field.ErrorList {
	var allErrs field.ErrorList
	specPath := field.NewPath("spec")

	if template.Spec.Title == "" {
		allErrs = append(allErrs, field.Required(specPath.Child("title"), "title is required"))
	}

	if template.Spec.Description == "" {
		allErrs = append(allErrs, field.Required(specPath.Child("description"), "description is required"))
	}

	if !validVisibilities.Has(template.Spec.Visibility) {
		allErrs = append(allErrs, field.NotSupported(specPath.Child("visibility"), template.Spec.Visibility, validVisibilities.UnsortedList()))
	} else {
		switch template.Spec.Visibility {
		case kubermaticv1.PolicyTemplateVisibilityProject:
			if template.Spec.ProjectID == "" {
				allErrs = append(allErrs, field.Required(specPath.Child("projectID"), fmt.Sprintf("projectID is required when visibility is '%s'", kubermaticv1.PolicyTemplateVisibilityProject)))
			}
			if template.Spec.Target != nil && template.Spec.Target.ProjectSelector != nil {
				allErrs = append(allErrs, field.Forbidden(specPath.Child("target", "projectSelector"), fmt.Sprintf("projectSelector cannot be set when visibility is '%s'; scope is defined by projectID", kubermaticv1.PolicyTemplateVisibilityProject)))
			}
		case kubermaticv1.PolicyTemplateVisibilityGlobal:
			if template.Spec.ProjectID != "" {
				allErrs = append(allErrs, field.Invalid(specPath.Child("projectID"), template.Spec.ProjectID, fmt.Sprintf("projectID must be empty when visibility is '%s'", kubermaticv1.PolicyTemplateVisibilityGlobal)))
			}
		}
	}

	if template.Spec.Target != nil {
		targetPath := specPath.Child("target")
		if template.Spec.Target.ProjectSelector != nil {
			_, err := metav1.LabelSelectorAsSelector(template.Spec.Target.ProjectSelector)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(targetPath.Child("projectSelector"), template.Spec.Target.ProjectSelector, fmt.Sprintf("invalid label selector syntax: %v", err)))
			}
		}
		if template.Spec.Target.ClusterSelector != nil {
			_, err := metav1.LabelSelectorAsSelector(template.Spec.Target.ClusterSelector)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(targetPath.Child("clusterSelector"), template.Spec.Target.ClusterSelector, fmt.Sprintf("invalid label selector syntax: %v", err)))
			}
		}
	}

	// Validate the raw PolicySpec from Kyverno
	if template.Spec.PolicySpec.Raw == nil || len(template.Spec.PolicySpec.Raw) == 0 {
		allErrs = append(allErrs, field.Required(specPath.Child("policySpec"), "policySpec is required"))
	} else {
		var kyvernoPolicySpec kyvernoapiv1.Spec
		if err := json.Unmarshal(template.Spec.PolicySpec.Raw, &kyvernoPolicySpec); err != nil {
			allErrs = append(allErrs, field.Invalid(specPath.Child("policySpec"), "<raw data>", fmt.Sprintf("failed to unmarshal policySpec into Kyverno Spec: %v", err)))
		} else {
			allErrs = append(allErrs, validateKyvernoPolicySpec(specPath.Child("policySpec"), &kyvernoPolicySpec)...)
		}
	}

	return allErrs
}

// validateKyvernoPolicySpec performs structural validation on the Kyverno PolicySpec.
// This is not exhaustive Kyverno validation but checks key components.
func validateKyvernoPolicySpec(fldPath *field.Path, spec *kyvernoapiv1.Spec) field.ErrorList {
	var allErrs field.ErrorList

	if len(spec.Rules) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("rules"), "at least one rule is required"))
	}

	for i, rule := range spec.Rules {
		rulePath := fldPath.Child("rules").Index(i)
		if rule.Name == "" {
			allErrs = append(allErrs, field.Required(rulePath.Child("name"), "rule name is required"))
		}

		hasMatch := len(rule.MatchResources.Any) > 0 || len(rule.MatchResources.All) > 0

		if !hasMatch {
			allErrs = append(allErrs, field.Required(rulePath.Child("match"), "a match block (match.any or match.all) is required"))
		}

		hasAction := rule.HasValidate() || rule.HasMutate() || rule.HasGenerate() || rule.HasVerifyImages()
		if !hasAction {
			allErrs = append(allErrs, field.Required(rulePath.Child("<action>"), "at least one action (validate, mutate, generate, verifyImages) is required per rule"))
		}

		if rule.HasValidate() {
			if rule.Validation.Message == "" && rule.Validation.RawPattern == nil && rule.Validation.RawAnyPattern == nil && rule.Validation.ForEachValidation == nil && rule.Validation.Manifests == nil && rule.Validation.PodSecurity == nil && rule.Validation.CEL == nil {
				allErrs = append(allErrs, field.Required(rulePath.Child("validate"), "validation rule requires a message, pattern, anyPattern, foreach, manifests, podSecurity or cel block"))
			}
		}

		if rule.HasMutate() && rule.Mutation.RawPatchStrategicMerge == nil && len(rule.Mutation.PatchesJSON6902) == 0 && rule.Mutation.ForEachMutation == nil {
			allErrs = append(allErrs, field.Required(rulePath.Child("mutate"), "mutation rule requires a patchStrategicMerge, patchesJson6902, or foreach block"))
		}
		if rule.HasGenerate() && rule.Generation.RawData == nil && rule.Generation.Clone == (kyvernoapiv1.CloneFrom{}) {
			allErrs = append(allErrs, field.Required(rulePath.Child("generate"), "generation rule requires a data or clone block"))
		}
		if rule.HasVerifyImages() && len(rule.VerifyImages) == 0 {
			allErrs = append(allErrs, field.Required(rulePath.Child("verifyImages"), "verifyImages rule requires image verification entries"))
		}
	}

	// Kyverno allows the following values for validationFailureAction: audit, enforce, Audit, Enforce.
	// The lowercase values are deprecated, but Kyverno still supports them, so we need to check for them.
	// This field is deprecated too and scheduled to be removed in future versions.
	validActions := []string{string(kyvernoapiv1.Audit), string(kyvernoapiv1.Enforce), "audit", "enforce"}
	validActionSet := sets.NewString(validActions...)
	if spec.ValidationFailureAction != "" && !validActionSet.Has(string(spec.ValidationFailureAction)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("validationFailureAction"), spec.ValidationFailureAction, validActions))
	}

	return allErrs
}
