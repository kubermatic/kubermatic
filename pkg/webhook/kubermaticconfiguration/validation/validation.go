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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic KubermaticConfiguration CRD.
type validator struct {
}

// NewValidator returns a new KubermaticConfiguration validator.
func NewValidator() *validator {
	return &validator{}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	config, ok := obj.(*kubermaticv1.KubermaticConfiguration)
	if !ok {
		return nil, errors.New("object is not a KubermaticConfiguration")
	}

	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to apply default values: %w", err)
	}

	return nil, validation.ValidateKubermaticConfigurationSpec(&defaulted.Spec).ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	config, ok := newObj.(*kubermaticv1.KubermaticConfiguration)
	if !ok {
		return nil, errors.New("new object is not a KubermaticConfiguration")
	}

	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to apply default values: %w", err)
	}

	return nil, validation.ValidateKubermaticConfigurationSpec(&defaulted.Spec).ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
