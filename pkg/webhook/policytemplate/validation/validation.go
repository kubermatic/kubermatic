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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.Validator[*kubermaticv1.PolicyTemplate] = &validator{}

// validator for validating Kubermatic PolicyTemplate CRs.
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new policy template validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		client: client,
	}
}

func (v *validator) ValidateCreate(ctx context.Context, obj *kubermaticv1.PolicyTemplate) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj).ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldTemplate, newTemplate *kubermaticv1.PolicyTemplate) (admission.Warnings, error) {
	allErrs := v.validate(ctx, newTemplate)

	specPath := field.NewPath("spec")
	if oldTemplate.Spec.Visibility != newTemplate.Spec.Visibility {
		allErrs = append(allErrs, field.Invalid(specPath.Child("visibility"), newTemplate.Spec.Visibility, "visibility is immutable"))
	}
	if oldTemplate.Spec.ProjectID != newTemplate.Spec.ProjectID {
		allErrs = append(allErrs, field.Invalid(specPath.Child("projectID"), newTemplate.Spec.ProjectID, "projectID is immutable"))
	}

	return nil, allErrs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.PolicyTemplate) (admission.Warnings, error) {
	return nil, nil
}

func (v *validator) validate(ctx context.Context, template *kubermaticv1.PolicyTemplate) field.ErrorList {
	err := validation.ValidatePolicyTemplate(template)

	return err
}
