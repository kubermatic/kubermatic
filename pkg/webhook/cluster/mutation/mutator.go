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

package mutation

import (
	"context"
	"crypto/x509"
	"errors"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	clustermutation "k8c.io/kubermatic/v2/pkg/mutation/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Mutator for mutating Kubermatic Cluster CRD.
type Mutator struct {
	client       ctrlruntimeclient.Client
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter
	caBundle     *x509.CertPool

	// disableProviderMutation is only for unit tests, to ensure no
	// provider would phone home to validate dummy test credentials
	disableProviderMutation bool
}

// NewAdmissionHandler returns a new cluster AdmissionHandler.
func NewMutator(client ctrlruntimeclient.Client, configGetter provider.KubermaticConfigurationGetter, seedGetter provider.SeedGetter, caBundle *x509.CertPool) *Mutator {
	return &Mutator{
		client:       client,
		configGetter: configGetter,
		seedGetter:   seedGetter,
		caBundle:     caBundle,
	}
}

func (m *Mutator) Mutate(ctx context.Context, oldCluster, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, *field.Error) {
	// do not perform mutations on clusters in deletion
	if newCluster.DeletionTimestamp != nil {
		return newCluster, nil
	}

	seed, provider, fieldErr := m.buildDefaultingDependencies(ctx, newCluster)
	if fieldErr != nil {
		return nil, fieldErr
	}

	config, err := m.configGetter(ctx)
	if err != nil {
		return nil, field.InternalError(nil, err)
	}

	// apply defaults to the existing clusters
	defaultTemplate, err := defaulting.GetDefaultingClusterTemplate(ctx, m.client, seed)
	if err != nil {
		return nil, field.InternalError(nil, err)
	}

	if err := defaulting.DefaultClusterSpec(ctx, &newCluster.Spec, newCluster.Annotations, defaultTemplate, seed, config, provider); err != nil {
		return nil, field.InternalError(nil, err)
	}

	// perform operation-dependent mutations
	if oldCluster == nil {
		fieldErr = clustermutation.MutateCreate(newCluster, config, seed, provider)
	} else {
		fieldErr = clustermutation.MutateUpdate(oldCluster, newCluster, config, seed, provider)
	}

	return newCluster, fieldErr
}

func (m *Mutator) buildDefaultingDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Seed, provider.CloudProvider, *field.Error) {
	seed, err := m.seedGetter()
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}
	if seed == nil {
		return nil, nil, field.InternalError(nil, errors.New("webhook is not configured with -seed-name, cannot validate Clusters"))
	}

	if m.disableProviderMutation {
		return seed, nil, nil
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(&c.Spec, seed)
	if fieldErr != nil {
		return nil, nil, fieldErr
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, m.client)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, m.caBundle)
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	return seed, cloudProvider, nil
}
