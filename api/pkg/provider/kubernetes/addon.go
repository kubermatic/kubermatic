package kubernetes

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddonProvider struct that holds required components of the AddonProvider implementation
type AddonProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient kubermaticImpersonationClient
	// accessibleAddons is the set of addons that the provider should provide CRUD access to
	accessibleAddons sets.String
}

// NewAddonProvider returns a new addon provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation
func NewAddonProvider(
	createSeedImpersonatedClient kubermaticImpersonationClient,
	accessibleAddons sets.String) *AddonProvider {
	return &AddonProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		accessibleAddons:             accessibleAddons,
	}
}

// New creates a new addon in the given cluster
func (p *AddonProvider) New(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	addon := genAddon(cluster, addonName, variables)

	addon, err = seedImpersonatedClient.Addons(cluster.Status.NamespaceName).Create(addon)
	if err != nil {
		return nil, err
	}

	return addon, nil
}

// NewUnsecured creates a new addon in the given cluster
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resource
func (p *AddonProvider) NewUnsecured(cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	client, err := p.createSeedImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	addon := genAddon(cluster, addonName, variables)

	addon, err = client.Addons(cluster.Status.NamespaceName).Create(addon)
	if err != nil {
		return nil, err
	}

	return addon, nil
}

func genAddon(cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension) *kubermaticv1.Addon {
	gv := kubermaticv1.SchemeGroupVersion
	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:            addonName,
			Namespace:       cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))},
			Labels:          map[string]string{},
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

	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return seedImpersonatedClient.Addons(cluster.Status.NamespaceName).Get(addonName, metav1.GetOptions{})
}

// GetUnsecured returns the given addon
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
func (p *AddonProvider) GetUnsecured(cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addonName) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	client, err := p.createSeedImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	return client.Addons(cluster.Status.NamespaceName).Get(addonName, metav1.GetOptions{})
}

// List returns all addons in the given cluster
func (p *AddonProvider) List(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error) {
	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	addonList, err := seedImpersonatedClient.Addons(cluster.Status.NamespaceName).List(metav1.ListOptions{})
	if err != nil {
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
	client, err := p.createSeedImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	addonList, err := client.Addons(cluster.Status.NamespaceName).List(metav1.ListOptions{})
	if err != nil {
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

	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	addon, err = seedImpersonatedClient.Addons(cluster.Status.NamespaceName).Update(addon)
	if err != nil {
		return nil, err
	}

	return addon, nil
}

// Delete deletes the given addon
func (p *AddonProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, addonName string) error {
	if !p.accessibleAddons.Has(addonName) {
		return kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addonName))
	}

	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	return seedImpersonatedClient.Addons(cluster.Status.NamespaceName).Delete(addonName, &metav1.DeleteOptions{})
}

// UpdateUnsecured updates an addon
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to update the resource
func (p *AddonProvider) UpdateUnsecured(cluster *kubermaticv1.Cluster, addon *kubermaticv1.Addon) (*kubermaticv1.Addon, error) {
	if !p.accessibleAddons.Has(addon.Name) {
		return nil, kerrors.NewUnauthorized(fmt.Sprintf("addon not accessible: %v", addon.Name))
	}

	client, err := p.createSeedImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	addon, err = client.Addons(cluster.Status.NamespaceName).Update(addon)
	if err != nil {
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

	client, err := p.createSeedImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return err
	}

	return client.Addons(cluster.Status.NamespaceName).Delete(addonName, &metav1.DeleteOptions{})
}

func AddonProviderFactory(seedKubeconfigGetter provider.SeedKubeconfigGetter, accessibleAddons sets.String) provider.AddonProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.AddonProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewKubermaticImpersonationClient(cfg)

		return NewAddonProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedKubermaticClientSet,
			accessibleAddons,
		), nil
	}
}
