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
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/webhook/util"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic User CRD.
type validator struct {
	client           ctrlruntimeclient.Client
	seedsGetter      provider.SeedsGetter
	seedClientGetter provider.SeedClientGetter
}

// NewValidator returns a new user validator.
func NewValidator(client ctrlruntimeclient.Client,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter) *validator {
	return &validator{
		client:           client,
		seedsGetter:      seedsGetter,
		seedClientGetter: seedClientGetter,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	user, ok := obj.(*kubermaticv1.User)
	if !ok {
		return nil, errors.New("object is not a User")
	}

	errs := validation.ValidateUserCreate(user)

	if err := v.validateProjectRelationship(ctx, user, nil); err != nil {
		errs = append(errs, err)
	}

	// Validate email uniqueness
	if err := validation.ValidateUserEmailUniqueness(ctx, v.client, user.Spec.Email, ""); err != nil {
		errs = append(errs, err)
	}

	return nil, errs.ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldUser, ok := oldObj.(*kubermaticv1.User)
	if !ok {
		return nil, errors.New("old object is not a User")
	}

	newUser, ok := newObj.(*kubermaticv1.User)
	if !ok {
		return nil, errors.New("new object is not a User")
	}

	errs := validation.ValidateUserUpdate(oldUser, newUser)

	if err := v.validateProjectRelationship(ctx, newUser, oldUser); err != nil {
		errs = append(errs, err)
	}

	if !strings.EqualFold(oldUser.Spec.Email, newUser.Spec.Email) {
		if err := validation.ValidateUserEmailUniqueness(ctx, v.client, newUser.Spec.Email, newUser.Name); err != nil {
			errs = append(errs, err)
		}
	}

	return nil, errs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	user, ok := obj.(*kubermaticv1.User)
	if !ok {
		return nil, errors.New("object is not a User")
	}

	return nil, validation.ValidateUserDelete(ctx, user, v.client, v.seedsGetter, v.seedClientGetter)
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
