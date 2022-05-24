//go:build dualstack

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

package dualstack

import (
	"context"
	"net/http"
	"os"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/utils/pointer"
)

type clusterNetworkingConfig models.ClusterNetworkingConfig

func defaultClusterNetworkingConfig() clusterNetworkingConfig {
	c := models.ClusterNetworkingConfig{
		NodeCIDRMaskSizeIPV4: 20,
		NodeCIDRMaskSizeIPV6: 100,
		ProxyMode:            "ebpf",
		IPFamily:             "IPv4+IPv6",
		Pods: &models.NetworkRanges{
			CIDRBlocks: []string{"172.25.0.0/16", "fd00::/99"},
		},
		Services: &models.NetworkRanges{
			CIDRBlocks: []string{"10.240.16.0/20", "fd03::/120"},
		},
		KonnectivityEnabled: true,
	}

	return clusterNetworkingConfig(c)
}

func (c clusterNetworkingConfig) WithProxyMode(proxyMode string) clusterNetworkingConfig {
	c.ProxyMode = proxyMode
	return c
}

type createMachineDeploymentParams project.CreateMachineDeploymentParams

func defaultCreateMachineDeploymentParams() createMachineDeploymentParams {
	md := project.CreateMachineDeploymentParams{
		Body: &models.NodeDeployment{
			Name: "name",
			Spec: &models.NodeDeploymentSpec{
				Replicas: pointer.Int32(1),
				Template: &models.NodeSpec{
					SSHUserName:     "root",
					Cloud:           nil, // must fill in later
					OperatingSystem: nil, // must fill in later
				},
			},
		},
		ClusterID:  "clusterID",
		ProjectID:  "projectID",
		Context:    context.Background(),
		HTTPClient: http.DefaultClient,
	}
	return createMachineDeploymentParams(md)
}

func (md createMachineDeploymentParams) WithName(name string) createMachineDeploymentParams {
	md.Body.Name = name
	return md
}

func (md createMachineDeploymentParams) WithClusterID(clusterID string) createMachineDeploymentParams {
	md.ClusterID = clusterID
	return md
}

func (md createMachineDeploymentParams) WithProjectID(projectID string) createMachineDeploymentParams {
	md.ProjectID = projectID
	return md
}

func (md createMachineDeploymentParams) WithOS(os models.OperatingSystemSpec) createMachineDeploymentParams {
	md.Body.Spec.Template.OperatingSystem = &os
	return md
}

func (md createMachineDeploymentParams) WithNodeSpec(nodeSpec models.NodeCloudSpec) createMachineDeploymentParams {
	md.Body.Spec.Template.Cloud = &nodeSpec
	return md
}

type createClusterRequest models.CreateClusterSpec

func defaultClusterRequest() createClusterRequest {
	netConfig := models.ClusterNetworkingConfig(defaultClusterNetworkingConfig())
	clusterSpec := models.CreateClusterSpec{}
	clusterSpec.Cluster = &models.Cluster{
		Type: "kubernetes",
		Name: "test-default-azure-cluster",
		Spec: &models.ClusterSpec{
			Cloud: &models.CloudSpec{
				DatacenterName: "azure-westeurope",
				Azure: &models.AzureCloudSpec{
					ClientID:        os.Getenv("AZURE_CLIENT_ID"),
					ClientSecret:    os.Getenv("AZURE_CLIENT_SECRET"),
					SubscriptionID:  os.Getenv("AZURE_SUBSCRIPTION_ID"),
					TenantID:        os.Getenv("AZURE_TENANT_ID"),
					LoadBalancerSKU: "standard",
				},
			},
			CniPlugin: &models.CNIPluginSettings{
				Version: "v1.11",
				Type:    "cilium",
			},
			ClusterNetwork: &netConfig,
			Version:        models.Semver(utils.KubernetesVersion()),
		},
	}

	return createClusterRequest(clusterSpec)
}

func (c createClusterRequest) WithCNI(cni models.CNIPluginSettings) createClusterRequest {
	c.Cluster.Spec.CniPlugin = &cni
	return c
}

func (c createClusterRequest) WithOS(os models.OperatingSystemSpec) createClusterRequest {
	c.NodeDeployment.Spec.Template.OperatingSystem = &os
	return c
}

