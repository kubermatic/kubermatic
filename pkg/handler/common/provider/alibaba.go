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

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/alibaba"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/sets"
)

const requestScheme = "https"

func AlibabaInstanceTypesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID, region string) (interface{}, error) {

	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Alibaba == nil {
		return nil, errors.NewNotFound("cloud spec for %s", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	accessKeyID, accessKeySecret, err := alibaba.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.Alibaba)
	if err != nil {
		return nil, err
	}

	settings, err := settingsProvider.GetGlobalSettings()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return ListAlibabaInstanceTypes(accessKeyID, accessKeySecret, region, settings.Spec.MachineDeploymentVMResourceQuota)

}

func ListAlibabaInstanceTypes(accessKeyID string, accessKeySecret string, region string, quota kubermaticv1.MachineDeploymentVMResourceQuota) (apiv1.AlibabaInstanceTypeList, error) {
	// Alibaba has way too many instance types that are not all available in each region
	// recommendedInstanceFamilies are those families that are recommended in this document:
	// https://www.alibabacloud.com/help/doc-detail/25378.htm?spm=a2c63.p38356.b99.47.7acf342enhNVmo
	recommendedInstanceFamilies := sets.NewString("ecs.g6", "ecs.g5", "ecs.g5se", "ecs.g5ne", "ecs.ic5", "ecs.c6", "ecs.c5", "ecs.r6", "ecs.r5", "ecs.d1ne", "ecs.i2", "ecs.i2g", "ecs.hfc6", "ecs.hfg6", "ecs.hfr6")
	gpuInstanceFamilies := sets.NewString("ecs.gn3", "ecs.ga1", "ecs.gn4", "ecs.gn6i", "ecs.vgn6i", "ecs.gn6e", "ecs.gn6v", "ecs.vgn5i", "ecs.gn5", "ecs.gn5i")
	availableInstanceFamilies := sets.String{}
	instanceTypes := apiv1.AlibabaInstanceTypeList{}

	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to create client: %v", err))
	}

	// get all families that are available for the Region
	requestFamilies := ecs.CreateDescribeInstanceTypeFamiliesRequest()
	requestFamilies.Scheme = requestScheme
	requestFamilies.RegionId = region

	instTypeFamilies, err := client.DescribeInstanceTypeFamilies(requestFamilies)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list instance type families: %v", err))
	}

	if quota.EnableGPU {
		recommendedInstanceFamilies.Insert(gpuInstanceFamilies.List()...)
	}

	for _, instanceFamily := range instTypeFamilies.InstanceTypeFamilies.InstanceTypeFamily {
		if recommendedInstanceFamilies.Has(instanceFamily.InstanceTypeFamilyId) {
			availableInstanceFamilies.Insert(instanceFamily.InstanceTypeFamilyId)
		}
	}

	// get all instance types and filter afterwards, to reduce calls
	requestInstanceTypes := ecs.CreateDescribeInstanceTypesRequest()
	requestInstanceTypes.Scheme = requestScheme

	instTypes, err := client.DescribeInstanceTypes(requestInstanceTypes)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list instance types: %v", err))
	}

	for _, instType := range instTypes.InstanceTypes.InstanceType {
		if availableInstanceFamilies.Has(instType.InstanceTypeFamily) {
			it := apiv1.AlibabaInstanceType{
				ID:           instType.InstanceTypeId,
				CPUCoreCount: instType.CpuCoreCount,
				GPUCoreCount: instType.GPUAmount,
				MemorySize:   instType.MemorySize,
			}
			instanceTypes = append(instanceTypes, it)
		}
	}

	return filterByQuota(instanceTypes, quota), nil
}

func filterByQuota(instances apiv1.AlibabaInstanceTypeList, quota kubermaticv1.MachineDeploymentVMResourceQuota) apiv1.AlibabaInstanceTypeList {
	filteredRecords := apiv1.AlibabaInstanceTypeList{}

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range instances {
		keep := true

		if !handlercommon.FilterCPU(r.CPUCoreCount, quota.MinCPU, quota.MaxCPU) {
			keep = false
		}
		if !handlercommon.FilterMemory(int(r.MemorySize), quota.MinRAM, quota.MaxRAM) {
			keep = false
		}

		if keep {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

func AlibabaZonesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID, region string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Alibaba == nil {
		return nil, errors.NewNotFound("cloud spec for %s", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	accessKeyID, accessKeySecret, err := alibaba.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.Alibaba)
	if err != nil {
		return nil, err
	}

	return ListAlibabaZones(ctx, accessKeyID, accessKeySecret, region)
}

func ListAlibabaZones(ctx context.Context, accessKeyID string, accessKeySecret string, region string) (apiv1.AlibabaZoneList, error) {
	zones := apiv1.AlibabaZoneList{}

	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to create client: %v", err))
	}

	requestZones := ecs.CreateDescribeZonesRequest()
	requestZones.Scheme = requestScheme

	responseZones, err := client.DescribeZones(requestZones)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list zones: %v", err))
	}

	for _, zone := range responseZones.Zones.Zone {
		z := apiv1.AlibabaZone{
			ID: zone.ZoneId,
		}
		zones = append(zones, z)
	}

	return zones, nil
}
