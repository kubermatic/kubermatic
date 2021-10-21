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

// GetSeedCluster returns the SeedCluster object
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

// AWSSubnetNoCredentialsEndpoint handles the request to list AWS availability subnets in a given vpc, using credentials
func AWSSubnetNoCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(cluster.GetClusterReq)
		return providercommon.AWSSubnetNoCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
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

// EKSTypesReq represent a request for EKS types.
// swagger:parameters listEKSClusters
type EKSTypesReq struct {
	EKSCommonReq
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

type credentials struct {
	accessKeyID     string
	secretAccessKey string
}

func getPresetCredentials(userInfo *provider.UserInfo, presetName string, presetProvider provider.PresetProvider) (*credentials, error) {

	preset, err := presetProvider.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, fmt.Errorf("can not get preset %s for the user %s", presetName, userInfo.Email)
	}

	aws := preset.Spec.AWS
	if aws == nil {
		return nil, fmt.Errorf("credentials for AWS not present in preset %s for the user %s", presetName, userInfo.Email)
	}
	return &credentials{
		accessKeyID:     aws.AccessKeyID,
		secretAccessKey: aws.SecretAccessKey,
	}, nil
}

// Validate validates EKSCommonReq request
func (req EKSCommonReq) Validate() error {
	if len(req.Credential) == 0 && len(req.AccessKeyID) == 0 && len(req.SecretAccessKey) == 0 {
		return fmt.Errorf("AWS credentials cannot be empty")
	}
	if len(req.Region) == 0 {
		return fmt.Errorf("AWS region cannot be empty")
	}
	return nil
}

func ListEKSClustersEndpoint(userInfoGetter provider.UserInfoGetter, presetsProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(EKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		accessKeyID := req.AccessKeyID
		secretAccessKey := req.SecretAccessKey
		presetName := req.Credential
		region := req.Region

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Preset is used
		if len(presetName) > 0 {
			credentials, err := getPresetCredentials(userInfo, presetName, presetsProvider)
			if err != nil {
				return nil, fmt.Errorf("error getting preset credentials for AWS: %v", err)
			}
			accessKeyID = credentials.accessKeyID
			secretAccessKey = credentials.secretAccessKey
		}
		return providercommon.ListEKSClusters(ctx, accessKeyID, secretAccessKey, region)
	}
}

// EC2CommonReq represent a request with common parameters for .
type EC2CommonReq struct {
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

func DecodeEC2CommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EC2CommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// Validate validates EC2RegionReq request
func (req EC2CommonReq) Validate() error {
	if len(req.Credential) == 0 && len(req.AccessKeyID) == 0 && len(req.SecretAccessKey) == 0 {
		return fmt.Errorf("AWS credentials cannot be empty")
	}
	return nil
}

func ListEC2RegionsEndpoint(userInfoGetter provider.UserInfoGetter, presetsProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(EC2CommonReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		accessKeyID := req.AccessKeyID
		secretAccessKey := req.SecretAccessKey
		presetName := req.Credential

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Preset is used
		if len(presetName) > 0 {
			credentials, err := getPresetCredentials(userInfo, presetName, presetsProvider)
			if err != nil {
				return nil, fmt.Errorf("error getting preset credentials for AWS: %v", err)
			}
			accessKeyID = credentials.accessKeyID
			secretAccessKey = credentials.secretAccessKey
		}
		return providercommon.ListEC2Regions(ctx, accessKeyID, secretAccessKey)
	}
}
