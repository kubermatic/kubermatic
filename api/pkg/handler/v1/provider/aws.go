package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	awsProvider "github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
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
}

// AWSZoneReq represent a request for AWS zones.
// swagger:parameters listAWSZones
type AWSZoneReq struct {
	AWSCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
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

// DecodeAWSCommonReq decodes the base type for a AWS special endpoint request
func DecodeAWSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// DecodeAWSZoneReq decodes a request for a list of AWS zones
func DecodeAWSZoneReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSZoneReq

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

// AWSZoneEndpoint handles the request to list AWS availability zones in a given region, using provided credentials
func AWSZoneEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSZoneReq)

		keyID := req.AccessKeyID
		keySecret := req.SecretAccessKey

		if len(req.Credential) > 0 && credentialManager.GetPresets().AWS.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().AWS.Credentials {
				if credential.Name == req.Credential {
					keyID = credential.AccessKeyID
					keySecret = credential.SecretAccessKey
					break
				}
			}
		}

		return listAWSZones(ctx, keyID, keySecret, req.DC, seedsGetter)
	}
}

// AWSZoneNoCredentialsEndpoint handles the request to list AWS availability zones in a given region, using credentials from a given datacenter
func AWSZoneNoCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.AWS == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		keyID := cluster.Spec.Cloud.AWS.AccessKeyID
		keySecret := cluster.Spec.Cloud.AWS.SecretAccessKey
		return listAWSZones(ctx, keyID, keySecret, cluster.Spec.Cloud.DatacenterName, seedsGetter)
	}
}

func listAWSZones(ctx context.Context, keyID, keySecret, datacenterName string, seedsGetter provider.SeedsGetter) (apiv1.AWSZoneList, error) {
	zones := apiv1.AWSZoneList{}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("the %s is not AWS datacenter", datacenterName)
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter)
	if err != nil {
		return nil, err
	}

	zoneResults, err := ec2.GetAvailabilityZonesInRegion(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		AWS: &kubermaticv1.AWSCloudSpec{
			AccessKeyID:     keyID,
			SecretAccessKey: keySecret,
		},
	}, datacenter.Spec.AWS.Region)
	if err != nil {
		return nil, err
	}

	for _, z := range zoneResults {
		zones = append(zones, apiv1.AWSZone{Name: *z.ZoneName})
	}

	return zones, err
}

// AWSSubnetEndpoint handles the request to list AWS availability subnets in a given vpc, using provided credentials
func AWSSubnetEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSubnetReq)

		keyID := req.AccessKeyID
		keySecret := req.SecretAccessKey

		if len(req.Credential) > 0 && credentialManager.GetPresets().AWS.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().AWS.Credentials {
				if credential.Name == req.Credential {
					keyID = credential.AccessKeyID
					keySecret = credential.SecretAccessKey
					break
				}
			}
		}

		return listAWSSubnets(ctx, keyID, keySecret, req.DC, req.VPC, seedsGetter)
	}
}

// AWSSubnetNoCredentialsEndpoint handles the request to list AWS availability subnets in a given vpc, using credentials
func AWSSubnetNoCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.AWS == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		keyID := cluster.Spec.Cloud.AWS.AccessKeyID
		keySecret := cluster.Spec.Cloud.AWS.SecretAccessKey
		return listAWSSubnets(ctx, keyID, keySecret, cluster.Spec.Cloud.DatacenterName, cluster.Spec.Cloud.AWS.VPCID, seedsGetter)
	}
}

func listAWSSubnets(ctx context.Context, keyID, keySecret, datacenterName string, vpcID string, seedsGetter provider.SeedsGetter) (apiv1.AWSSubnetList, error) {
	subnets := apiv1.AWSSubnetList{}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("the %s is not AWS datacenter", datacenterName)
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter)
	if err != nil {
		return nil, fmt.Errorf("couldn't create cloud provider: %v", err)
	}

	subnetResults, err := ec2.GetSubnets(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		AWS: &kubermaticv1.AWSCloudSpec{
			AccessKeyID:     keyID,
			SecretAccessKey: keySecret,
		},
	}, vpcID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get subnets: %v", err)
	}

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

	return subnets, err
}

// AWSVPCEndpoint handles the request to list AWS VPC's, using provided credentials
func AWSVPCEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSVPCReq)

		keyID := req.AccessKeyID
		keySecret := req.SecretAccessKey

		if len(req.Credential) > 0 && credentialManager.GetPresets().AWS.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().AWS.Credentials {
				if credential.Name == req.Credential {
					keyID = credential.AccessKeyID
					keySecret = credential.SecretAccessKey
					break
				}
			}
		}

		return listAWSVPCS(keyID, keySecret, req.DC, seedsGetter)
	}
}

func listAWSVPCS(keyID, keySecret, datacenterName string, seedsGetter provider.SeedsGetter) (apiv1.AWSVPCList, error) {
	vpcs := apiv1.AWSVPCList{}
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("the %s is not AWS datacenter", datacenterName)
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter)
	if err != nil {
		return nil, fmt.Errorf("couldn't create cloud provider: %v", err)
	}

	vpcsResults, err := ec2.GetVPCS(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		AWS: &kubermaticv1.AWSCloudSpec{
			AccessKeyID:     keyID,
			SecretAccessKey: keySecret,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get vpc's: %v", err)
	}

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
