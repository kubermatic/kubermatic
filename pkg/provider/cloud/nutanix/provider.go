/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package nutanix

import (
	"errors"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type Nutanix struct {
	dc                *kubermaticv1.DatacenterSpecNutanix
	secretKeySelector provider.SecretKeySelectorValueFunc
}

var _ provider.CloudProvider = &Nutanix{}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Nutanix, error) {
	if dc.Spec.Nutanix == nil {
		return nil, errors.New("datacenter is not a Nutanix datacenter")
	}

	return &Nutanix{
		dc:                dc.Spec.Nutanix,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func (n *Nutanix) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	return nil
}
func (n *Nutanix) InitializeCloudProvider(*kubermaticv1.Cluster, provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return nil, nil
}

func (n *Nutanix) CleanUpCloudProvider(*kubermaticv1.Cluster, provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return nil, nil
}

func (n *Nutanix) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (n *Nutanix) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

func (n *Nutanix) ReconcileCluster(*kubermaticv1.Cluster, provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return nil, nil
}
