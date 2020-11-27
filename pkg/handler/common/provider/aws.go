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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
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

func AWSSizeNoCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
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

	return AWSSizes(dc.Spec.AWS.Region)
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

func AWSSizes(region string) (apiv1.AWSSizeList, error) {
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