func (c createClusterRequest) WithNode(node models.NodeCloudSpec) createClusterRequest {
	c.NodeDeployment.Spec.Template.Cloud = &node
	return c
}

func (c createClusterRequest) WithCloud(cloud models.CloudSpec) createClusterRequest {
	c.Cluster.Spec.Cloud = &cloud
	return c
}

func (c createClusterRequest) WithName(name string) createClusterRequest {
	c.Cluster.Name = name
	return c
}

func (c createClusterRequest) WithNetworkConfig(netConfig models.ClusterNetworkingConfig) createClusterRequest {
	c.Cluster.Spec.ClusterNetwork = &netConfig
	return c
}

// cloud providers

type clusterSpec interface {
	NodeSpec() models.NodeCloudSpec
	CloudSpec() models.CloudSpec
}

type azure struct{}

var _ clusterSpec = azure{}

func (a azure) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Azure: &models.AzureNodeSpec{
			AssignAvailabilitySet: true,
			AssignPublicIP:        true,
			DataDiskSize:          int32(30),
			OSDiskSize:            70,
			Size:                  pointer.String("Standard_B2s"),
		},
	}
}

func (a azure) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "azure-westeurope",
		Azure: &models.AzureCloudSpec{
			ClientID:        os.Getenv("AZURE_CLIENT_ID"),
			ClientSecret:    os.Getenv("AZURE_CLIENT_SECRET"),
			SubscriptionID:  os.Getenv("AZURE_SUBSCRIPTION_ID"),
			TenantID:        os.Getenv("AZURE_TENANT_ID"),
			LoadBalancerSKU: "standard",
		},
	}
}

type aws struct{}

var _ clusterSpec = aws{}

func (a aws) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Aws: &models.AWSNodeSpec{
			AMI:                           "",
			AssignPublicIP:                true,
			AvailabilityZone:              "eu-central-1b",
			InstanceType:                  pointer.String("t3a.small"),
			IsSpotInstance:                false,
			SpotInstanceMaxPrice:          "",
			SpotInstancePersistentRequest: false,
			SubnetID:                      "subnet-01d796b6b81cab01c",
			VolumeSize:                    pointer.Int64(64),
			VolumeType:                    pointer.String("standard"),
		},
	}
}

func (a aws) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "aws-eu-central-1a",
		Aws: &models.AWSCloudSpec{
			AccessKeyID:             os.Getenv("AWS_ACCESS_KEY_ID"),
			ControlPlaneRoleARN:     "",
			InstanceProfileName:     "",
			NodePortsAllowedIPRange: "0.0.0.0/0",
			RouteTableID:            "",
			SecretAccessKey:         os.Getenv("AWS_SECRET_ACCESS_KEY"),
			SecurityGroupID:         "",
			VPCID:                   "vpc-819f62e9",
		},
	}
}

type gcp struct{}

var _ clusterSpec = gcp{}

func (a gcp) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Gcp: &models.GCPNodeSpec{
			DiskSize:    25,
			DiskType:    "pd-standard",
			MachineType: "e2-highcpu-2",
			Preemptible: false,
			Zone:        "europe-west3-a",
		},
	}
}

func (a gcp) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "gcp-westeurope",
		Gcp: &models.GCPCloudSpec{
			Network:        "global/networks/dualstack",
			ServiceAccount: os.Getenv("GOOGLE_SERVICE_ACCOUNT"),
			Subnetwork:     "projects/kubermatic-dev/regions/europe-west3/subnetworks/dualstack-europe-west3",
		},
	}
}

// operating systems

func ubuntu() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Ubuntu: &models.UbuntuSpec{},
	}
}

func rhel() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Rhel: &models.RHELSpec{},
	}
}

func sles() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Sles: &models.SLESSpec{},
	}
}

func centos() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Centos: &models.CentOSSpec{},
	}
}

func flatcar() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Flatcar: &models.FlatcarSpec{},
	}
}

// cnis

func cilium() models.CNIPluginSettings {
	return models.CNIPluginSettings{
		Version: "v1.11",
		Type:    "cilium",
	}
}

func canal() models.CNIPluginSettings {
	return models.CNIPluginSettings{
		Type:    "canal",
		Version: "v3.22",
	}
}
