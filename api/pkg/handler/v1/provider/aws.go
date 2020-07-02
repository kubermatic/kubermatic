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

	ec2 "github.com/cristim/ec2-instances-info"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	machineconversions "github.com/kubermatic/kubermatic/api/pkg/machine"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	awsprovider "github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var data *ec2.InstanceData

// Due to big amount of data we are loading AWS instance types only once. Do not edit it.
func init() {
	data, _ = ec2.Data()
}

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
}

// AWSSubnetReq represent a request for AWS subnets.
// swagger:parameters listAWSSubnets
type AWSSubnetReq struct {
	AWSCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
	// in: header
	// name: VPC
	VPC string `json:"vpc"`
}

// AWSVPCReq represent a request for AWS vpc's.
// swagger:parameters listAWSVPCS
type AWSVPCReq struct {
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
}

// DecodeAWSSizesReq decodes the base type for a AWS special endpoint request
func DecodeAWSSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSSizeReq
	req.Region = r.Header.Get("Region")
	return req, nil
}

// DecodeAWSCommonReq decodes the base type for a AWS special endpoint request
func DecodeAWSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// DecodeAWSSubnetReq decodes a request for a list of AWS subnets
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

	req.VPC = r.Header.Get("VPC")

	return req, nil
}

// DecodeAWSVPCReq decodes a request for a list of AWS vpc's
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

// AWSSizeEndpoint handles the request to list available AWS sizes.
func AWSSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSizeReq)
		return awsSizes(req.Region)
	}
}

// AWSSizeNoCredentialsEndpoint handles the request to list available AWS sizes.
func AWSSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.AWS == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		dc, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		if dc.Spec.AWS == nil {
			return nil, errors.NewNotFound("cloud spec (dc) for ", req.ClusterID)
		}

		return awsSizes(dc.Spec.AWS.Region)
	}
}

func awsSizes(region string) (apiv1.AWSSizeList, error) {
	if data == nil {
		return nil, fmt.Errorf("AWS instance type data not initialized")
	}

	sizes := apiv1.AWSSizeList{}
	for _, i := range *data {
		// TODO: Make the check below more generic, working for all the providers. It is needed as the pods
		//  with memory under 2 GB will be full with required pods like kube-proxy, CNI etc.
		if i.Memory >= 2 {
			pricing, ok := i.Pricing[region]
			if !ok {
				continue
			}

			// Filter out unavailable or too expensive instance types (>1$ per hour).
			price := pricing.Linux.OnDemand
			if price == 0 || price > 1 {
				continue
			}

			sizes = append(sizes, apiv1.AWSSize{
				Name:       i.InstanceType,
				PrettyName: i.PrettyName,
				Memory:     i.Memory,
				VCPUs:      i.VCPU,
				Price:      price,
			})
		}
	}

	return sizes, nil
}

// AWSSubnetEndpoint handles the request to list AWS availability subnets in a given vpc, using provided credentials
func AWSSubnetEndpoint(presetsProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSubnetReq)

		accessKeyID := req.AccessKeyID
		secretAccessKey := req.SecretAccessKey
		vpcID := req.VPC

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.AWS; credential != nil {
				accessKeyID = credential.AccessKeyID
				secretAccessKey = credential.SecretAccessKey
				vpcID = credential.VPCID
			}
		}

		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		subnetList, err := listAWSSubnets(accessKeyID, secretAccessKey, vpcID, dc)
		if err != nil {
			return nil, err
		}
		if len(subnetList) > 0 {
			subnetList[0].IsDefaultSubnet = true
		}

		return subnetList, nil
	}
}

// AWSSubnetWithClusterCredentialsEndpoint handles the request to list AWS availability subnets in a given vpc, using credentials
func AWSSubnetWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.AWS == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		accessKeyID, secretAccessKey, err := awsprovider.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}

		subnetList, err := listAWSSubnets(accessKeyID, secretAccessKey, cluster.Spec.Cloud.AWS.VPCID, dc)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, err
		}

		machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
		if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return SetDefaultSubnet(machineDeployments, subnetList)
	}
}

