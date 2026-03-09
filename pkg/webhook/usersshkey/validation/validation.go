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
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/webhook/util"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic UserSSHKey CRD.
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new user SSH key validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		client: client,
	}
}

var _ admission.Validator[*kubermaticv1.UserSSHKey] = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, key *kubermaticv1.UserSSHKey) (admission.Warnings, error) {
	errs := validation.ValidateUserSSHKeyCreate(key)

	if err := v.validateProjectRelationship(ctx, key, nil); err != nil {
		errs = append(errs, err)
	}

	return nil, errs.ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldKey, newKey *kubermaticv1.UserSSHKey) (admission.Warnings, error) {
	errs := validation.ValidateUserSSHKeyUpdate(oldKey, newKey)

	if err := v.validateProjectRelationship(ctx, newKey, oldKey); err != nil {
		errs = append(errs, err)
	}

	return nil, errs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.UserSSHKey) (admission.Warnings, error) {
	return nil, nil
}

func (v *validator) validateProjectRelationship(ctx context.Context, key *kubermaticv1.UserSSHKey, oldKey *kubermaticv1.UserSSHKey) *field.Error {
	if err := util.OptimisticallyCheckIfProjectIsValid(ctx, v.client, key.Spec.Project, oldKey != nil); err != nil {
		return field.Invalid(field.NewPath("spec", "project"), key.Spec.Project, err.Error())
	}

	return nil
}
