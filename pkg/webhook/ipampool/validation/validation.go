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
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating IPAMPool CRD.
type validator struct {
	ipamPoolValidator *validation.IPAMPoolvalidator
}

// NewValidator returns a new IPAMPool validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		ipamPoolValidator: validation.NewIPAMPoolValidator(client),
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	ipamPool, ok := obj.(*kubermaticv1.IPAMPool)
	if !ok {
		return errors.New("object is not a IPAMPool")
	}
	return v.ipamPoolValidator.ValidateCreate(ctx, ipamPool)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	oldIPAMPool, ok := oldObj.(*kubermaticv1.IPAMPool)
	if !ok {
		return errors.New("old object is not a IPAMPool")
	}
	newIPAMPool, ok := newObj.(*kubermaticv1.IPAMPool)
	if !ok {
		return errors.New("new object is not a IPAMPool")
	}
	return v.ipamPoolValidator.ValidateUpdate(ctx, oldIPAMPool, newIPAMPool)
}

func (v *validator) ValidateDelete(_ context.Context, _ runtime.Object) error {
	// NOP we allow delete operation
	return nil
}
