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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating GroupProjectBinding CRD.
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new GroupProjectBinding validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		client: client,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	groupProjectBinding, ok := obj.(*kubermaticv1.GroupProjectBinding)
	if !ok {
		return nil, errors.New("new object is not a GroupProjectBinding")
	}

	return nil, validateCreate(ctx, groupProjectBinding, v.client)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldGroupProjectBinding, ok := oldObj.(*kubermaticv1.GroupProjectBinding)
	if !ok {
		return nil, errors.New("existing object is not a GroupProjectBinding")
	}

	newGroupProjectBinding, ok := newObj.(*kubermaticv1.GroupProjectBinding)
	if !ok {
		return nil, errors.New("updated object is not a GroupProjectBinding")
	}

	return nil, validateUpdate(ctx, oldGroupProjectBinding, newGroupProjectBinding, v.client)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, validateDelete(ctx, obj)
}
