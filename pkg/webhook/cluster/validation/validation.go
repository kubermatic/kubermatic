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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic Cluster CRD.
type validator struct {
	features   features.FeatureGate
	client     ctrlruntimeclient.Client
	seedGetter provider.SeedGetter
	caBundle   *x509.CertPool

	// disableProviderValidation is only for unit tests, to ensure no
	// provide would phone home to validate dummy test credentials
	disableProviderValidation bool
}

// NewValidator returns a new cluster validator.
func NewValidator(client ctrlruntimeclient.Client, seedGetter provider.SeedGetter, features features.FeatureGate, caBundle *x509.CertPool) *validator {
	return &validator{
		client:     client,
		features:   features,
		seedGetter: seedGetter,
		caBundle:   caBundle,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	cluster, ok := obj.(*kubermaticv1.Cluster)
	if !ok {
		return errors.New("object is not a Cluster")
	}

	datacenter, cloudProvider, err := v.buildValidationDependencies(ctx, cluster)
	if err != nil {
		return err
	}

	return validation.ValidateNewClusterSpec(ctx, &cluster.Spec, datacenter, cloudProvider, v.features, nil).ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	oldCluster, ok := oldObj.(*kubermaticv1.Cluster)
	if !ok {
		return errors.New("old object is not a Cluster")
	}

	newCluster, ok := newObj.(*kubermaticv1.Cluster)
	if !ok {
		return errors.New("new object is not a Cluster")
	}

	datacenter, cloudProvider, err := v.buildValidationDependencies(ctx, newCluster)
	if err != nil {
		return err
	}

	return validation.ValidateClusterUpdate(ctx, newCluster, oldCluster, datacenter, cloudProvider, v.features).ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *validator) buildValidationDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, provider.CloudProvider, *field.Error) {
	seed, err := v.seedGetter()
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}
	if seed == nil {
		return nil, nil, field.InternalError(nil, errors.New("webhook is not configured with -seed-name, cannot validate Clusters"))
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(&c.Spec, seed)
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
