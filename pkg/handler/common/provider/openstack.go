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
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func OpenstackSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID string, caBundle *x509.CertPool) (interface{}, error) {
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
		return nil, fmt.Errorf("error getting dc: %w", err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	settings, err := settingsProvider.GetGlobalSettings(ctx)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return GetOpenstackSizes(creds, datacenter, settings.Spec.MachineDeploymentVMResourceQuota, caBundle)
}

func OpenstackTenantWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, projectID, clusterID string, caBundle *x509.CertPool) (interface{}, error) {
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
	return GetOpenstackProjects(userInfo, seedsGetter, creds, datacenterName, caBundle)
}

func OpenstackNetworkWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, projectID, clusterID string, caBundle *x509.CertPool) (interface{}, error) {
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
	return GetOpenstackNetworks(ctx, userInfo, seedsGetter, creds, datacenterName, caBundle)
}

func OpenstackSecurityGroupWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, projectID, clusterID string, caBundle *x509.CertPool) (interface{}, error) {
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

	return GetOpenstackSecurityGroups(ctx, userInfo, seedsGetter, creds, datacenterName, caBundle)
}

func OpenstackSubnetsWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, projectID, clusterID, networkID string, caBundle *x509.CertPool) (interface{}, error) {
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

	return GetOpenstackSubnets(ctx, userInfo, seedsGetter, creds, networkID, datacenterName, caBundle)
}

func OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, projectID, clusterID string, caBundle *x509.CertPool) (interface{}, error) {
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
		return nil, fmt.Errorf("error getting dc: %w", err)
	}

	creds, err := getCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	return GetOpenstackAvailabilityZones(datacenter, creds, caBundle)
}

func GetOpenstackAvailabilityZones(datacenter *kubermaticv1.Datacenter, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]apiv1.OpenstackAvailabilityZone, error) {
	availabilityZones, err := openstack.GetAvailabilityZones(datacenter.Spec.Openstack.AuthURL, datacenter.Spec.Openstack.Region, credentials, caBundle)
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

func GetOpenstackSubnets(ctx context.Context, userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, credentials *resources.OpenstackCredentials, networkID, datacenterName string, caBundle *x509.CertPool) ([]apiv1.OpenstackSubnet, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	subnets, err := openstack.GetSubnets(ctx, authURL, region, networkID, credentials, caBundle)
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

func GetOpenstackNetworks(ctx context.Context, userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, credentials *resources.OpenstackCredentials, datacenterName string, caBundle *x509.CertPool) ([]apiv1.OpenstackNetwork, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	networks, err := openstack.GetNetworks(ctx, authURL, region, credentials, caBundle)
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

func GetOpenstackProjects(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, credentials *resources.OpenstackCredentials, datacenterName string, caBundle *x509.CertPool) ([]apiv1.OpenstackTenant, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	projects, err := openstack.GetTenants(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get projects: %w", err)
	}

	apiProjects := []apiv1.OpenstackTenant{}
	for _, project := range projects {
		apiProjects = append(apiProjects, apiv1.OpenstackTenant{
			Name: project.Name,
			ID:   project.ID,
		})
	}

	return apiProjects, nil
}

func GetOpenstackSizes(credentials *resources.OpenstackCredentials, datacenter *kubermaticv1.Datacenter,
	quota kubermaticv1.MachineDeploymentVMResourceQuota, caBundle *x509.CertPool) ([]apiv1.OpenstackSize, error) {
	flavors, err := openstack.GetFlavors(datacenter.Spec.Openstack.AuthURL,
		datacenter.Spec.Openstack.Region, credentials, caBundle)
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

func GetOpenStackFlavorSize(credentials *resources.OpenstackCredentials, authURL, region string,
	caBundle *x509.CertPool, flavorName string) (*apiv1.OpenstackSize, error) {
	flavors, err := openstack.GetFlavors(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, err
	}

	for _, flavor := range flavors {
		if strings.EqualFold(flavor.Name, flavorName) {
			return &apiv1.OpenstackSize{
				Slug:     flavor.Name,
				Memory:   flavor.RAM,
				VCPUs:    flavor.VCPUs,
				Disk:     flavor.Disk,
				Swap:     flavor.Swap,
				Region:   region,
				IsPublic: flavor.IsPublic,
			}, nil
		}
	}

	return nil, fmt.Errorf("cannot find openstack flavor %q size", flavorName)
}

func filterOpenStackByQuota(instances []apiv1.OpenstackSize, quota kubermaticv1.MachineDeploymentVMResourceQuota) []apiv1.OpenstackSize {
	var filteredRecords []apiv1.OpenstackSize

	filteredRecords = make([]apiv1.OpenstackSize, 0)
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

func GetOpenstackSecurityGroups(ctx context.Context, userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, credentials *resources.OpenstackCredentials, datacenterName string, caBundle *x509.CertPool) ([]apiv1.OpenstackSecurityGroup, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	securityGroups, err := openstack.GetSecurityGroups(ctx, authURL, region, credentials, caBundle)
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
		return nil, utilerrors.NewNotFound("cloud spec for ", clusterID)
	}
	return cluster, nil
}

func getOpenstackAuthURLAndRegion(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, datacenterName string) (string, string, error) {
	_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return "", "", fmt.Errorf("failed to find datacenter %q: %w", datacenterName, err)
	}

	if len(dc.Spec.Openstack.AuthURL) == 0 {
		return "", "", fmt.Errorf("empty authURL in datacenter %q", datacenterName)
	}

	if len(dc.Spec.Openstack.Region) == 0 {
		return "", "", fmt.Errorf("empty region in datacenter %q", datacenterName)
	}

	return dc.Spec.Openstack.AuthURL, dc.Spec.Openstack.Region, nil
}

func getCredentials(ctx context.Context, cloudSpec kubermaticv1.CloudSpec) (*resources.OpenstackCredentials, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	credentials, err := openstack.GetCredentialsForCluster(cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return credentials, nil
}
