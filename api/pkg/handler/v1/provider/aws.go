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
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	awsProvider "github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

// AWSSizeReq represent a request for AWS VM sizes.
// swagger:parameters listAWSSizes
type AWSSizeReq struct {
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

// AWSSizeEndpoint handles the request to list available AWS sizes.
func AWSSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSSizeReq)
		return awsSizes(req.Region)
	}
}

// AWSSizeNoCredentialsEndpoint handles the request to list available AWS sizes.
func AWSSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
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

		dc, err := dc.GetDatacenter(seedsGetter, cluster.Spec.Cloud.DatacenterName)
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
	data, err := ec2.Data()
	if err != nil {
		return nil, err
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

			// Filter out unavailable or too expensive instance types (>1.5$ per hour). TODO: Parametrize cost?
			price := pricing.Linux.OnDemand
			if price == 0 || price > 1.5 {
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

// AWSZoneEndpoint handles the request to list AWS availability zones in a given region, using provided credentials
func AWSZoneEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter, privilegedSeedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
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

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		seed, datacenter, err := provider.DatacenterFromSeedMap(seeds, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest("%v", err)
		}
		privilegedSeedClient, err := privilegedSeedClientGetter(seed)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to get client for seed: %v", err))
		}
		return listAWSZones(ctx, keyID, keySecret, datacenter, privilegedSeedClient, nil)
	}
}

// AWSZoneWithClusterCredentialsEndpoint handles the request to list AWS availability zones in a given region, using credentials from a given datacenter
func AWSZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
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

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		_, datacenter, err := provider.DatacenterFromSeedMap(seeds, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.NewBadRequest("%v", err)
		}

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "clusterprovider is not a kubernetesprovider.Clusterprovider")
		}

		keyID := cluster.Spec.Cloud.AWS.AccessKeyID
		keySecret := cluster.Spec.Cloud.AWS.SecretAccessKey
		return listAWSZones(ctx, keyID, keySecret, datacenter, assertedClusterProvider.GetSeedClusterAdminRuntimeClient(), cluster.Spec.Cloud.AWS.CredentialsReference)
	}
}

func listAWSZones(ctx context.Context, keyID, keySecret string, datacenter *kubermaticv1.Datacenter, privilegedSeedClient ctrlruntimeclient.Client, credRef *providerconfig.GlobalSecretKeySelector) (apiv1.AWSZoneList, error) {
	zones := apiv1.AWSZoneList{}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("cluster is not in an AWS Datacenter")
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter, provider.SecretKeySelectorValueFuncFactory(ctx, privilegedSeedClient))
	if err != nil {
		return nil, err
	}

	zoneResults, err := ec2.GetAvailabilityZonesInRegion(kubermaticv1.CloudSpec{
		AWS: &kubermaticv1.AWSCloudSpec{
			AccessKeyID:          keyID,
			SecretAccessKey:      keySecret,
			CredentialsReference: credRef,
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
func AWSSubnetEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter, privilegedSeedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
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

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		seed, dc, err := provider.DatacenterFromSeedMap(seeds, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		privilegedSeedClient, err := privilegedSeedClientGetter(seed)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to get client for seed: %v", err))
		}

		return listAWSSubnets(ctx, keyID, keySecret, req.VPC, dc, privilegedSeedClient, nil)
	}
}

// AWSSubnetWithClusterCredentialsEndpoint handles the request to list AWS availability subnets in a given vpc, using credentials
func AWSSubnetWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
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

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		_, dc, err := provider.DatacenterFromSeedMap(seeds, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		keyID := cluster.Spec.Cloud.AWS.AccessKeyID
		keySecret := cluster.Spec.Cloud.AWS.SecretAccessKey
		return listAWSSubnets(ctx, keyID, keySecret, cluster.Spec.Cloud.AWS.VPCID, dc, assertedClusterProvider.GetSeedClusterAdminRuntimeClient(), cluster.Spec.Cloud.AWS.CredentialsReference)
	}
}

func listAWSSubnets(ctx context.Context, keyID, keySecret, vpcID string, datacenter *kubermaticv1.Datacenter, privilegedSeedClient ctrlruntimeclient.Client, credRef *providerconfig.GlobalSecretKeySelector) (apiv1.AWSSubnetList, error) {
	subnets := apiv1.AWSSubnetList{}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("datacenter is not an AWS datacenter")
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter, provider.SecretKeySelectorValueFuncFactory(ctx, privilegedSeedClient))
	if err != nil {
		return nil, fmt.Errorf("couldn't create cloud provider: %v", err)
	}

	subnetResults, err := ec2.GetSubnets(kubermaticv1.CloudSpec{
		AWS: &kubermaticv1.AWSCloudSpec{
			AccessKeyID:          keyID,
			SecretAccessKey:      keySecret,
			CredentialsReference: credRef,
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
func AWSVPCEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter, privilegedSeedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
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

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		seed, datacenter, err := provider.DatacenterFromSeedMap(seeds, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		privilegedSeedClient, err := privilegedSeedClientGetter(seed)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to get client for seed: %v", err))
		}

		return listAWSVPCS(ctx, keyID, keySecret, datacenter, privilegedSeedClient)
	}
}

func listAWSVPCS(ctx context.Context, keyID, keySecret string, datacenter *kubermaticv1.Datacenter, privilegedSeedClient ctrlruntimeclient.Client) (apiv1.AWSVPCList, error) {
	vpcs := apiv1.AWSVPCList{}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("datacenter is not an AWS datacenter")
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter, provider.SecretKeySelectorValueFuncFactory(ctx, privilegedSeedClient))
	if err != nil {
		return nil, fmt.Errorf("couldn't create cloud provider: %v", err)
	}

	vpcsResults, err := ec2.GetVPCS(kubermaticv1.CloudSpec{
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
