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
	"crypto/x509"
	"errors"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &validator{}

// validator for validating Kubermatic ClusterTemplate CRs.
type validator struct {
	features     features.FeatureGate
	client       ctrlruntimeclient.Client
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter
	caBundle     *x509.CertPool

	// disableProviderValidation is only for unit tests, to ensure no
	// provider would phone home to validate dummy test credentials
	disableProviderValidation bool
}

// NewValidator returns a new cluster template validator.
func NewValidator(client ctrlruntimeclient.Client, seedGetter provider.SeedGetter, seedClientGetter provider.SeedClientGetter, configGetter provider.KubermaticConfigurationGetter, features features.FeatureGate, caBundle *x509.CertPool) *validator {
	return &validator{
		client:       client,
		features:     features,
		seedGetter:   seedGetter,
		configGetter: configGetter,
		caBundle:     caBundle,
	}
}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return v.validate(ctx, obj)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return v.validate(ctx, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *validator) validate(ctx context.Context, obj runtime.Object) error {
	template, ok := obj.(*kubermaticv1.ClusterTemplate)
	if !ok {
		return errors.New("object is not a ClusterTemplate")
	}

	var errs field.ErrorList
	datacenter, cloudProvider, err := v.buildValidationDependencies(ctx, &template.Spec)
	if err != nil {
		return err
	}

	config, configErr := v.configGetter(ctx)
	if configErr != nil {
		return configErr
	}

	versionManager := version.NewFromConfiguration(config)
	errs = validation.ValidateClusterTemplate(ctx, template, datacenter, cloudProvider, v.features, versionManager, nil)

	return errs.ToAggregate()
}

func (v *validator) buildValidationDependencies(ctx context.Context, c *kubermaticv1.ClusterSpec) (*kubermaticv1.Datacenter, provider.CloudProvider, *field.Error) {
	seed, err := v.seedGetter()
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}
	if seed == nil {
		return nil, nil, field.InternalError(nil, errors.New("webhook is not configured with -seed-name, cannot validate Clusters"))
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(c, seed)
	if fieldErr != nil {
		return nil, nil, fieldErr
	}

	if v.disableProviderValidation {
		return datacenter, nil, nil
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, v.client)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, v.caBundle)
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	return datacenter, cloudProvider, nil
}
