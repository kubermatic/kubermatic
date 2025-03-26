//go:build !ee

/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package seed

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type fixedNameValidator struct {
	upstream *validator
	name     string
}

var _ admission.CustomValidator = &fixedNameValidator{}

func NewValidator(
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	features features.FeatureGate,
) (*fixedNameValidator, error) {
	upstream, err := newSeedValidator(seedsGetter, seedClientGetter, features)
	if err != nil {
		return nil, err
	}

	return &fixedNameValidator{
		upstream: upstream,
		name:     provider.DefaultSeedName,
	}, nil
}

func (v *fixedNameValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj, false)
}

func (v *fixedNameValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, newObj, false)
}

func (v *fixedNameValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj, true)
}

func (v *fixedNameValidator) validate(ctx context.Context, obj runtime.Object, isDelete bool) error {
	// restrict names to CE-compatible names, i.e. only "kubermatic";
	// this is both to make the validation easier (we can use the default
	// seedsGetter if there cannot be more than one Seed in CE) and to make
	// misconfiguration harder (we warn the user early if they create misnamed Seeds)
	if !isDelete {
		subject, ok := obj.(*kubermaticv1.Seed)
		if !ok {
			return errors.New("given object is not a Seed")
		}

		if subject.Name != v.name {
			return fmt.Errorf("cannot create Seed %s: it must be named %s", subject.Name, v.name)
		}
	}

	return v.upstream.validate(ctx, obj, isDelete)
}
