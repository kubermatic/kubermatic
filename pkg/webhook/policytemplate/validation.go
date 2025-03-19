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

package policytemplate

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator validates PolicyTemplates.
type validator struct {
	log              *zap.SugaredLogger
	client           ctrlruntimeclient.Client
	scheme           *runtime.Scheme
	seedGetter       func() (*kubermaticv1.Seed, error)
	seedClientGetter func(*kubermaticv1.Seed) (ctrlruntimeclient.Client, error)
}

// NewValidator returns a new PolicyTemplate validator.
func NewValidator(
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	scheme *runtime.Scheme,
	seedGetter func() (*kubermaticv1.Seed, error),
	seedClientGetter func(*kubermaticv1.Seed) (ctrlruntimeclient.Client, error),
) *validator {
	return &validator{
		log:              log,
		client:           client,
		scheme:           scheme,
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	template := obj.(*kubermaticv1.PolicyTemplate)

	if errs := validation.ValidatePolicyTemplateCreate(ctx, v.client, template); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}

	return nil, nil
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldTemplate := oldObj.(*kubermaticv1.PolicyTemplate)
	newTemplate := newObj.(*kubermaticv1.PolicyTemplate)

	if errs := validation.ValidatePolicyTemplateUpdate(ctx, v.client, oldTemplate, newTemplate); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}

	return nil, nil
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
