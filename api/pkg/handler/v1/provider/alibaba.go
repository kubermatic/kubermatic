package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/alibaba"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
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

// AlibabaInstanceTypeReq represent a request for Alibaba instance types.
// swagger:parameters listAlibabaInstanceTypes
type AlibabaInstanceTypeReq struct {
	AlibabaCommonReq
	// in: header
	// name: Region
	Region string
}

// AlibabaInstanceTypesNoCredentialReq represent a request for Alibaba instance types.
// swagger:parameters listAlibabaInstanceTypesNoCredentials
type AlibabaInstanceTypesNoCredentialReq struct {
	common.GetClusterReq
	// in: header
	// name: Region
	Region string
}

func DecodeAlibabaInstanceTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AlibabaInstanceTypeReq

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

func DecodeAlibabaInstanceTypesNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AlibabaInstanceTypesNoCredentialReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)
	req.Region = r.Header.Get("Region")

	return req, nil
}

func AlibabaInstanceTypesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AlibabaInstanceTypeReq)

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

func AlibabaInstanceTypesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AlibabaInstanceTypesNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.Alibaba == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		accessKeyID, accessKeySecret, err := alibaba.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}

		return listAlibabaInstanceTypes(ctx, accessKeyID, accessKeySecret, req.Region)
	}
}

func listAlibabaInstanceTypes(ctx context.Context, accessKeyID string, accessKeySecret string, region string) (apiv1.AlibabaInstanceTypeList, error) {
	instanceTypes := apiv1.AlibabaInstanceTypeList{}

	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to create client: %v", err))
	}

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.Scheme = "https"

	instTypes, err := client.DescribeInstanceTypes(request)
	if err != nil {
		return apiv1.AlibabaInstanceTypeList{}, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list instance types: %v", err))
	}

	for _, instType := range instTypes.InstanceTypes.InstanceType {
		// Pods with memory under 2 GB will be full with required pods like kube-proxy, CNI etc.
		if instType.MemorySize >= 2 {
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
