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

package v2

import (
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

// Constraint represents a gatekeeper Constraint
// swagger:model Constraint
type Constraint struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`

	Spec   kubermaticv1.ConstraintSpec `json:"spec"`
	Status *ConstraintStatus           `json:"status,omitempty"`
}

// ConstraintStatus represents a constraint status which holds audit info.
type ConstraintStatus struct {
	Enforcement    string      `json:"enforcement,omitempty"`
	AuditTimestamp string      `json:"auditTimestamp,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
	Synced         *bool       `json:"synced,omitempty"`
}

// Violation represents a gatekeeper constraint violation.
type Violation struct {
	EnforcementAction string `json:"enforcementAction,omitempty"`
	Kind              string `json:"kind,omitempty"`
	Message           string `json:"message,omitempty"`
	Name              string `json:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
}

type ExternalClusterState string
type ExternalClusterMDState string

const (
	// ProvisioningExternalClusterState state indicates the cluster is being created.
	ProvisioningExternalClusterState ExternalClusterState = "Provisioning"

	// StoppedExternalClusterState state indicates the cluster is stopped, this state is specific to AKS clusters.
	StoppedExternalClusterState ExternalClusterState = "Stopped"

	// StoppingExternalClusterState state indicates the cluster is stopping, this state is specific to AKS clusters.
	StoppingExternalClusterState ExternalClusterState = "Stopping"

	// RunningExternalClusterState state indicates the cluster has been created and is fully usable.
	RunningExternalClusterState ExternalClusterState = "Running"

	// ReconcilingExternalClusterState state indicates that some work is actively being done on the cluster, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	ReconcilingExternalClusterState ExternalClusterState = "Reconciling"

	// DeletingExternalClusterState state indicates the cluster is being deleted.
	DeletingExternalClusterState ExternalClusterState = "Deleting"

	// StartingExternalClusterState state indicates the cluster is starting.
	StartingExternalClusterState ExternalClusterState = "Starting"

	// UnknownExternalClusterState indicates undefined state.
	UnknownExternalClusterState ExternalClusterState = "Unknown"

	// ErrorExternalClusterState state indicates the cluster is unusable. It will be automatically deleted. Details can be found in the
	// `statusMessage` field.
	ErrorExternalClusterState ExternalClusterState = "Error"

	// WarningExternalClusterState state indicates the cluster is usable but with some warnings. Details can be found in the
	// `statusMessage` field.
	WarningExternalClusterState ExternalClusterState = "Warning"
)

const (
	// ProvisioningExternalClusterMDState state indicates the cluster machine deployment is being created.
	ProvisioningExternalClusterMDState ExternalClusterMDState = "Provisioning"

	// RunningExternalClusterMDState state indicates the cluster machine deployment has been created and is fully usable.
	RunningExternalClusterMDState ExternalClusterMDState = "Running"

	// StoppedExternalClusterMDState state indicates the cluster machine deployment is stopped. This state is specific to AKS clusters.
	StoppedExternalClusterMDState ExternalClusterMDState = "Stopped"

	// ReconcilingExternalClusterMDState state indicates that some work is actively being done on the machine deployment, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	ReconcilingExternalClusterMDState ExternalClusterMDState = "Reconciling"

	// DeletingExternalClusterMDState state indicates the machine deployment is being deleted.
	DeletingExternalClusterMDState ExternalClusterMDState = "Deleting"

	// StartingExternalClusterMDState state indicates the cluster machine deployment is starting.
	StartingExternalClusterMDState ExternalClusterMDState = "Starting"

	// UnknownExternalClusterMDState indicates undefined state.
	UnknownExternalClusterMDState ExternalClusterMDState = "Unknown"

	// ErrorExternalClusterMDState state indicates the machine deployment is unusable. It will be automatically deleted. Details can be found in the
	// `statusMessage` field.
	ErrorExternalClusterMDState ExternalClusterMDState = "Error"

	// WarningExternalClusterMDState state indicates the machine deployment is usable but with some warnings. Details can be found in the
	// `statusMessage` field.
	WarningExternalClusterMDState ExternalClusterMDState = "Warning"
)

// ExternalClusterStatus defines the external cluster status.
type ExternalClusterStatus struct {
	State         ExternalClusterState `json:"state"`
	StatusMessage string               `json:"statusMessage,omitempty"`
	AKS           *AKSClusterStatus    `json:"aks,omitempty"`
}

// ExternalClusterCloudSpec represents an object holding cluster cloud details
// swagger:model ExternalClusterCloudSpec
type ExternalClusterCloudSpec struct {
	GKE          *GKECloudSpec     `json:"gke,omitempty"`
	EKS          *EKSCloudSpec     `json:"eks,omitempty"`
	AKS          *AKSCloudSpec     `json:"aks,omitempty"`
	KubeOne      *KubeOneSpec      `json:"kubeOne,omitempty"`
	BringYourOwn *BringYourOwnSpec `json:"bringYourOwn,omitempty"`
}

type BringYourOwnSpec struct{}

type KubeOneSpec struct {
	// Manifest Base64 encoded manifest
	Manifest         string            `json:"manifest,omitempty"`
	SSHKey           KubeOneSSHKey     `json:"sshKey,omitempty"`
	ContainerRuntime string            `json:"containerRuntime,omitempty"`
	CloudSpec        *KubeOneCloudSpec `json:"cloudSpec,omitempty"`
}

// SSHKeySpec represents the details of a ssh key.
type KubeOneSSHKey struct {
	// PrivateKey Base64 encoded privateKey
	PrivateKey string `json:"privateKey,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

type KubeOneCloudSpec struct {
	AWS                 *KubeOneAWSCloudSpec                 `json:"aws,omitempty"`
	GCP                 *KubeOneGCPCloudSpec                 `json:"gcp,omitempty"`
	Azure               *KubeOneAzureCloudSpec               `json:"azure,omitempty"`
	DigitalOcean        *KubeOneDigitalOceanCloudSpec        `json:"digitalocean,omitempty"`
	OpenStack           *KubeOneOpenStackCloudSpec           `json:"openstack,omitempty"`
	Hetzner             *KubeOneHetznerCloudSpec             `json:"hetzner,omitempty"`
	VSphere             *KubeOneVSphereCloudSpec             `json:"vsphere,omitempty"`
	VMwareCloudDirector *KubeOneVMwareCloudDirectorCloudSpec `json:"vmwareclouddirector,omitempty"`
	Nutanix             *KubeOneNutanixCloudSpec             `json:"nutanix,omitempty"`
}

// KubeOneAWSCloudSpec specifies access data to Amazon Web Services.
type KubeOneAWSCloudSpec struct {
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// KubeOneGCPCloudSpec specifies access data to GCP.
type KubeOneGCPCloudSpec struct {
	ServiceAccount string `json:"serviceAccount"`
}

// KubeOneAzureCloudSpec specifies access credentials to Azure cloud.
type KubeOneAzureCloudSpec struct {
	TenantID       string `json:"tenantID"`
	SubscriptionID string `json:"subscriptionID"`
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
}

// KubeOneDigitalOceanCloudSpec specifies access data to DigitalOcean.
type KubeOneDigitalOceanCloudSpec struct {
	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`
}

// KubeOneOpenStackCloudSpec specifies access data to an OpenStack cloud.
type KubeOneOpenStackCloudSpec struct {
	AuthURL  string `json:"authURL"`
	Username string `json:"username"`
	Password string `json:"password"`

	// Project, formally known as tenant.
	Project string `json:"project"`
	// ProjectID, formally known as tenantID.
	ProjectID string `json:"projectID"`

	Domain string `json:"domain"`
	Region string `json:"region"`
}

// KubeOneVSphereCloudSpec credentials represents a credential for accessing vSphere.
type KubeOneVSphereCloudSpec struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// KubeOneVMwareCloudDirectorCloudSpec represents credentials for accessing VMWare Cloud Director.
type KubeOneVMwareCloudDirectorCloudSpec struct {
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Organization string `json:"organization"`
	VDC          string `json:"vdc"`
}

// KubeOneHetznerCloudSpec specifies access data to hetzner cloud.
type KubeOneHetznerCloudSpec struct {
	// Token is used to authenticate with the Hetzner cloud API.
	Token string `json:"token"`
}

// KubeOneNutanixCloudSpec specifies the access data to Nutanix.
type KubeOneNutanixCloudSpec struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Endpoint is the Nutanix API (Prism Central) endpoint
	Endpoint string `json:"endpoint"`
	// Port is the Nutanix API (Prism Central) port
	Port string `json:"port"`

	// PrismElementUsername to be used for the CSI driver
	PrismElementUsername string `json:"elementUsername"`
	// PrismElementPassword to be used for the CSI driver
	PrismElementPassword string `json:"elementPassword"`
	// PrismElementEndpoint to access Nutanix Prism Element for the CSI driver
	PrismElementEndpoint string `json:"elementEndpoint"`

	// ClusterName is the Nutanix cluster that this user cluster will be deployed to.
	// +optional
	ClusterName   string `json:"clusterName,omitempty"`
	AllowInsecure bool   `json:"allowInsecure,omitempty"`
	ProxyURL      string `json:"proxyURL,omitempty"`
}

type GKECloudSpec struct {
	Name           string `json:"name"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	Zone           string `json:"zone"`
}

type EKSCloudSpec struct {
	Name                 string `json:"name"`
	AccessKeyID          string `json:"accessKeyID,omitempty" required:"true"`
	SecretAccessKey      string `json:"secretAccessKey,omitempty" required:"true"`
	Region               string `json:"region" required:"true"`
	AssumeRoleARN        string `json:"assumeRoleARN,omitempty"` //nolint:tagliatelle
	AssumeRoleExternalID string `json:"assumeRoleExternalID,omitempty"`
}

type EKSClusterSpec struct {
	// The VPC configuration used by the cluster control plane. Amazon EKS VPC resources
	// have specific requirements to work properly with Kubernetes. For more information,
	// see Cluster VPC Considerations (https://docs.aws.amazon.com/eks/latest/userguide/network_reqs.html)
	// and Cluster Security Group Considerations (https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html)
	// in the Amazon EKS User Guide. You must specify at least two subnets. You
	// can specify up to five security groups, but we recommend that you use a dedicated
	// security group for your cluster control plane.
	//
	// ResourcesVpcConfig is a required field

	ResourcesVpcConfig VpcConfigRequest `json:"vpcConfigRequest" required:"true"`

	// The Kubernetes network configuration for the cluster.
	KubernetesNetworkConfig *EKSKubernetesNetworkConfigResponse `json:"kubernetesNetworkConfig,omitempty"`

	// The desired Kubernetes version for your cluster. If you don't specify a value
	// here, the latest version available in Amazon EKS is used.
	Version string `json:"version,omitempty"`

	// The Unix epoch timestamp in seconds for when the cluster was created.
	CreatedAt *time.Time `json:"createdAt,omitempty"`

	// The metadata that you apply to the cluster to assist with categorization
	// and organization. Each tag consists of a key and an optional value. You define
	// both. Cluster tags do not propagate to any other resources associated with
	// the cluster.
	Tags map[string]*string `json:"tags,omitempty"`

	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to AWS API operations on your
	// behalf. For more information, see Amazon EKS Service IAM Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon EKS User Guide .
	//
	// RoleArn is a required field
	RoleArn string `json:"roleArn,omitempty" required:"true"`
}

// The Kubernetes network configuration for the cluster. The response contains
// a value for serviceIpv6Cidr or serviceIpv4Cidr, but not both.
type EKSKubernetesNetworkConfigResponse struct {
	// The IP family used to assign Kubernetes pod and service IP addresses. The
	// IP family is always ipv4, unless you have a 1.21 or later cluster running
	// version 1.10.1 or later of the Amazon VPC CNI add-on and specified ipv6 when
	// you created the cluster.
	IpFamily string `json:"ipFamily,omitempty"` //nolint:staticcheck

	// The CIDR block that Kubernetes pod and service IP addresses are assigned
	// from. Kubernetes assigns addresses from an IPv4 CIDR block assigned to a
	// subnet that the node is in. If you didn't specify a CIDR block when you created
	// the cluster, then Kubernetes assigns addresses from either the 10.100.0.0/16
	// or 172.20.0.0/16 CIDR blocks. If this was specified, then it was specified
	// when the cluster was created and it can't be changed.
	ServiceIpv4Cidr *string `json:"serviceIpv4Cidr,omitempty"`

	// The CIDR block that Kubernetes pod and service IP addresses are assigned
	// from if you created a 1.21 or later cluster with version 1.10.1 or later
	// of the Amazon VPC CNI add-on and specified ipv6 for ipFamily when you created
	// the cluster. Kubernetes assigns service addresses from the unique local address
	// range (fc00::/7) because you can't specify a custom IPv6 CIDR block when
	// you create the cluster.
	ServiceIpv6Cidr *string `json:"serviceIpv6Cidr,omitempty"`
}

type AKSCloudSpec struct {
	Name           string `json:"name"`
	TenantID       string `json:"tenantID,omitempty" required:"true"`
	SubscriptionID string `json:"subscriptionID,omitempty" required:"true"`
	ClientID       string `json:"clientID,omitempty" required:"true"`
	ClientSecret   string `json:"clientSecret,omitempty" required:"true"`
	ResourceGroup  string `json:"resourceGroup" required:"true"`
	Location       string `json:"location"`
}

type (
	AKSProvisioningState string
	AKSPowerState        string
)

type AKSClusterStatus struct {
	// ProvisioningState - Defines values for AKS cluster provisioning state.
	ProvisioningState AKSProvisioningState `json:"provisioningState"`
	// PowerState - Defines values for AKS cluster power state.
	PowerState AKSPowerState `json:"powerState"`
}

type VpcConfigRequest struct {
	// The VPC associated with your cluster.
	VpcId *string `json:"vpcId,omitempty"` //nolint:tagliatelle,staticcheck

	// Specify one or more security groups for the cross-account elastic network
	// interfaces that Amazon EKS creates to use to allow communication between
	// your nodes and the Kubernetes control plane.
	// For more information, see Amazon EKS security group considerations (https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html)
	// in the Amazon EKS User Guide .
	SecurityGroupIds []string `json:"securityGroupIds" required:"true"`

	// Specify subnets for your Amazon EKS nodes. Amazon EKS creates cross-account
	// elastic network interfaces in these subnets to allow communication between
	// your nodes and the Kubernetes control plane.
	SubnetIds []string `json:"subnetIds" required:"true"`
}

// GKEImage represents an object of GKE image.
// swagger:model GKEImage
type GKEImage struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"default,omitempty"`
}

// GKEImageList represents an array of GKE images.
// swagger:model GKEImageList
type GKEImageList []GKEImage

// GKEDiskTypeList represents an array of GKE disk types.
// swagger:model GKEDiskTypeList
type GKEDiskTypeList []GKEDiskType

// GKEDiskType represents a object of GKE disk type.
// swagger:model GKEDiskType
type GKEDiskType struct {
	// Name of the resource.
	Name string `json:"name"`
	// Description: An optional description of this resource.
	Description string `json:"description,omitempty"`
	// DefaultDiskSizeGb: Server-defined default disk size in GB.
	DefaultDiskSizeGb int64 `json:"defaultDiskSizeGb,omitempty"`
	// Kind: Type of the resource. Always compute#diskType for
	// disk types.
	Kind string `json:"kind,omitempty"`
}

// GKEZone represents a object of GKE zone.
// swagger:model GKEZone
type GKEZone struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"default,omitempty"`
}

// GKEZoneList represents an array of GKE zones.
// swagger:model GKEZoneList
type GKEZoneList []GKEZone

// AKSLocationList represents a list of AKS Locations.
// swagger:model AKSLocationList
type AKSLocationList []AKSLocation

// AKSLocation represents an object of Azure Location.
// swagger:model AKSLocation
type AKSLocation struct {
	// The location name.
	Name string `json:"name,omitempty"`
	// READ-ONLY; The category of the region.
	RegionCategory string `json:"regionCategory,omitempty"`
}

// AzureResourceGroup represents an object of Azure ResourceGroup information.
type AzureResourceGroup struct {
	// The name of the resource group.
	Name string `json:"name,omitempty"`
}

// AzureResourceGroupList represents an list of AKS ResourceGroups.
// swagger:model AzureResourceGroupList
type AzureResourceGroupList []AzureResourceGroup

type EKSMachineDeploymentCloudSpec struct {
	// The Unix epoch timestamp in seconds for when the managed node group was created.
	CreatedAt time.Time `json:"createdAt,omitempty"`

	// The subnets to use for the Auto Scaling group that is created for your node
	// group. These subnets must have the tag key kubernetes.io/cluster/CLUSTER_NAME
	// with a value of shared, where CLUSTER_NAME is replaced with the name of your
	// cluster. If you specify launchTemplate, then don't specify SubnetId (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateNetworkInterface.html)
	// in your launch template, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	//
	// Subnets is a required field
	Subnets []string `json:"subnets" required:"true"`

	// The Amazon Resource Name (ARN) of the IAM role to associate with your node
	// group. The Amazon EKS worker node kubelet daemon makes calls to AWS APIs
	// on your behalf. Nodes receive permissions for these API calls through an
	// IAM instance profile and associated policies. Before you can launch nodes
	// and register them into a cluster, you must create an IAM role for those nodes
	// to use when they are launched. For more information, see Amazon EKS node
	// IAM role (https://docs.aws.amazon.com/eks/latest/userguide/worker_node_IAM_role.html)
	// in the Amazon EKS User Guide . If you specify launchTemplate, then don't
	// specify IamInstanceProfile (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_IamInstanceProfile.html)
	// in your launch template, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	//
	// NodeRole is a required field
	NodeRole string `json:"nodeRole" required:"true"`

	// The AMI type for your node group. GPU instance types should use the AL2_x86_64_GPU
	// AMI type. Non-GPU instances should use the AL2_x86_64 AMI type. Arm instances
	// should use the AL2_ARM_64 AMI type. All types use the Amazon EKS optimized
	// Amazon Linux 2 AMI. If you specify launchTemplate, and your launch template
	// uses a custom AMI, then don't specify amiType, or the node group deployment
	// will fail. For more information about using launch templates with Amazon
	// EKS, see Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	AmiType string `json:"amiType,omitempty"`

	// The architecture of the machine image.
	Architecture string `json:"architecture,omitempty"`

	// The capacity type for your node group. Possible values ON_DEMAND | SPOT
	CapacityType string `json:"capacityType,omitempty"`

	// The root device disk size (in GiB) for your node group instances. The default
	// disk size is 20 GiB. If you specify launchTemplate, then don't specify diskSize,
	// or the node group deployment will fail. For more information about using
	// launch templates with Amazon EKS, see Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	DiskSize int32 `json:"diskSize,omitempty"`

	// Specify the instance types for a node group. If you specify a GPU instance
	// type, be sure to specify AL2_x86_64_GPU with the amiType parameter. If you
	// specify launchTemplate, then you can specify zero or one instance type in
	// your launch template or you can specify 0-20 instance types for instanceTypes.
	// If however, you specify an instance type in your launch template and specify
	// any instanceTypes, the node group deployment will fail. If you don't specify
	// an instance type in a launch template or for instanceTypes, then t3.medium
	// is used, by default. If you specify Spot for capacityType, then we recommend
	// specifying multiple values for instanceTypes. For more information, see Managed
	// node group capacity types (https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html#managed-node-group-capacity-types)
	// and Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	InstanceTypes []string `json:"instanceTypes,omitempty"`

	// The Kubernetes labels to be applied to the nodes in the node group when they
	// are created.
	Labels map[string]string `json:"labels,omitempty"`

	// The metadata applied to the node group to assist with categorization and
	// organization. Each tag consists of a key and an optional value. You define
	// both. Node group tags do not propagate to any other resources associated
	// with the node group, such as the Amazon EC2 instances or subnets.
	Tags map[string]string `json:"tags,omitempty"`

	// The scaling configuration details for the Auto Scaling group that is created
	// for your node group.
	ScalingConfig EKSNodegroupScalingConfig `json:"scalingConfig,omitempty"`

	// The Kubernetes version to use for your managed nodes. By default, the Kubernetes
	// version of the cluster is used, and this is the only accepted specified value.
	// If you specify launchTemplate, and your launch template uses a custom AMI,
	// then don't specify version, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	Version string `json:"version,omitempty"`
}

type EKSNodegroupScalingConfig struct {
	// The current number of nodes that the managed node group should maintain.
	DesiredSize int32 `json:"desiredSize,omitempty"`

	// The maximum number of nodes that the managed node group can scale out to.
	// For information about the maximum number that you can specify, see Amazon
	// EKS service quotas (https://docs.aws.amazon.com/eks/latest/userguide/service-quotas.html)
	// in the Amazon EKS User Guide.
	MaxSize int32 `json:"maxSize,omitempty"`

	// The minimum number of nodes that the managed node group can scale in to.
	// This number must be greater than zero.
	MinSize int32 `json:"minSize,omitempty"`
}

// StorageClassList represents a list of Kubernetes StorageClass.
// swagger:model StorageClassList
type StorageClassList []StorageClass

// StorageClass represents a Kubernetes StorageClass
// swagger:model StorageClass
type StorageClass struct {
	Name string `json:"name"`
}
