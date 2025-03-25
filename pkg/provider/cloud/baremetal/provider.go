/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package baremetal

import (
	"context"
	"encoding/base64"
	"errors"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/client-go/tools/clientcmd"
)

type baremetal struct{}

// NewCloudProvider creates a new baremetal provider.
func NewCloudProvider() provider.CloudProvider {
	return &baremetal{}
}

var _ provider.CloudProvider = &baremetal{}

func (b *baremetal) DefaultCloudSpec(_ context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.Baremetal == nil {
		return errors.New("baremetal cloud provider spec is empty")
	}

	if spec.Cloud.Baremetal.Tinkerbell == nil {
		return errors.New("tinkerbell spec is empty! tinkerbell spec is required")
	}
	return nil
}

func (b *baremetal) ValidateCloudSpec(_ context.Context, spec kubermaticv1.CloudSpec) error {
	if spec.Baremetal.Tinkerbell != nil {
		kubeconfig := spec.Baremetal.Tinkerbell.Kubeconfig
		if kubeconfig == "" {
			return errors.New("tinkerbell kubeconfig is empty")
		}
		// Tinkerbell kubeconfig is encoded in base64
		config, err := base64.StdEncoding.DecodeString(kubeconfig)
		if err != nil {
			return err
		}

		_, err = clientcmd.RESTConfigFromKubeConfig(config)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *baremetal) InitializeCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (b *baremetal) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (b *baremetal) CleanUpCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (b *baremetal) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}
