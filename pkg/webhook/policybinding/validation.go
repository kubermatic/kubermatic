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

package policybinding

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &validatingHandler{}

// validatingHandler handles validating webhook requests for PolicyBinding resources
type validatingHandler struct {
	log              *zap.SugaredLogger
	client           ctrlruntimeclient.Client
	decoder          admission.Decoder
	seedGetter       provider.SeedGetter
	seedClientGetter provider.SeedClientGetter
}

func (h *validatingHandler) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	binding, ok := obj.(*kubermaticv1.PolicyBinding)
	if !ok {
		return nil, fmt.Errorf("expected PolicyBinding but got: %T", obj)
	}

	seedClient, err := h.getSeedClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed client: %w", err)
	}

	if errs := validation.ValidatePolicyBindingCreate(ctx, seedClient, binding); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}

	return nil, nil
}

func (h *validatingHandler) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	// We don't have any special validation for deletion
	return nil, nil
}

func (h *validatingHandler) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (warnings admission.Warnings, err error) {
	oldBinding, ok := oldObj.(*kubermaticv1.PolicyBinding)
	if !ok {
		return nil, fmt.Errorf("expected PolicyBinding but got: %T", oldObj)
	}

	newBinding, ok := newObj.(*kubermaticv1.PolicyBinding)
	if !ok {
		return nil, fmt.Errorf("expected PolicyBinding but got: %T", newObj)
	}

	seedClient, err := h.getSeedClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed client: %w", err)
	}

	if errs := validation.ValidatePolicyBindingUpdate(ctx, seedClient, oldBinding, newBinding); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}

	return nil, nil
}

// NewValidator creates a new validating webhook handler for PolicyBinding
func NewValidator(log *zap.SugaredLogger, client ctrlruntimeclient.Client, scheme *runtime.Scheme, seedGetter provider.SeedGetter, seedClientGetter provider.SeedClientGetter) *validatingHandler {
	return &validatingHandler{
		log:              log,
		client:           client,
		decoder:          admission.NewDecoder(scheme),
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

func (h *validatingHandler) getSeedClient(ctx context.Context) (ctrlruntimeclient.Client, error) {
	seed, err := h.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get current seed: %w", err)
	}
	if seed == nil {
		return nil, fmt.Errorf("webhook not configured for a seed cluster")
	}

	client, err := h.seedClientGetter(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed client: %w", err)
	}

	return client, nil
}
