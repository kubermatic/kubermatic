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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating PolicyInstance CRD
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new policy instance validator
func NewValidator(client ctrlruntimeclient.Client) admission.CustomValidator {
	return &validator{
		client: client,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	policyInstance, ok := obj.(*kubermaticv1.PolicyInstance)
	if !ok {
		return nil, errors.New("object is not a PolicyInstance")
	}
	return nil, validateCreate(ctx, policyInstance, v.client)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldPolicyInstance, ok := oldObj.(*kubermaticv1.PolicyInstance)
	if !ok {
		return nil, errors.New("old object is not a PolicyInstance")
	}

	newPolicyInstance, ok := newObj.(*kubermaticv1.PolicyInstance)
	if !ok {
		return nil, errors.New("new object is not a PolicyInstance")
	}

	return nil, validateUpdate(ctx, oldPolicyInstance, newPolicyInstance, v.client)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	policyInstance, ok := obj.(*kubermaticv1.PolicyInstance)
	if !ok {
		return nil, errors.New("object is not a PolicyInstance")
	}

	return nil, validateDelete(ctx, policyInstance, v.client)
}
