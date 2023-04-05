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

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	clustermutation "k8c.io/kubermatic/v3/pkg/mutation/cluster"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/provider/cloud"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Mutator for mutating Kubermatic Cluster CRD.
type Mutator struct {
	seedClient       ctrlruntimeclient.Client
	datacenterGetter provider.DatacenterGetter
	configGetter     provider.KubermaticConfigurationGetter
	caBundle         *x509.CertPool

	// disableProviderMutation is only for unit tests, to ensure no
	// provider would phone home to validate dummy test credentials
	disableProviderMutation bool
}

// NewAdmissionHandler returns a new cluster AdmissionHandler.
func NewMutator(client ctrlruntimeclient.Client, configGetter provider.KubermaticConfigurationGetter, datacenterGetter provider.DatacenterGetter, caBundle *x509.CertPool) *Mutator {
	return &Mutator{
		seedClient:       client,
		configGetter:     configGetter,
		datacenterGetter: datacenterGetter,
		caBundle:         caBundle,
	}
}

func (m *Mutator) Mutate(ctx context.Context, oldCluster, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, *field.Error) {
	// do not perform mutations on clusters in deletion
	if newCluster.DeletionTimestamp != nil {
		return newCluster, nil
	}

	provider, datacenter, fieldErr := m.determineCloudProvider(ctx, newCluster)
	if fieldErr != nil {
		return nil, fieldErr
	}

	config, err := m.configGetter(ctx)
	if err != nil {
		return nil, field.InternalError(nil, err)
	}

	// apply defaults to the existing clusters
	defaultTemplate, err := defaulting.GetDefaultingClusterTemplate(ctx, m.seedClient, config)
	if err != nil {
		return nil, field.InternalError(nil, err)
	}

	if err := defaulting.DefaultClusterSpec(ctx, &newCluster.Spec, defaultTemplate, config, m.datacenterGetter, provider); err != nil {
		return nil, field.InternalError(nil, err)
	}

	// perform operation-dependent mutations
	if oldCluster == nil {
		fieldErr = clustermutation.MutateCreate(newCluster, config, datacenter, provider)
	} else {
		fieldErr = clustermutation.MutateUpdate(oldCluster, newCluster, config, datacenter, provider)
	}

	return newCluster, fieldErr
}

func (m *Mutator) determineCloudProvider(ctx context.Context, c *kubermaticv1.Cluster) (provider.CloudProvider, *kubermaticv1.Datacenter, *field.Error) {
	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(ctx, &c.Spec, m.datacenterGetter)
	if fieldErr != nil {
		return nil, nil, fieldErr
	}

	if m.disableProviderMutation {
		return nil, datacenter, nil
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, m.seedClient)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, m.caBundle)
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	return cloudProvider, datacenter, nil
}
