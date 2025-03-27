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

package vmwareclouddirector

import (
	"context"
	"errors"
	"fmt"

	"github.com/vmware/go-vcloud-director/v2/govcd"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	vappFinalizer = "kubermatic.k8c.io/cleanup-vmware-cloud-director-vapp"
)

type Provider struct {
	dc                *kubermaticv1.DatacenterSpecVMwareCloudDirector
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new VMware Cloud Director provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Provider, error) {
	if dc.Spec.VMwareCloudDirector == nil {
		return nil, errors.New("datacenter is not a VMware Cloud Director datacenter")
	}

	return &Provider{
		secretKeySelector: secretKeyGetter,
		dc:                dc.Spec.VMwareCloudDirector,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &Provider{}

func (p *Provider) DefaultCloudSpec(_ context.Context, _ *kubermaticv1.ClusterSpec) error {
	return nil
}

func (p *Provider) ValidateCloudSpec(_ context.Context, spec kubermaticv1.CloudSpec) error {
	if spec.VMwareCloudDirector == nil {
		return errors.New("not a VMware Cloud Director spec")
	}

	// vApp will be created and managed by the controller. We don't allow end-users to specify this field at the time
	// of cluster creation.
	if spec.VMwareCloudDirector.VApp != "" {
		return fmt.Errorf("vApp should not be set on cluster creation: %s", spec.VMwareCloudDirector.VApp)
	}

	// Validate credentials.
	client, err := NewClient(spec, p.secretKeySelector, p.dc)
	if err != nil {
		return fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	// Ensure that Organization exists.
	org, err := client.GetOrganization()
	if err != nil {
		return fmt.Errorf("failed to get organization %s: %w", client.Auth.Organization, err)
	}

	// Ensure that VDC exists.
	vdc, err := org.GetVDCByNameOrId(client.Auth.VDC, false)
	if err != nil {
		return fmt.Errorf("failed to get organization VDC '%s': %w", client.Auth.VDC, err)
	}

	// Ensure that the network exists
	if spec.VMwareCloudDirector.OVDCNetwork != "" || spec.VMwareCloudDirector.OVDCNetworks != nil {
		_, err := getOrgVDCNetworks(vdc, *spec.VMwareCloudDirector)
		if err != nil {
			return fmt.Errorf("failed to get organization VDC networks '%s': %w", client.Auth.VDC, err)
		}
	}

	return nil
}

func (p *Provider) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return p.reconcileCluster(ctx, cluster, update, false)
}

func (*Provider) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (p *Provider) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return p.reconcileCluster(ctx, cluster, update, true)
}

func (p *Provider) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// Cleanup is not required if finalizer was not present.
	if !kuberneteshelper.HasFinalizer(cluster, vappFinalizer) {
		return nil, nil
	}

	// Create an authenticated client.
	client, err := NewClient(cluster.Spec.Cloud, p.secretKeySelector, p.dc)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	org, err := client.GetOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", client.Auth.Organization, err)
	}

	vdc, err := client.GetVDCForOrg(*org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization VDC '%s': %w", client.Auth.VDC, err)
	}

	vAppName := cluster.Spec.Cloud.VMwareCloudDirector.VApp
	if vAppName == "" {
		vAppName = fmt.Sprintf(ResourceNamePattern, cluster.Name)
	}

	vapp, err := vdc.GetVAppByNameOrId(vAppName, true)
	if err != nil && !errors.Is(err, govcd.ErrorEntityNotFound) {
		return nil, fmt.Errorf("failed to get vApp '%s': %w", vAppName, err)
	}

	// We need to delete the vApp
	if err == nil {
		err := deleteVApp(vdc, vapp)
		if err != nil {
			return nil, fmt.Errorf("unable to delete vApp: %w", err)
		}
	}

	// vApp has been removed at this point. We need to cleanup the finalizer
	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, vappFinalizer)
	})
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (p *Provider) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.VMwareCloudDirector == nil || newSpec.VMwareCloudDirector == nil {
		return errors.New("'VMwareCloudDirector' spec is empty")
	}

	if oldSpec.VMwareCloudDirector.OVDCNetwork != newSpec.VMwareCloudDirector.OVDCNetwork {
		return fmt.Errorf("updating VMware Cloud Director OVDCNetwork is not supported (was %s, updated to %s)", oldSpec.VMwareCloudDirector.OVDCNetwork, newSpec.VMwareCloudDirector.OVDCNetwork)
	}

	if oldSpec.VMwareCloudDirector.OVDCNetworks != nil && newSpec.VMwareCloudDirector.OVDCNetworks != nil {
		diff := sets.NewString(oldSpec.VMwareCloudDirector.OVDCNetworks...).Difference(sets.NewString(newSpec.VMwareCloudDirector.OVDCNetworks...))
		if diff.Len() > 0 {
			return fmt.Errorf("updating VMware Cloud Director OVDCNetworks is not supported (was %s, updated to %s)", oldSpec.VMwareCloudDirector.OVDCNetworks, newSpec.VMwareCloudDirector.OVDCNetworks)
		}
	}

	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (creds *resources.VMwareCloudDirectorCredentials, err error) {
	username := cloud.VMwareCloudDirector.Username
	password := cloud.VMwareCloudDirector.Password
	apiToken := cloud.VMwareCloudDirector.APIToken
	organization := cloud.VMwareCloudDirector.Organization
	vdc := cloud.VMwareCloudDirector.VDC

	if organization == "" {
		if cloud.VMwareCloudDirector.CredentialsReference == nil {
			return nil, errors.New("no credentials provided")
		}
		organization, err = secretKeySelector(cloud.VMwareCloudDirector.CredentialsReference, resources.VMwareCloudDirectorOrganization)
		if err != nil {
			return nil, err
		}
	}

	if vdc == "" {
		if cloud.VMwareCloudDirector.CredentialsReference == nil {
			return nil, errors.New("no credentials provided")
		}
		vdc, err = secretKeySelector(cloud.VMwareCloudDirector.CredentialsReference, resources.VMwareCloudDirectorVDC)
		if err != nil {
			return nil, err
		}
	}

	// Check if API Token exists.
	if apiToken == "" && cloud.VMwareCloudDirector.CredentialsReference != nil {
		apiToken, _ = secretKeySelector(cloud.VMwareCloudDirector.CredentialsReference, resources.VMwareCloudDirectorAPIToken)
	}
	if apiToken != "" {
		return &resources.VMwareCloudDirectorCredentials{
			Organization: organization,
			APIToken:     apiToken,
			VDC:          vdc,
		}, nil
	}

	// Check for Username/password since API token doesn't exist.
	if username == "" {
		if cloud.VMwareCloudDirector.CredentialsReference == nil {
			return nil, errors.New("no credentials provided")
		}
		username, err = secretKeySelector(cloud.VMwareCloudDirector.CredentialsReference, resources.VMwareCloudDirectorUsername)
		if err != nil {
			return nil, err
		}
	}

	if password == "" {
		if cloud.VMwareCloudDirector.CredentialsReference == nil {
			return nil, errors.New("no credentials provided")
		}
		password, err = secretKeySelector(cloud.VMwareCloudDirector.CredentialsReference, resources.VMwareCloudDirectorPassword)
		if err != nil {
			return nil, err
		}
	}

	return &resources.VMwareCloudDirectorCredentials{
		Username:     username,
		Password:     password,
		Organization: organization,
		VDC:          vdc,
	}, nil
}

