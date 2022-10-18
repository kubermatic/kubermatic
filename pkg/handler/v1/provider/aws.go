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
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/utils/pointer"
)

// AWSCommonReq represent a request with common parameters for AWS.
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
	// name: ExternalID
	AssumeRoleExternalID string
	// in: header
	// name: VPC
	VPC string
}

// AWSSubnetReq represent a request for AWS subnets.
// swagger:parameters listAWSSubnets
type AWSSubnetReq struct {
	AWSCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// AWSVPCReq represent a request for AWS vpc's.
// swagger:parameters listAWSVPCS
type AWSVPCReq struct {
	AWSCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// AWSSecurityGroupsReq represent a request for AWS Security Group IDs.
// swagger:parameters listAWSSecurityGroups
type AWSSecurityGroupsReq struct {
	AWSCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// AWSSizeReq represent a request for AWS VM sizes.
// swagger:parameters listAWSSizes
type AWSSizeReq struct {
	// in: header
	// name: Region
	Region string

	// architecture query parameter. Supports: arm64 and x64 types.
	// in: query
	Architecture string `json:"architecture,omitempty"`
}

// DecodeAWSSizesReq decodes the base type for a AWS special endpoint request.
func DecodeAWSSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSSizeReq
	req.Region = r.Header.Get("Region")

	req.Architecture = r.URL.Query().Get("architecture")
	if len(req.Architecture) > 0 {
		if req.Architecture == handlercommon.ARM64Architecture || req.Architecture == handlercommon.X64Architecture {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported architecture: %s", req.Architecture)
	}

	return req, nil
}

// DecodeAWSCommonReq decodes the base type for a AWS special endpoint request.
func DecodeAWSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.AssumeRoleARN = r.Header.Get("AssumeRoleARN")
	req.AssumeRoleExternalID = r.Header.Get("AssumeRoleExternalID")
	req.Credential = r.Header.Get("Credential")
	req.VPC = r.Header.Get("VPC")

	return req, nil
}

// DecodeAWSSubnetReq decodes a request for a list of AWS subnets.
func DecodeAWSSubnetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSSubnetReq

	commonReq, err := DecodeAWSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AWSCommonReq = commonReq.(AWSCommonReq)

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

// DecodeAWSVPCReq decodes a request for a list of AWS vpc's.
func DecodeAWSVPCReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSVPCReq

	commonReq, err := DecodeAWSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AWSCommonReq = commonReq.(AWSCommonReq)

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

// DecodeAWSSecurityGroupsReq decodes a request for a list of AWS Security Groups.
func DecodeAWSSecurityGroupsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSSecurityGroupsReq

	commonReq, err := DecodeAWSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AWSCommonReq = commonReq.(AWSCommonReq)

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

// AWSSizeEndpoint handles the request to list available AWS sizes.
func AWSSizeEndpoint(settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSizeReq)
		settings, err := settingsProvider.GetGlobalSettings(ctx)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return providercommon.AWSSizes(req.Region, req.Architecture, settings.Spec.MachineDeploymentVMResourceQuota)
	}
}

// AWSSizeNoCredentialsEndpoint handles the request to list available AWS sizes.
func AWSSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return providercommon.AWSSizeNoCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID, "")
	}
}

// AWSSubnetEndpoint handles the request to list AWS availability subnets in a given vpc, using provided credentials.
func AWSSubnetEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSubnetReq)

		accessKeyID := req.AccessKeyID
		secretAccessKey := req.SecretAccessKey
		assumeRoleARN := req.AssumeRoleARN
		assumeRoleExternalID := req.AssumeRoleExternalID
		vpcID := req.VPC

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.AWS; credential != nil {
				accessKeyID = credential.AccessKeyID
				secretAccessKey = credential.SecretAccessKey
				assumeRoleARN = credential.AssumeRoleARN
				assumeRoleExternalID = credential.AssumeRoleExternalID
				vpcID = credential.VPCID
			}
		}

		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		subnetList, err := providercommon.ListAWSSubnets(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, vpcID, dc)
		if err != nil {
			return nil, err
		}
		if len(subnetList) > 0 {
			subnetList[0].IsDefaultSubnet = true
		}

		return subnetList, nil
	}
}

// AWSSubnetWithClusterCredentialsEndpoint handles the request to list AWS availability subnets in a given vpc, using credentials.
func AWSSubnetWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return providercommon.AWSSubnetNoCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

// AWSVPCEndpoint handles the request to list AWS VPC's, using provided credentials.
func AWSVPCEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSVPCReq)

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		credentials, err := getAWSCredentialsFromRequest(ctx, req.AWSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		return listAWSVPCS(ctx, credentials.accessKeyID, credentials.secretAccessKey, credentials.assumeRoleARN, credentials.assumeRoleExternalID, datacenter)
	}
}

// AWSSecurityGroupsEndpoint handles the request to list AWS Security Groups, using provided credentials.
func AWSSecurityGroupsEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSecurityGroupsReq)

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		credentials, err := getAWSCredentialsFromRequest(ctx, req.AWSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		return listSecurityGroup(ctx, credentials.accessKeyID, credentials.secretAccessKey, credentials.assumeRoleARN, credentials.assumeRoleExternalID, datacenter.Spec.AWS.Region, credentials.vpcID)
	}
}

