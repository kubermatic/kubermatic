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
	"fmt"
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

type openstack struct{}

var _ clusterSpec = openstack{}

func (a openstack) getImage(osName string) string {
	switch osName {
	case "ubuntu":
		return "Ubuntu Focal 20.04 (2021-07-01)"
	case "centos":
		return "CentOS 8 (2021-07-05)"
	case "flatcar":
		return "Flatcar Stable (2022-05-10)"
	default:
		return fmt.Sprintf("unknown os: %s", osName)
	}
}

func (a openstack) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Openstack: &models.OpenstackNodeSpec{
			AvailabilityZone:          "fes1",
			Flavor:                    pointer.String("l1c.small"),
			Image:                     pointer.String("Ubuntu Focal 20.04 (2021-07-01)"),
			InstanceReadyCheckPeriod:  "5s",
			InstanceReadyCheckTimeout: "120s",
			RootDiskSizeGB:            0,
			Tags:                      nil,
			UseFloatingIP:             true,
		},
	}
}

func (a openstack) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "syseleven-fes1",
		Openstack: &models.OpenstackCloudSpec{
			ApplicationCredentialID:     "",
			ApplicationCredentialSecret: "",
			Domain:                      os.Getenv("OS_USER_DOMAIN_NAME"),
			FloatingIPPool:              os.Getenv("OS_FLOATING_IP_POOL"),
			IPV6SubnetID:                "",
			IPV6SubnetPool:              "",
			Network:                     "",
			NodePortsAllowedIPRange:     "0.0.0.0/0",
			Password:                    os.Getenv("OS_PASSWORD"),
			Project:                     os.Getenv("OS_PROJECT_NAME"),
			ProjectID:                   "",
			RouterID:                    "",
			SecurityGroups:              "default",
			SubnetID:                    "",
			Token:                       "",
			UseOctavia:                  false,
			UseToken:                    false,
			Username:                    os.Getenv("OS_USERNAME"),
			CredentialsReference:        nil,
			NodePortsAllowedIPRanges: &models.NetworkRanges{
				CIDRBlocks: []string{
					"0.0.0.0/0",
					"::/0",
				},
			},
		},
	}
}

type hetzner struct{}

var _ clusterSpec = hetzner{}

func (a hetzner) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Hetzner: &models.HetznerNodeSpec{
			Type: pointer.String("cx21"),
		},
	}
}

func (a hetzner) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "hetzner-hel1",
		Hetzner: &models.HetznerCloudSpec{
			Token: os.Getenv("HETZNER_TOKEN"),
		},
	}
}

type do struct{}

var _ clusterSpec = do{}

func (a do) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Digitalocean: &models.DigitaloceanNodeSpec{
			Backups:    false,
			IPV6:       true, // TODO: Could be set to false once MC is fixed.
			Monitoring: false,
			Size:       pointer.String("c-2"),
		},
	}
}

func (a do) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "do-fra1",
		Digitalocean: &models.DigitaloceanCloudSpec{
			Token: os.Getenv("DO_TOKEN"),
		},
	}
}

type equinix struct{}

var _ clusterSpec = equinix{}

func (a equinix) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Packet: &models.PacketNodeSpec{
			InstanceType: pointer.String("c3.small.x86"),
		},
	}
}

func (a equinix) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "packet-am",
		Packet: &models.PacketCloudSpec{
			APIKey:    os.Getenv("METAL_AUTH_TOKEN"),
			ProjectID: os.Getenv("METAL_PROJECT_ID"),
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

func rockyLinux() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		RockyLinux: &models.RockyLinuxSpec{},
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
