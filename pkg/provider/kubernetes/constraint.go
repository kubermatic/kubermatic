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

package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ConstraintProvider struct that holds required components in order manage constraints
type ConstraintProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

// NewConstraintProvider returns a constraint provider
func NewConstraintProvider(client ctrlruntimeclient.Client) (*ConstraintProvider, error) {
	return &ConstraintProvider{
		clientPrivileged: client,
	}, nil
}

// List gets all constraints
func (p *ConstraintProvider) List(cluster *kubermaticv1.Cluster) (*kubermaticv1.ConstraintList, error) {
	constraints := &kubermaticv1.ConstraintList{}
	if err := p.clientPrivileged.List(context.Background(), constraints, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
		return nil, fmt.Errorf("failed to list constraints: %v", err)
	}

	return constraints, nil
}

// Get gets a constraint using a privileged client
func (p *ConstraintProvider) Get(cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.Constraint, error) {

	constraint := &kubermaticv1.Constraint{}
	if err := p.clientPrivileged.Get(context.Background(), types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: name}, constraint); err != nil {
		return nil, err
	}

	return constraint, nil
}
