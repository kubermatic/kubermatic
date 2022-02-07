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

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// AWSSizeNoCredentialsEndpoint handles the request to list available AWS sizes.
func AWSSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(awsSizeNoCredentialsReq)
		return providercommon.AWSSizeNoCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID, req.Architecture)
	}
}

// awsSizeNoCredentialsReq represent a request for AWS machine types resources
// swagger:parameters listAWSSizesNoCredentialsV2
type awsSizeNoCredentialsReq struct {
	cluster.GetClusterReq
	// architecture query parameter. Supports: arm64 and x64 types.
	// in: query
	Architecture string `json:"architecture,omitempty"`
}

// GetSeedCluster returns the SeedCluster object.
func (req awsSizeNoCredentialsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeAWSSizeNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req awsSizeNoCredentialsReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	req.Architecture = r.URL.Query().Get("architecture")
	if len(req.Architecture) > 0 {
		if req.Architecture == handlercommon.ARM64Architecture || req.Architecture == handlercommon.X64Architecture {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported architecture: %s", req.Architecture)
	}

	return req, nil
}

// AWSSubnetNoCredentialsEndpoint handles the request to list AWS availability subnets in a given vpc, using credentials.
func AWSSubnetNoCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(cluster.GetClusterReq)
		return providercommon.AWSSubnetNoCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

// EKSTypesReq represent a request for EKS types.
// swagger:parameters validateEKSCredentials
type EKSTypesReq struct {
	EKSCommonReq
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
	// in: header
	// name: Region
	Region string
}

func DecodeEKSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")
	req.Region = r.Header.Get("Region")

	return req, nil
}

func DecodeEKSTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSTypesReq

	commonReq, err := DecodeEKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.EKSCommonReq = commonReq.(EKSCommonReq)

	return req, nil
}

// EKSClusterListReq represent a request for EKS cluster list.
// swagger:parameters listEKSClusters
type EKSClusterListReq struct {
	common.ProjectReq
	EKSCommonReq
}

func (req EKSCommonReq) Validate() error {
	if len(req.Credential) != 0 {
		return nil
	}
	if len(req.AccessKeyID) == 0 || len(req.SecretAccessKey) == 0 || len(req.Region) == 0 {
		return fmt.Errorf("EKS Credentials or Region cannot be empty")
	}
	return nil
}

func DecodeEKSClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSClusterListReq

	commonReq, err := DecodeEKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.EKSCommonReq = commonReq.(EKSCommonReq)
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
		credential, err := getEKSCredentialsFromReq(ctx, req.EKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, *credential, req.ProjectID)
	}
}

// AWSCommonReq represent a request with common parameters for .
type AWSCommonReq struct {
	// in: header
	// name: AccessKeyID
	AccessKeyID string
	// in: header
	// name: SecretAccessKey
	SecretAccessKey string
	// in: header
	// name: Credential
	Credential string
	// in: header
	// name: AssumeRoleARN
	AssumeRoleARN string
	// in: header
	// name: AssumeRoleExternalID
	AssumeRoleExternalID string
}

func DecodeAWSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")
	req.AssumeRoleARN = r.Header.Get("AssumeRoleARN")
	req.AssumeRoleExternalID = r.Header.Get("AssumeRoleExternalID")

	return req, nil
}

// Validate validates AWSCommonReq request.
func (req AWSCommonReq) Validate() error {
	if len(req.Credential) == 0 && len(req.AccessKeyID) == 0 && len(req.SecretAccessKey) == 0 {
		return fmt.Errorf("AWS credentials cannot be empty")
	}
	return nil
}

func EKSValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req.EKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return nil, providercommon.ValidateEKSCredentials(ctx, *credential)
	}
}

func getEKSCredentialsFromReq(ctx context.Context, req EKSCommonReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) (*providercommon.EKSCredential, error) {
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
