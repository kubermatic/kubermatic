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
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/features"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/provider/cloud"
	"k8c.io/kubermatic/v3/pkg/validation"
	"k8c.io/kubermatic/v3/pkg/version"

	"k8s.io/apimachinery/pkg/runtime"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic Cluster CRD.
type validator struct {
	features         features.FeatureGate
	seedClient       ctrlruntimeclient.Client
	datacenterGetter provider.DatacenterGetter
	configGetter     provider.KubermaticConfigurationGetter
	caBundle         *x509.CertPool

	// disableProviderValidation is only for unit tests, to ensure no
	// provider would phone home to validate dummy test credentials
	disableProviderValidation bool
}

// NewValidator returns a new cluster validator.
func NewValidator(seedClient ctrlruntimeclient.Client, configGetter provider.KubermaticConfigurationGetter, datacenterGetter provider.DatacenterGetter, features features.FeatureGate, caBundle *x509.CertPool) *validator {
	return &validator{
		seedClient:       seedClient,
		features:         features,
		configGetter:     configGetter,
		datacenterGetter: datacenterGetter,
		caBundle:         caBundle,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	cluster, ok := obj.(*kubermaticv1.Cluster)
	if !ok {
		return errors.New("object is not a Cluster")
	}

	// This validates the charset and the max length.
	if errs := k8svalidation.IsDNS1035Label(cluster.Name); len(errs) != 0 {
		return fmt.Errorf("cluster name must be valid rfc1035 label: %s", strings.Join(errs, ","))
	}

	if len(cluster.Name) > validation.MaxClusterNameLength {
		return fmt.Errorf("cluster name exceeds maximum allowed length of %d characters", validation.MaxClusterNameLength)
	}

	datacenter, cloudProvider, err := v.buildValidationDependencies(ctx, cluster)
	if err != nil {
		return err
	}

	config, configErr := v.configGetter(ctx)
	if configErr != nil {
		return configErr
	}

	versionManager := version.NewFromConfiguration(config)
	errs := validation.ValidateNewClusterSpec(ctx, &cluster.Spec, datacenter, cloudProvider, versionManager, v.features, nil)

	return errs.ToAggregate()
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

	config, configErr := v.configGetter(ctx)
	if configErr != nil {
		return configErr
	}

	updateManager := version.NewFromConfiguration(config)
	errs := validation.ValidateClusterUpdate(ctx, newCluster, oldCluster, datacenter, cloudProvider, updateManager, v.features)

	return errs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *validator) buildValidationDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, provider.CloudProvider, *field.Error) {
	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(ctx, &c.Spec, v.datacenterGetter)
	if fieldErr != nil {
		return nil, nil, fieldErr
	}

	if v.disableProviderValidation {
		return datacenter, nil, nil
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, v.seedClient)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, v.caBundle)
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	return datacenter, cloudProvider, nil
}
