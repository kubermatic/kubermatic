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

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// EKSTypesReq represent a request for EKS types.
// swagger:parameters validateEKSCredentials
type EKSTypesReq struct {
	EKSCommonReq
	// in: header
	// name: Region
	Region string
}

// EKSCommonReq represent a request with common parameters for EKS.
type EKSCommonReq struct {
	// in: header
	// name: AccessKeyID
	AccessKeyID string
	// in: header
	// name: SecretAccessKey
	SecretAccessKey string
	// in: header
	// name: Credential
	Credential string
}

func DecodeEKSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeEKSTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSTypesReq

	commonReq, err := DecodeEKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Region = r.Header.Get("Region")
	req.EKSCommonReq = commonReq.(EKSCommonReq)

	return req, nil
}

// EKSClusterListReq represent a request for EKS cluster list.
// swagger:parameters listEKSClusters
type EKSClusterListReq struct {
	common.ProjectReq
	EKSTypesReq
}

func (req EKSTypesReq) Validate() error {
	if len(req.Credential) != 0 {
		return nil
	}
	if len(req.AccessKeyID) == 0 || len(req.SecretAccessKey) == 0 || len(req.Region) == 0 {
		return fmt.Errorf("EKS Credentials or Region cannot be empty")
	}
	return nil
}

func (req EKSSubnetIDsReq) Validate() error {
	if len(req.VpcId) == 0 {
		return fmt.Errorf("EKS VPC ID cannot be empty")
	}
	if len(req.Credential) != 0 {
		return nil
	}
	if len(req.AccessKeyID) == 0 || len(req.SecretAccessKey) == 0 {
		return fmt.Errorf("EKS Credentials cannot be empty")
	}
	if len(req.Region) == 0 {
		return fmt.Errorf("Region cannot be empty")
	}
	return nil
}

func DecodeEKSClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSClusterListReq

	typesReq, err := DecodeEKSTypesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.EKSTypesReq = typesReq.(EKSTypesReq)
	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

func ListEKSClustersEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSClusterListReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}
		credential, err := getEKSCredentialsFromReq(ctx, req.EKSTypesReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, *credential, req.ProjectID)
	}
}

func ListEKSVpcIdsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSVpcIds(ctx, *credential)
	}
}

func ListEKSSubnetIDsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSSubnetIDsReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req.EKSTypesReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSSubnetIDs(ctx, *credential, req.VpcId)
	}
}

func ListEKSSecurityGroupIDsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSSubnetIDsReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req.EKSTypesReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSSecurityGroupIDs(ctx, *credential, req.VpcId)
	}
}

func ListEKSRegionsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)

		credential, err := getEKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSRegions(ctx, *credential)
	}
}

func EKSValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return nil, providercommon.ValidateEKSCredentials(ctx, *credential)
	}
}

func getEKSCredentialsFromReq(ctx context.Context, req EKSTypesReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) (*providercommon.EKSCredential, error) {
	accessKeyID := req.AccessKeyID
	secretAccessKey := req.SecretAccessKey
	region := req.Region

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	presetName := req.Credential
	if len(presetName) > 0 {
		preset, err := presetProvider.GetPreset(userInfo, presetName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", presetName, userInfo.Email))
		}
		if credentials := preset.Spec.EKS; credentials != nil {
			accessKeyID = credentials.AccessKeyID
			secretAccessKey = credentials.SecretAccessKey
			region = credentials.Region
		}
	}

	return &providercommon.EKSCredential{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}, nil
}

// EKSSubnetsReq represent a request for EKS subnets.
// swagger:parameters listEKSSubnetIDs
type EKSSubnetIDsReq struct {
	EKSTypesReq
	// in: header
	// name: VpcId
	VpcId string
}

func DecodeEKSSubnetIDsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSSubnetIDsReq

	typesReq, err := DecodeEKSTypesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.EKSTypesReq = typesReq.(EKSTypesReq)
	req.VpcId = r.Header.Get("VpcId")

	return req, nil
}
