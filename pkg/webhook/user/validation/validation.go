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
	"errors"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/webhook/util"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic User CRD.
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new user validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		client: client,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	user, ok := obj.(*kubermaticv1.User)
	if !ok {
		return errors.New("object is not a User")
	}

	errs := validation.ValidateUserCreate(user)

	if err := v.validateProjectRelationship(ctx, user, nil); err != nil {
		errs = append(errs, err)
	}

	return errs.ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	oldUser, ok := oldObj.(*kubermaticv1.User)
	if !ok {
		return errors.New("old object is not a User")
	}

	newUser, ok := newObj.(*kubermaticv1.User)
	if !ok {
		return errors.New("new object is not a User")
	}

	errs := validation.ValidateUserUpdate(oldUser, newUser)

	if err := v.validateProjectRelationship(ctx, newUser, oldUser); err != nil {
		errs = append(errs, err)
	}

	return errs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *validator) validateProjectRelationship(ctx context.Context, user *kubermaticv1.User, oldUser *kubermaticv1.User) *field.Error {
	if !kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email) {
		return nil
	}

	if err := util.OptimisticallyCheckIfProjectIsValid(ctx, v.client, user.Spec.Project, oldUser != nil); err != nil {
		return field.Invalid(field.NewPath("spec", "project"), user.Spec.Project, err.Error())
	}

	return nil
}
