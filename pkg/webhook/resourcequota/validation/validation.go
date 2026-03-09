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

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Resource Quota CRD.
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new Resource Quota validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		client: client,
	}
}

var _ admission.Validator[*kubermaticv1.ResourceQuota] = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	return nil, validateCreate(ctx, obj, v.client)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	return nil, validateUpdate(ctx, oldObj, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	return nil, validateDelete(ctx, obj, v.client)
}
