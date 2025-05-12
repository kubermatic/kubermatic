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
	"errors"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &validator{}

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

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj).ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldTemplate, ok := oldObj.(*kubermaticv1.PolicyTemplate)
	if !ok {
		return nil, errors.New("old object is not a PolicyTemplate")
	}

	newTemplate, ok := newObj.(*kubermaticv1.PolicyTemplate)
	if !ok {
		return nil, errors.New("new object is not a PolicyTemplate")
	}

	allErrs := v.validate(ctx, newObj)

	specPath := field.NewPath("spec")
	if oldTemplate.Spec.Visibility != newTemplate.Spec.Visibility {
		allErrs = append(allErrs, field.Invalid(specPath.Child("visibility"), newTemplate.Spec.Visibility, "visibility is immutable"))
	}
	if oldTemplate.Spec.ProjectID != newTemplate.Spec.ProjectID {
		allErrs = append(allErrs, field.Invalid(specPath.Child("projectID"), newTemplate.Spec.ProjectID, "projectID is immutable"))
	}

	return nil, allErrs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *validator) validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	template, ok := obj.(*kubermaticv1.PolicyTemplate)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), errors.New("object is not a PolicyTemplate"))}
	}

	err := validation.ValidatePolicyTemplate(template)

	return err
}