func (p *Provider) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool) (*kubermaticv1.Cluster, error) {
	// Create an authenticated client.
	client, err := NewClient(cluster.Spec.Cloud, p.secretKeySelector, p.dc)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	// Ensure that Organization exists.
	org, err := client.GetOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", client.Auth.Organization, err)
	}

	// Ensure that VDC exists.
	vdc, err := client.GetVDCForOrg(*org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization VDC '%s': %w", client.Auth.VDC, err)
	}

	// 1. Create the vApp, if it doesn't exist.
	cluster, err = reconcileVApp(ctx, cluster, update, vdc)
	if err != nil {
		return nil, err
	}

	// 2. Configure the required networks for vApp
	cluster, err = reconcileNetwork(ctx, cluster, update, vdc)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func ValidateCredentials(ctx context.Context, dc *kubermaticv1.DatacenterSpecVMwareCloudDirector, username, password, apiToken, organization, vdc string) error {
	client, err := NewClientWithCreds(username, password, apiToken, organization, vdc, dc.URL, dc.AllowInsecure)
	if err != nil {
		return fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	// We need to validate VDC as well since we scope everything down to VDC.
	// It's not consumed in the credentials directly.
	org, err := client.GetOrganization()
	if err != nil {
		return fmt.Errorf("failed to get organization %s: %w", client.Auth.Organization, err)
	}

	_, err = client.GetVDCForOrg(*org)
	if err != nil {
		return fmt.Errorf("failed to get organization VDC '%s': %w", vdc, err)
	}

	return err
}

func ListCatalogs(ctx context.Context, auth Auth) (apiv1.VMwareCloudDirectorCatalogList, error) {
	client, err := NewClientWithAuth(auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	org, err := client.GetOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", auth.Organization, err)
	}

	catalogs, err := org.QueryCatalogList()
	if err != nil {
		return nil, fmt.Errorf("failed to get list catalog for organization %s: %w", auth.Organization, err)
	}

	var catlogArr apiv1.VMwareCloudDirectorCatalogList
	for _, catalog := range catalogs {
		catlogArr = append(catlogArr, apiv1.VMwareCloudDirectorCatalog{
			Name: catalog.Name,
		})
	}
	return catlogArr, nil
}

func ListTemplates(ctx context.Context, auth Auth, catalogName string) (apiv1.VMwareCloudDirectorTemplateList, error) {
	client, err := NewClientWithAuth(auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	org, err := client.GetOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", auth.Organization, err)
	}

	catalog, err := org.GetCatalogByNameOrId(catalogName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog '%s': %w", catalogName, err)
	}

	templates, err := catalog.QueryVappTemplateList()
	if err != nil {
		return nil, fmt.Errorf("failed to list templates for catalog '%s': %w", catalogName, err)
	}

	var templateArr apiv1.VMwareCloudDirectorTemplateList
	for _, template := range templates {
		templateArr = append(templateArr, apiv1.VMwareCloudDirectorTemplate{
			Name: template.Name,
		})
	}
	return templateArr, nil
}

func ListOVDCNetworks(ctx context.Context, auth Auth) (apiv1.VMwareCloudDirectorNetworkList, error) {
	client, err := NewClientWithAuth(auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	org, err := client.GetOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", auth.Organization, err)
	}

	orgVDC, err := client.GetVDCForOrg(*org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization VDC '%s': %w", auth.VDC, err)
	}

	var orgVDCNetworks apiv1.VMwareCloudDirectorNetworkList
	for _, an := range orgVDC.Vdc.AvailableNetworks {
		for _, reference := range an.Network {
			if reference.HREF != "" {
				orgVDCNetworks = append(orgVDCNetworks, apiv1.VMwareCloudDirectorNetwork{
					Name: reference.Name,
				})
			}
		}
	}

	return orgVDCNetworks, nil
}

func ListStorageProfiles(ctx context.Context, auth Auth) (apiv1.VMwareCloudDirectorStorageProfileList, error) {
	client, err := NewClientWithAuth(auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}

	org, err := client.GetOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", auth.Organization, err)
	}

	orgVDC, err := client.GetVDCForOrg(*org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization VDC %q: %w", auth.VDC, err)
	}

	var storageProfiles apiv1.VMwareCloudDirectorStorageProfileList
	if orgVDC.Vdc.VdcStorageProfiles == nil {
		return storageProfiles, nil
	}

	for _, reference := range orgVDC.Vdc.VdcStorageProfiles.VdcStorageProfile {
		if reference.HREF != "" {
			storageProfiles = append(storageProfiles, apiv1.VMwareCloudDirectorStorageProfile{
				Name: reference.Name,
			})
		}
	}

	return storageProfiles, nil
}
