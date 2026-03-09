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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating GroupProjectBinding CRD.
type validator struct {
}

// NewValidator returns a new GroupProjectBinding validator.
func NewValidator() *validator {
	return &validator{}
}

var _ admission.Validator[*kubermaticv1.GroupProjectBinding] = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj *kubermaticv1.GroupProjectBinding) (admission.Warnings, error) {
	return nil, validateCreate(ctx, obj)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj *kubermaticv1.GroupProjectBinding) (admission.Warnings, error) {
	return nil, validateUpdate(ctx, oldObj, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.GroupProjectBinding) (admission.Warnings, error) {
	return nil, validateDelete(ctx, obj)
}
