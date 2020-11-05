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
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ConstraintProvider struct that holds required components in order manage constraints
type ConstraintProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient impersonationClient
	clientPrivileged             ctrlruntimeclient.Client
}

// NewConstraintProvider returns a constraint provider
func NewConstraintProvider(createSeedImpersonatedClient impersonationClient, client ctrlruntimeclient.Client) (*ConstraintProvider, error) {
	return &ConstraintProvider{
		clientPrivileged:             client,
		createSeedImpersonatedClient: createSeedImpersonatedClient,
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

// Delete deletes a constraint
func (p *ConstraintProvider) Delete(cluster *kubermaticv1.Cluster, userInfo *provider.UserInfo, name string) error {

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	return impersonationClient.Delete(context.Background(), &kubermaticv1.Constraint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	})
}

// DeleteUnsecured deletes a constraint using a privileged client
func (p *ConstraintProvider) DeleteUnsecured(cluster *kubermaticv1.Cluster, name string) error {
	return p.clientPrivileged.Delete(context.Background(), &kubermaticv1.Constraint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	})
}

func (p *ConstraintProvider) Create(userInfo *provider.UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	err = impersonationClient.Create(context.Background(), constraint)
	return constraint, err
}

func (p *ConstraintProvider) CreateUnsecured(constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {

	err := p.clientPrivileged.Create(context.Background(), constraint)
	return constraint, err
}
