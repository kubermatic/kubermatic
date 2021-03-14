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

package provider

import (
	"context"
	"fmt"
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func OpenstackSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {
	cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("error getting dc: %v", err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	settings, err := settingsProvider.GetGlobalSettings()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return GetOpenstackSizes(creds, datacenterName, datacenter, settings.Spec.MachineDeploymentVMResourceQuota)
}

func OpenstackTenantWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	return GetOpenstackTenants(userInfo, seedsGetter, creds, datacenterName)
}

func OpenstackNetworkWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	return GetOpenstackNetworks(userInfo, seedsGetter, creds, datacenterName)
}

func OpenstackSecurityGroupWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	return GetOpenstackSecurityGroups(userInfo, seedsGetter, creds, datacenterName)
}

func OpenstackSubnetsWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID, networkID string) (interface{}, error) {
	cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	return GetOpenstackSubnets(userInfo, seedsGetter, creds, networkID, datacenterName)
}

func OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("error getting dc: %v", err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	return GetOpenstackAvailabilityZones(creds, datacenterName, datacenter)
}

func GetOpenstackAvailabilityZones(creds *resources.OpenstackCredentials, datacenterName string, datacenter *kubermaticv1.Datacenter) ([]apiv1.OpenstackAvailabilityZone, error) {
	availabilityZones, err := openstack.GetAvailabilityZones(creds, datacenter.Spec.Openstack.AuthURL, datacenter.Spec.Openstack.Region)
	if err != nil {
		return nil, err
	}

	apiAvailabilityZones := []apiv1.OpenstackAvailabilityZone{}
	for _, availabilityZone := range availabilityZones {
		apiAvailabilityZones = append(apiAvailabilityZones, apiv1.OpenstackAvailabilityZone{
			Name: availabilityZone.ZoneName,
		})
	}

	return apiAvailabilityZones, nil
}

func GetOpenstackSubnets(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, creds *resources.OpenstackCredentials, networkID, datacenterName string) ([]apiv1.OpenstackSubnet, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	subnets, err := openstack.GetSubnets(creds, networkID, authURL, region)
	if err != nil {
		return nil, err
	}

	apiSubnetIDs := []apiv1.OpenstackSubnet{}
	for _, subnet := range subnets {
		apiSubnetIDs = append(apiSubnetIDs, apiv1.OpenstackSubnet{
			ID:   subnet.ID,
			Name: subnet.Name,
		})
	}

	return apiSubnetIDs, nil
}

func GetOpenstackNetworks(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, creds *resources.OpenstackCredentials, datacenterName string) ([]apiv1.OpenstackNetwork, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	networks, err := openstack.GetNetworks(creds, authURL, region)
	if err != nil {
		return nil, err
	}

	apiNetworks := []apiv1.OpenstackNetwork{}
	for _, network := range networks {
		apiNetwork := apiv1.OpenstackNetwork{
			Name:     network.Name,
			ID:       network.ID,
			External: network.External,
		}

		apiNetworks = append(apiNetworks, apiNetwork)
	}

	return apiNetworks, nil
}

func GetOpenstackTenants(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, creds *resources.OpenstackCredentials, datacenterName string) ([]apiv1.OpenstackTenant, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	tenants, err := openstack.GetTenants(creds, authURL, region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants: %v", err)
	}

	apiTenants := []apiv1.OpenstackTenant{}
	for _, tenant := range tenants {
		apiTenant := apiv1.OpenstackTenant{
			Name: tenant.Name,
			ID:   tenant.ID,
		}

		apiTenants = append(apiTenants, apiTenant)
	}

	return apiTenants, nil
}

func GetOpenstackSizes(creds *resources.OpenstackCredentials, datacenterName string, datacenter *kubermaticv1.Datacenter, quota kubermaticv1.MachineDeploymentVMResourceQuota) ([]apiv1.OpenstackSize, error) {
	flavors, err := openstack.GetFlavors(creds, datacenter.Spec.Openstack.AuthURL, datacenter.Spec.Openstack.Region)
	if err != nil {
		return nil, err
	}

	apiSizes := []apiv1.OpenstackSize{}
	for _, flavor := range flavors {
		apiSize := apiv1.OpenstackSize{
			Slug:     flavor.Name,
			Memory:   flavor.RAM,
			VCPUs:    flavor.VCPUs,
			Disk:     flavor.Disk,
			Swap:     flavor.Swap,
			Region:   datacenter.Spec.Openstack.Region,
			IsPublic: flavor.IsPublic,
		}
		if MeetsOpenstackNodeSizeRequirement(apiSize, datacenter.Spec.Openstack.NodeSizeRequirements) {
			if IsFlavorEnabled(apiSize, datacenter.Spec.Openstack.EnabledFlavors) {
				apiSizes = append(apiSizes, apiSize)
			}
		}
	}

	return filterOpenStackByQuota(apiSizes, quota), nil
}

func filterOpenStackByQuota(instances []apiv1.OpenstackSize, quota kubermaticv1.MachineDeploymentVMResourceQuota) []apiv1.OpenstackSize {
	var filteredRecords []apiv1.OpenstackSize

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range instances {
		keep := true

		if !handlercommon.FilterCPU(r.VCPUs, quota.MinCPU, quota.MaxCPU) {
			keep = false
		}
		if !handlercommon.FilterMemory(r.Memory/1024, quota.MinRAM, quota.MaxRAM) {
			keep = false
		}

		if keep {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

func MeetsOpenstackNodeSizeRequirement(apiSize apiv1.OpenstackSize, requirements kubermaticv1.OpenstackNodeSizeRequirements) bool {
	if apiSize.VCPUs < requirements.MinimumVCPUs {
		return false
	}
	if apiSize.Memory < requirements.MinimumMemory {
		return false
	}
	return true
}

func IsFlavorEnabled(apiSize apiv1.OpenstackSize, enabledFlavors []string) bool {
	if len(enabledFlavors) == 0 {
		// Flavors are enabled if no restrictions were made.
		return true
	}
	for _, flavor := range enabledFlavors {
		if flavor == apiSize.Slug {
			return true
		}
	}
	return false
}

func GetOpenstackSecurityGroups(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, creds *resources.OpenstackCredentials, datacenterName string) ([]apiv1.OpenstackSecurityGroup, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	securityGroups, err := openstack.GetSecurityGroups(creds, authURL, region)
	if err != nil {
		return nil, err
	}

	apiSecurityGroups := []apiv1.OpenstackSecurityGroup{}
	for _, securityGroup := range securityGroups {
		apiSecurityGroup := apiv1.OpenstackSecurityGroup{
			Name: securityGroup.Name,
			ID:   securityGroup.ID,
		}

		apiSecurityGroups = append(apiSecurityGroups, apiSecurityGroup)
	}

	return apiSecurityGroups, nil
}

func getClusterForOpenstack(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, projectID string, clusterID string) (*kubermaticv1.Cluster, error) {
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Openstack == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}
	return cluster, nil
}

func getOpenstackAuthURLAndRegion(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, datacenterName string) (string, string, error) {
	_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return "", "", fmt.Errorf("failed to find datacenter %q: %v", datacenterName, err)
	}
	return dc.Spec.Openstack.AuthURL, dc.Spec.Openstack.Region, nil
}

func getCredentials(ctx context.Context, cloudSpec kubermaticv1.CloudSpec) (*resources.OpenstackCredentials, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	credentials, err := openstack.GetCredentialsForCluster(cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}
