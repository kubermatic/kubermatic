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

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddonProvider struct that holds required components of the AddonProvider implementation
type AddonProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient ImpersonationClient
	// accessibleAddons is the set of addons that the provider should provide CRUD access to
	accessibleAddons sets.String
	// clientPrivileged is used for privileged operations
	clientPrivileged ctrlruntimeclient.Client
}

// NewAddonProvider returns a new addon provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation
func NewAddonProvider(
	clientPrivileged ctrlruntimeclient.Client,
	createSeedImpersonatedClient ImpersonationClient,
	accessibleAddons sets.String) *AddonProvider {
	return &AddonProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		accessibleAddons:             accessibleAddons,
		clientPrivileged:             clientPrivileged,
	}
}

// New creates a new addon in the given cluster
func (p *AddonProvider) New(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	addon := genAddon(cluster, addonName, variables, labels)

	if err = seedImpersonatedClient.Create(context.Background(), addon); err != nil {
		return nil, err
	}

	return addon, nil
}

// NewUnsecured creates a new addon in the given cluster
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resource
func (p *AddonProvider) NewUnsecured(cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	addon := genAddon(cluster, addonName, variables, labels)

	if err := p.clientPrivileged.Create(context.Background(), addon); err != nil {
		return nil, err
	}

	return addon, nil
}

func genAddon(cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) *kubermaticv1.Addon {
	gv := kubermaticv1.SchemeGroupVersion
	if labels == nil {
		labels = map[string]string{}
	}
	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:            addonName,
			Namespace:       cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))},
			Labels:          labels,
		},
		Spec: kubermaticv1.AddonSpec{
			Name: addonName,
			Cluster: v1.ObjectReference{
				Name:       cluster.Name,
				Namespace:  "",
				UID:        cluster.UID,
				APIVersion: cluster.APIVersion,
				Kind:       "Cluster",
			},
			Variables: *variables,
		},
	}
}

// Get returns the given addon, it uses the projectInternalName to determine the group the user belongs to
func (p *AddonProvider) Get(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	addon := &kubermaticv1.Addon{}
	if err := seedImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: cluster.Status.NamespaceName, Name: addonName}, addon); err != nil {
		return nil, err
	}
	return addon, nil
}

// GetUnsecured returns the given addon
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
func (p *AddonProvider) GetUnsecured(cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}
	addon := &kubermaticv1.Addon{}
	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: cluster.Status.NamespaceName, Name: addonName}, addon); err != nil {
		return nil, err
	}
	return addon, nil
}

// List returns all addons in the given cluster
func (p *AddonProvider) List(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error) {
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	addonList := &kubermaticv1.AddonList{}
	if err := seedImpersonatedClient.List(context.Background(), addonList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
		return nil, err
	}

	result := []*kubermaticv1.Addon{}
	for _, addon := range addonList.Items {
		if p.accessibleAddons.Has(addon.Name) {
			result = append(result, addon.DeepCopy())
		}
	}

	return result, nil
}

func (p *AddonProvider) ListUnsecured(cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error) {

	addonList := &kubermaticv1.AddonList{}
	if err := p.clientPrivileged.List(context.Background(), addonList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
		return nil, err
	}

	result := []*kubermaticv1.Addon{}
	for _, addon := range addonList.Items {
		if p.accessibleAddons.Has(addon.Name) {
			result = append(result, addon.DeepCopy())
		}
	}

	return result, nil
}

// Update updates an addon
func (p *AddonProvider) Update(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, addon *kubermaticv1.Addon) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addon.Name) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addon.Name))
	}

	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	addon.Namespace = cluster.Status.NamespaceName
	if err := seedImpersonatedClient.Update(context.Background(), addon); err != nil {
		return nil, err
	}

	return addon, nil
}

// Delete deletes the given addon
func (p *AddonProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, addonName string) error {
	if !p.accessibleAddons.Has(addonName) {
		return kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	return seedImpersonatedClient.Delete(context.Background(), &kubermaticv1.Addon{ObjectMeta: metav1.ObjectMeta{Name: addonName, Namespace: cluster.Status.NamespaceName}})
}

// UpdateUnsecured updates an addon
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to update the resource
func (p *AddonProvider) UpdateUnsecured(cluster *kubermaticv1.Cluster, addon *kubermaticv1.Addon) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addon.Name) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addon.Name))
	}

	addon.Namespace = cluster.Status.NamespaceName
	if err := p.clientPrivileged.Update(context.Background(), addon); err != nil {
		return nil, err
	}

	return addon, nil
}

// DeleteUnsecured deletes the given addon
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to delete the resource
func (p *AddonProvider) DeleteUnsecured(cluster *kubermaticv1.Cluster, addonName string) error {
	if !p.accessibleAddons.Has(addonName) {
		return kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	return p.clientPrivileged.Delete(context.Background(), &kubermaticv1.Addon{ObjectMeta: metav1.ObjectMeta{Name: addonName, Namespace: cluster.Status.NamespaceName}})
}

func AddonProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter, accessibleAddons sets.String) provider.AddonProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.AddonProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewImpersonationClient(cfg, mapper)
		clientPrivileged, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewAddonProvider(
			clientPrivileged,
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
			accessibleAddons,
		), nil
	}
}
