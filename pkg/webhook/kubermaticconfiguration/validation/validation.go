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
	"fmt"
	"slices"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type validator struct {
	seedsGetter provider.SeedsGetter
}

func NewValidator(seedsGetter provider.SeedsGetter) *validator {
	return &validator{
		seedsGetter: seedsGetter,
	}
}

var _ admission.Validator[*kubermaticv1.KubermaticConfiguration] = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, config *kubermaticv1.KubermaticConfiguration) (admission.Warnings, error) {
	return nil, v.validate(nil, config)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj *kubermaticv1.KubermaticConfiguration) (admission.Warnings, error) {
	return nil, v.validate(oldObj, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.KubermaticConfiguration) (admission.Warnings, error) {
	return nil, nil
}

func (v *validator) validate(oldConfig, newConfig *kubermaticv1.KubermaticConfiguration) error {
	defaulted, err := defaulting.DefaultConfiguration(newConfig, zap.NewNop().Sugar())
	if err != nil {
		return fmt.Errorf("failed to apply default values: %w", err)
	}

	allErrs := validation.ValidateKubermaticConfigurationSpec(&defaulted.Spec)

	fieldErr, err := v.validateLoginClient(oldConfig, defaulted)
	if err != nil {
		return err
	}
	if fieldErr != nil {
		allErrs = append(allErrs, fieldErr)
	}

	return allErrs.ToAggregate()
}

// validateLoginClient rejects a login client that a Seed also uses as its
// User Cluster client. On updates the check only runs when the effective login
// client changes, so that unrelated changes to an existing configuration are
// not blocked.
func (v *validator) validateLoginClient(oldConfig, defaulted *kubermaticv1.KubermaticConfiguration) (*field.Error, error) {
	loginClient := validation.SeparateLoginClientID(&defaulted.Spec)
	if loginClient == "" {
		return nil, nil
	}

	if oldConfig != nil {
		oldDefaulted, err := defaulting.DefaultConfiguration(oldConfig, zap.NewNop().Sugar())
		if err != nil {
			return nil, fmt.Errorf("failed to apply default values to the existing KubermaticConfiguration: %w", err)
		}

		if validation.SeparateLoginClientID(&oldDefaulted.Spec) == loginClient {
			return nil, nil
		}
	}

	seeds, err := v.seedsGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to list Seeds: %w", err)
	}

	var conflicting []string
	for name, seed := range seeds {
		if oidc := seed.Spec.OIDCProviderConfiguration; oidc != nil && oidc.IssuerClientID == loginClient {
			conflicting = append(conflicting, name)
		}
	}

	if len(conflicting) == 0 {
		return nil, nil
	}

	slices.Sort(conflicting)

	return field.Invalid(
		field.NewPath("spec", "auth", "clientID"),
		loginClient,
		fmt.Sprintf("must differ from spec.oidcProviderConfiguration.issuerClientID of Seed(s) %s", strings.Join(conflicting, ", ")),
	), nil
}