func listSecurityGroup(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, vpc string) (*apiv1.AWSSecurityGroupList, error) {
	securityGroupList := &apiv1.AWSSecurityGroupList{}

	securityGroups, err := awsprovider.GetSecurityGroups(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, vpc)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get Security Groups: %v", err))
	}

	for _, sg := range securityGroups {
		securityGroupList.IDs = append(securityGroupList.IDs, *sg.GroupId)
	}

	return securityGroupList, nil
}

type awsCredentials struct {
	accessKeyID          string
	secretAccessKey      string
	assumeRoleARN        string
	assumeRoleExternalID string
	vpcID                string
}

func getAWSCredentialsFromRequest(ctx context.Context, req AWSCommonReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) (*awsCredentials, error) {
	accessKeyID := req.AccessKeyID
	secretAccessKey := req.SecretAccessKey
	assumeRoleARN := req.AssumeRoleARN
	assumeRoleExternalID := req.AssumeRoleExternalID
	vpcID := req.VPC

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if len(req.Credential) > 0 {
		preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
		}
		if credential := preset.Spec.AWS; credential != nil {
			accessKeyID = credential.AccessKeyID
			secretAccessKey = credential.SecretAccessKey
			assumeRoleARN = credential.AssumeRoleARN
			assumeRoleExternalID = credential.AssumeRoleExternalID
			vpcID = credential.VPCID
		}
	}

	return &awsCredentials{
		accessKeyID:          accessKeyID,
		secretAccessKey:      secretAccessKey,
		assumeRoleARN:        assumeRoleARN,
		assumeRoleExternalID: assumeRoleExternalID,
		vpcID:                vpcID,
	}, nil
}

func listAWSVPCS(ctx context.Context, accessKeyID, secretAccessKey string, assumeRoleARN string, assumeRoleExternalID string, datacenter *kubermaticv1.Datacenter) (apiv1.AWSVPCList, error) {
	if datacenter.Spec.AWS == nil {
		return nil, utilerrors.NewBadRequest("datacenter is not an AWS datacenter")
	}

	vpcsResults, err := awsprovider.GetVPCS(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, datacenter.Spec.AWS.Region)
	if err != nil {
		return nil, err
	}

	vpcs := apiv1.AWSVPCList{}
	for _, vpc := range vpcsResults {
		var tags []apiv1.AWSTag
		var cidrBlockList []apiv1.AWSVpcCidrBlockAssociation
		var Ipv6CidrBlocList []apiv1.AWSVpcIpv6CidrBlockAssociation
		var name string

		for _, tag := range vpc.Tags {
			tags = append(tags, apiv1.AWSTag{Key: *tag.Key, Value: *tag.Value})
			if *tag.Key == "Name" {
				name = *tag.Value
			}
		}

		for _, cidr := range vpc.CidrBlockAssociationSet {
			cidrBlock := apiv1.AWSVpcCidrBlockAssociation{
				AssociationID: *cidr.AssociationId,
				CidrBlock:     *cidr.CidrBlock,
			}
			if cidr.CidrBlockState != nil {
				cidrBlock.State = string(cidr.CidrBlockState.State)

				if cidr.CidrBlockState.StatusMessage != nil {
					cidrBlock.StatusMessage = *cidr.CidrBlockState.StatusMessage
				}
			}
			cidrBlockList = append(cidrBlockList, cidrBlock)
		}

		for _, cidr := range vpc.Ipv6CidrBlockAssociationSet {
			cidrBlock := apiv1.AWSVpcIpv6CidrBlockAssociation{
				AWSVpcCidrBlockAssociation: apiv1.AWSVpcCidrBlockAssociation{
					AssociationID: *cidr.AssociationId,
					CidrBlock:     *cidr.Ipv6CidrBlock,
				},
			}
			if cidr.Ipv6CidrBlockState != nil {
				cidrBlock.State = string(cidr.Ipv6CidrBlockState.State)
				if cidr.Ipv6CidrBlockState.StatusMessage != nil {
					cidrBlock.StatusMessage = *cidr.Ipv6CidrBlockState.StatusMessage
				}
			}

			Ipv6CidrBlocList = append(Ipv6CidrBlocList, cidrBlock)
		}

		vpcs = append(vpcs, apiv1.AWSVPC{
			Name:                        name,
			VpcID:                       *vpc.VpcId,
			CidrBlock:                   *vpc.CidrBlock,
			DhcpOptionsID:               *vpc.DhcpOptionsId,
			InstanceTenancy:             string(vpc.InstanceTenancy),
			IsDefault:                   *vpc.IsDefault,
			OwnerID:                     *vpc.OwnerId,
			State:                       string(vpc.State),
			Tags:                        tags,
			Ipv6CidrBlockAssociationSet: Ipv6CidrBlocList,
			CidrBlockAssociationSet:     cidrBlockList,
		})
	}

	return vpcs, err
}