func SetDefaultSubnet(machineDeployments *clusterv1alpha1.MachineDeploymentList, subnets apiv1.AWSSubnetList) (apiv1.AWSSubnetList, error) {
	if len(subnets) == 0 {
		return nil, fmt.Errorf("the subnet list can not be empty")
	}
	if machineDeployments == nil {
		return nil, fmt.Errorf("the machine deployment list can not be nil")
	}

	machinesForAZ := map[string]int32{}

	for _, subnet := range subnets {
		machinesForAZ[subnet.AvailabilityZone] = 0
	}

	var machineCounter int32
	var replicas int32
	for _, md := range machineDeployments.Items {
		cloudSpec, err := machineconversions.GetAPIV2NodeCloudSpec(md.Spec.Template.Spec)
		if err != nil {
			return nil, fmt.Errorf("failed to get node cloud spec from machine deployment: %v", err)
		}
		if cloudSpec.AWS == nil {
			return nil, errors.NewBadRequest("cloud spec missing")
		}
		if md.Spec.Replicas != nil {
			replicas = *md.Spec.Replicas
		}

		machinesForAZ[cloudSpec.AWS.AvailabilityZone] += replicas
		machineCounter += replicas
	}
	// If no machines exist, set the first as a default
	if machineCounter == 0 {
		subnets[0].IsDefaultSubnet = true
		return subnets, nil
	}

	// If machines exist, but there are AZs in the region without machines
	// set a subnet in an AZ that doesn't yet have machines
	for i, subnet := range subnets {
		if machinesForAZ[subnet.AvailabilityZone] == 0 {
			subnets[i].IsDefaultSubnet = true
			return subnets, nil
		}
	}

	// If we already have machines for all AZs, just set the first
	subnets[0].IsDefaultSubnet = true
	return subnets, nil
}

func listAWSSubnets(accessKeyID, secretAccessKey, vpcID string, datacenter *kubermaticv1.Datacenter) (apiv1.AWSSubnetList, error) {

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("datacenter is not an AWS datacenter")
	}

	subnetResults, err := awsprovider.GetSubnets(accessKeyID, secretAccessKey, datacenter.Spec.AWS.Region, vpcID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get subnets: %v", err)
	}

	subnets := apiv1.AWSSubnetList{}
	for _, s := range subnetResults {
		subnetTags := []apiv1.AWSTag{}
		var subnetName string
		for _, v := range s.Tags {
			subnetTags = append(subnetTags, apiv1.AWSTag{Key: *v.Key, Value: *v.Value})
			if *v.Key == "Name" {
				subnetName = *v.Value
			}
		}

		// Even though Ipv6CidrBlockAssociationSet is defined as []*VpcIpv6CidrBlockAssociation in AWS,
		// it is currently not possible to use more than one cidr block.
		// In case there are blocks with state != associated, we check for it and use the first entry
		// that matches condition.
		var subnetIpv6 string
		for _, v := range s.Ipv6CidrBlockAssociationSet {
			if *v.Ipv6CidrBlockState.State == "associated" {
				subnetIpv6 = *v.Ipv6CidrBlock
				break
			}
		}

		subnets = append(subnets, apiv1.AWSSubnet{
			Name:                    subnetName,
			ID:                      *s.SubnetId,
			AvailabilityZone:        *s.AvailabilityZone,
			AvailabilityZoneID:      *s.AvailabilityZoneId,
			IPv4CIDR:                *s.CidrBlock,
			IPv6CIDR:                subnetIpv6,
			Tags:                    subnetTags,
			State:                   *s.State,
			AvailableIPAddressCount: *s.AvailableIpAddressCount,
			DefaultForAz:            *s.DefaultForAz,
		})

	}

	return subnets, nil
}

// AWSVPCEndpoint handles the request to list AWS VPC's, using provided credentials
func AWSVPCEndpoint(presetsProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSVPCReq)

		accessKeyID := req.AccessKeyID
		secretAccessKey := req.SecretAccessKey

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.AWS; credential != nil {
				accessKeyID = credential.AccessKeyID
				secretAccessKey = credential.SecretAccessKey
			}
		}

		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		return listAWSVPCS(accessKeyID, secretAccessKey, datacenter)
	}
}

func listAWSVPCS(accessKeyID, secretAccessKey string, datacenter *kubermaticv1.Datacenter) (apiv1.AWSVPCList, error) {

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("datacenter is not an AWS datacenter")
	}

	vpcsResults, err := awsprovider.GetVPCS(accessKeyID, secretAccessKey, datacenter.Spec.AWS.Region)
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
				if cidr.CidrBlockState.State != nil {
					cidrBlock.State = *cidr.CidrBlockState.State
				}
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
				if cidr.Ipv6CidrBlockState.State != nil {
					cidrBlock.State = *cidr.Ipv6CidrBlockState.State
				}
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
			InstanceTenancy:             *vpc.InstanceTenancy,
			IsDefault:                   *vpc.IsDefault,
			OwnerID:                     *vpc.OwnerId,
			State:                       *vpc.State,
			Tags:                        tags,
			Ipv6CidrBlockAssociationSet: Ipv6CidrBlocList,
			CidrBlockAssociationSet:     cidrBlockList,
		})

	}

	return vpcs, err
}
