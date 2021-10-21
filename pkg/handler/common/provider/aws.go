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
	"strings"

	ec2service "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	ec2 "github.com/cristim/ec2-instances-info"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"
	machineconversions "k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var data *ec2.InstanceData

// Due to big amount of data we are loading AWS instance types only once. Do not edit it.
func init() {
	data, _ = ec2.Data()
}

func AWSSubnetNoCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.AWS == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
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

	subnetList, err := ListAWSSubnets(accessKeyID, secretAccessKey, cluster.Spec.Cloud.AWS.VPCID, dc)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, err
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return SetDefaultSubnet(machineDeployments, subnetList)
}

func AWSSizeNoCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID, architecture string) (interface{}, error) {
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.AWS == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
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
		return nil, errors.NewNotFound("cloud spec (dc) for ", clusterID)
	}

	settings, err := settingsProvider.GetGlobalSettings()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return AWSSizes(dc.Spec.AWS.Region, architecture, settings.Spec.MachineDeploymentVMResourceQuota)
}

func ListAWSSubnets(accessKeyID, secretAccessKey, vpcID string, datacenter *kubermaticv1.Datacenter) (apiv1.AWSSubnetList, error) {

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

func AWSSizes(region, architecture string, quota kubermaticv1.MachineDeploymentVMResourceQuota) (apiv1.AWSSizeList, error) {
	if data == nil {
		return nil, fmt.Errorf("AWS instance type data not initialized")
	}

	sizes := apiv1.AWSSizeList{}
	for _, i := range *data {
		pricing, ok := i.Pricing[region]
		if !ok {
			continue
		}
		price := pricing.Linux.OnDemand
		// not available
		if price == 0 {
			continue
		}

		if !isValidArchitecture(architecture, i.PhysicalProcessor) {
			continue
		}

		machineArchitecture := handlercommon.X64Architecture
		if isARM64Architecture(i.PhysicalProcessor) {
			machineArchitecture = handlercommon.ARM64Architecture
		}

		sizes = append(sizes, apiv1.AWSSize{
			Name:         i.InstanceType,
			PrettyName:   i.PrettyName,
			Memory:       i.Memory,
			VCPUs:        i.VCPU,
			GPUs:         i.GPU,
			Price:        price,
			Architecture: machineArchitecture,
		})
	}

	return filterAWSByQuota(sizes, quota), nil
}

func isARM64Architecture(physicalProcessor string) bool {
	// right now there is only one Arm-based processors: Graviton2
	return strings.Contains(physicalProcessor, "Graviton")
}

func isValidArchitecture(architecture, processorType string) bool {
	if architecture == handlercommon.ARM64Architecture {
		return isARM64Architecture(processorType)
	}
	if architecture == handlercommon.X64Architecture {
		return !isARM64Architecture(processorType)
	}
	// otherwise don't filter out
	return true
}

func filterAWSByQuota(instances apiv1.AWSSizeList, quota kubermaticv1.MachineDeploymentVMResourceQuota) apiv1.AWSSizeList {
	filteredRecords := apiv1.AWSSizeList{}

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range instances {
		keep := true

		// Filter too expensive instance types (>1$ per hour) if GPU not enabled
		if !quota.EnableGPU && r.Price > 1 {
			continue
		}

		if !handlercommon.FilterGPU(r.GPUs, quota.EnableGPU) {
			keep = false
		}

		if !handlercommon.FilterCPU(r.VCPUs, quota.MinCPU, quota.MaxCPU) {
			keep = false
		}
		if !handlercommon.FilterMemory(int(r.Memory), quota.MinRAM, quota.MaxRAM) {
			keep = false
		}

		if keep {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

func ListEKSClusters(ctx context.Context, accessKeyID, secretAccessKey, region string) (apiv2.EKSClusterList, error) {
	clusters := apiv2.EKSClusterList{}
	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return clusters, err
	}

	list, err := client.EKS.ListClusters(&eks.ListClustersInput{})
	if err != nil {
		return clusters, fmt.Errorf("cannot list clusters in region=%s: %w", region, err)
	}
	for _, f := range list.Clusters {
		clusters = append(clusters, apiv2.EKSCluster{Name: *f})
	}
	return clusters, nil
}

func ListEC2Regions(ctx context.Context, accessKeyID, secretAccessKey, endpoint string) (apiv2.Regions, error) {
	regionInput := &ec2service.DescribeRegionsInput{}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, endpoint)
	if err != nil {
		return nil, err
	}

	// Retrieves all regions/endpoints that work with EC2
	regionOutput, err := client.EC2.DescribeRegions(regionInput)
	if err != nil {
		return nil, fmt.Errorf("cannot list regions: %w", err)
	}

	var regionList []string
	for _, region := range regionOutput.Regions {
		regionList = append(regionList, *region.RegionName)
	}
	return regionList, nil
}
