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

package kubernetes

import (
	"context"

	"k8c.io/kubermatic/v2/pkg/provider"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatingSystemProfileProvider struct {
	privilegedClient ctrlruntimeclient.Client
}

var _ provider.OperatingSystemProfileProvider = &OperatingSystemProfileProvider{}

func NewOperatingSystemProfileProvider(privilegedClient ctrlruntimeclient.Client) *OperatingSystemProfileProvider {
	return &OperatingSystemProfileProvider{
		privilegedClient: privilegedClient,
	}
}

func (p *OperatingSystemProfileProvider) ListUnsecured(ctx context.Context, namespace string) (*osmv1alpha1.OperatingSystemProfileList, error) {
	ospList := &osmv1alpha1.OperatingSystemProfileList{}
	err := p.privilegedClient.List(ctx, ospList, ctrlruntimeclient.InNamespace(namespace))
	return ospList, err
}
