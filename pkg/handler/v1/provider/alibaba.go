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
	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/alibaba"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/sets"
)

// AlibabaCommonReq represent a request with common parameters for Alibaba.
type AlibabaCommonReq struct {
	// in: header
	// name: AccessKeyID
	AccessKeyID string
	// in: header
	// name: AccessKeySecret
	AccessKeySecret string
	// in: header
	// name: Credential
	Credential string
}

// AlibabaReq represent a request for Alibaba instance types.
// swagger:parameters listAlibabaInstanceTypes listAlibabaZones
type AlibabaReq struct {
	AlibabaCommonReq
	// in: header
	// name: Region
	Region string
}

// AlibabaNoCredentialReq represent a request for Alibaba instance types.
// swagger:parameters listAlibabaInstanceTypesNoCredentials listAlibabaZonesNoCredentials
type AlibabaNoCredentialReq struct {
	common.GetClusterReq
	// in: header
	// name: Region
	Region string
}

func DecodeAlibabaReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AlibabaReq

	commonReq, err := DecodeAlibabaCommonReq(c, r)
	if err != nil {
		return nil, err
	}

	req.AlibabaCommonReq = commonReq.(AlibabaCommonReq)

	req.Region = r.Header.Get("Region")
	return req, nil
}

func DecodeAlibabaCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AlibabaCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.AccessKeySecret = r.Header.Get("AccessKeySecret")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeAlibabaNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AlibabaNoCredentialReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)
	req.Region = r.Header.Get("Region")

	return req, nil
}

const requestScheme = "https"

func AlibabaInstanceTypesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AlibabaReq)

		accessKeyID := req.AccessKeyID
		accessKeySecret := req.AccessKeySecret

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Alibaba; credentials != nil {
				accessKeyID = credentials.AccessKeyID
				accessKeySecret = credentials.AccessKeySecret
			}
		}
		return listAlibabaInstanceTypes(ctx, accessKeyID, accessKeySecret, req.Region)
	}
}

func AlibabaInstanceTypesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AlibabaNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.Alibaba == nil {
			return nil, errors.NewNotFound("cloud spec for %s", req.ClusterID)
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

		return listAlibabaInstanceTypes(ctx, accessKeyID, accessKeySecret, req.Region)
	}
}

func listAlibabaInstanceTypes(ctx context.Context, accessKeyID string, accessKeySecret string, region string) (apiv1.AlibabaInstanceTypeList, error) {
	// Alibaba has way too many instance types that are not all available in each region
	// recommendedInstanceFamilies are those families that are recommended in this document:
	// https://www.alibabacloud.com/help/doc-detail/25378.htm?spm=a2c63.p38356.b99.47.7acf342enhNVmo
	recommendedInstanceFamilies := sets.NewString("ecs.g6", "ecs.g5", "ecs.g5se", "ecs.g5ne", "ecs.ic5", "ecs.c6", "ecs.c5", "ecs.r6", "ecs.r5", "ecs.d1ne", "ecs.i2", "ecs.i2g", "ecs.hfc6", "ecs.hfg6", "ecs.hfr6")
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
				MemorySize:   instType.MemorySize,
			}
			instanceTypes = append(instanceTypes, it)
		}
	}

	return instanceTypes, nil
}

func AlibabaZonesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AlibabaReq)

		accessKeyID := req.AccessKeyID
		accessKeySecret := req.AccessKeySecret

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Alibaba; credentials != nil {
				accessKeyID = credentials.AccessKeyID
				accessKeySecret = credentials.AccessKeySecret
			}
		}
		return listAlibabaZones(ctx, accessKeyID, accessKeySecret, req.Region)
	}
}

func AlibabaZonesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AlibabaNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.Alibaba == nil {
			return nil, errors.NewNotFound("cloud spec for %s", req.ClusterID)
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

		return listAlibabaZones(ctx, accessKeyID, accessKeySecret, req.Region)
	}
}

func listAlibabaZones(ctx context.Context, accessKeyID string, accessKeySecret string, region string) (apiv1.AlibabaZoneList, error) {
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
