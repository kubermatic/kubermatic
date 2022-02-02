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

package kubernetes

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SeedProvider struct that holds required components in order seeds.
type SeedProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

func NewSeedProvider(client ctrlruntimeclient.Client) *SeedProvider {
	return &SeedProvider{
		clientPrivileged: client,
	}
}

func (p *SeedProvider) UpdateUnsecured(seed *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
	if err := p.clientPrivileged.Update(context.Background(), seed); err != nil {
		return nil, err
	}
	return seed, nil
}
