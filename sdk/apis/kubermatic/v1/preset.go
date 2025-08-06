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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// PresetList is the type representing a PresetList.
type PresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of presets
	Items []Preset `json:"items"`
}

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// Presets are preconfigured cloud provider credentials that can be applied
// to new clusters. This frees end users from having to know the actual
// credentials used for their clusters.
type Preset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PresetSpec `json:"spec"`
}

// Presets specifies default presets for supported providers.
type PresetSpec struct {
	// Access data for DigitalOcean.
	Digitalocean *Digitalocean `json:"digitalocean,omitempty"`
	// Access data for Hetzner.
	Hetzner *Hetzner `json:"hetzner,omitempty"`
	// Access data for Microsoft Azure Cloud.
	Azure *Azure `json:"azure,omitempty"`
	// Access data for vSphere.
	VSphere *VSphere `json:"vsphere,omitempty"`
	// Access data for Baremetal (Tinkerbell only for now).
	Baremetal *Baremetal `json:"baremetal,omitempty"`
	// Access data for Amazon Web Services(AWS) Cloud.
	AWS *AWS `json:"aws,omitempty"`
	// Access data for OpenStack.
	Openstack *Openstack `json:"openstack,omitempty"`
	// Deprecated: The Packet / Equinix Metal provider is deprecated and will be REMOVED IN VERSION 2.29.
	// This provider is no longer supported. Migrate your configurations away from "packet" immediately.
	// Access data for Packet Cloud.
	// NOOP.
	Packet *Packet `json:"packet,omitempty"`
	// Access data for Google Cloud Platform(GCP).
	GCP *GCP `json:"gcp,omitempty"`
	// Access data for KuberVirt.
	Kubevirt *Kubevirt `json:"kubevirt,omitempty"`
	// Access data for Alibaba Cloud.
	Alibaba *Alibaba `json:"alibaba,omitempty"`
	// Access data for Anexia.
	Anexia *Anexia `json:"anexia,omitempty"`
	// Access data for Nutanix.
	Nutanix *Nutanix `json:"nutanix,omitempty"`
	// Access data for VMware Cloud Director.
	VMwareCloudDirector *VMwareCloudDirector `json:"vmwareclouddirector,omitempty"`
	// Access data for Google Kubernetes Engine(GKE).
	GKE *GKE `json:"gke,omitempty"`
	// Access data for Amazon Elastic Kubernetes Service(EKS).
	EKS *EKS `json:"eks,omitempty"`
	// Access data for Azure Kubernetes Service(AKS).
	AKS *AKS `json:"aks,omitempty"`

	Fake *Fake `json:"fake,omitempty"`

	// RequiredEmails is a list of e-mail addresses that this presets should
	// be restricted to. Each item in the list can be either a full e-mail
	// address or just a domain name. This restriction is only enforced in the
	// KKP API.
	RequiredEmails []string `json:"requiredEmails,omitempty"`

	// Projects is a list of project IDs that this preset is limited to.
	Projects []string `json:"projects,omitempty"`

	// Only enabled presets will be available in the KKP dashboard.
	Enabled *bool `json:"enabled,omitempty"`
}

func (s PresetSpec) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}

	return *s.Enabled
}

func (s *PresetSpec) SetEnabled(enabled bool) {
	s.Enabled = &enabled
}

type ProviderPreset struct {
	// Only enabled presets will be available in the KKP dashboard.
	Enabled *bool `json:"enabled,omitempty"`
	// IsCustomizable marks a preset as editable on the KKP UI; Customizable presets still have the credentials obscured on the UI, but other fields that are not considered private are displayed during cluster creation. Users can then update those fields, if required.
	// NOTE: This is only supported for OpenStack Cloud Provider in KKP 2.26. Support for other providers will be added later on.
	IsCustomizable bool `json:"isCustomizable,omitempty"`
	// If datacenter is set, this preset is only applicable to the
	// configured datacenter.
	Datacenter string `json:"datacenter,omitempty"`
}

func (s ProviderPreset) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}

	return *s.Enabled
}

type Digitalocean struct {
	ProviderPreset `json:",inline"`

	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`
}

func (s Digitalocean) IsValid() bool {
	return len(s.Token) > 0
}

type Hetzner struct {
	ProviderPreset `json:",inline"`

	// Token is used to authenticate with the Hetzner API.
	Token string `json:"token"`

	// Network is the pre-existing Hetzner network in which the machines are running.
	// While machines can be in multiple networks, a single one must be chosen for the
	// HCloud CCM to work.
	// If this is empty, the network configured on the datacenter will be used.
	Network string `json:"network,omitempty"`
}

func (s Hetzner) IsValid() bool {
	return len(s.Token) > 0
}

type Azure struct {
	ProviderPreset `json:",inline"`

	// The Azure Active Directory Tenant used for the user cluster.
	TenantID string `json:"tenantID"`
	// The Azure Subscription used for the user cluster.
	SubscriptionID string `json:"subscriptionID"`
	// The service principal used to access Azure.
	ClientID string `json:"clientID"`
	// The client secret corresponding to the given service principal.
	ClientSecret string `json:"clientSecret"`

	// The resource group that will be used to look up and create resources for the cluster in.
	// If set to empty string at cluster creation, a new resource group will be created and this field will be updated to
	// the generated resource group's name.
	ResourceGroup string `json:"resourceGroup,omitempty"`
	// Optional: Defines a second resource group that will be used for VNet related resources instead.
	// If left empty, NO additional resource group will be created and all VNet related resources use the resource group defined by `resourceGroup`.
	VNetResourceGroup string `json:"vnetResourceGroup,omitempty"`
	// The name of the VNet resource used for setting up networking in.
	// If set to empty string at cluster creation, a new VNet will be created and this field will be updated to
	// the generated VNet's name.
	VNetName string `json:"vnet,omitempty"`
	// The name of a subnet in the VNet referenced by `vnet`.
	// If set to empty string at cluster creation, a new subnet will be created and this field will be updated to
	// the generated subnet's name. If no VNet is defined at cluster creation, this field should be empty as well.
	SubnetName string `json:"subnet,omitempty"`
	// The name of a route table associated with the subnet referenced by `subnet`.
	// If set to empty string at cluster creation, a new route table will be created and this field will be updated to
	// the generated route table's name. If no subnet is defined at cluster creation, this field should be empty as well.
	RouteTableName string `json:"routeTable,omitempty"`
	// The name of a security group associated with the subnet referenced by `subnet`.
	// If set to empty string at cluster creation, a new security group will be created and this field will be updated to
	// the generated security group's name. If no subnet is defined at cluster creation, this field should be empty as well.
	SecurityGroup string `json:"securityGroup,omitempty"`
	// LoadBalancerSKU sets the LB type that will be used for the Azure cluster, possible values are "basic" and "standard", if empty, "standard" will be used
	LoadBalancerSKU LBSKU `json:"loadBalancerSKU"` //nolint:tagliatelle
}

func (s Azure) IsValid() bool {
	return len(s.TenantID) > 0 &&
		len(s.SubscriptionID) > 0 &&
		len(s.ClientID) > 0 &&
		len(s.ClientSecret) > 0
}

type VSphere struct {
	ProviderPreset `json:",inline"`

	// The vSphere user name.
	Username string `json:"username"`
	// The vSphere user password.
	Password string `json:"password"`

	// Deprecated: Use networks instead.
	VMNetName string `json:"vmNetName,omitempty"`
	// List of vSphere networks.
	Networks []string `json:"networks,omitempty"`
	// Datastore to be used for storing virtual machines and as a default for dynamic volume provisioning, it is mutually exclusive with DatastoreCluster.
	Datastore string `json:"datastore,omitempty"`
	// DatastoreCluster to be used for storing virtual machines, it is mutually exclusive with Datastore.
	DatastoreCluster string `json:"datastoreCluster,omitempty"`
	// ResourcePool is used to manage resources such as cpu and memory for vSphere virtual machines. The resource pool should be defined on vSphere cluster level.
	ResourcePool string `json:"resourcePool,omitempty"`
	// BasePath configures a vCenter folder path that KKP will create an individual cluster folder in.
	// If it's an absolute path, the RootPath configured in the datacenter will be ignored. If it is a relative path,
	// the BasePath part will be appended to the RootPath to construct the full path. For both cases,
	// the full folder structure needs to exist. KKP will only try to create the cluster folder.
	BasePath string `json:"basePath,omitempty"`
}

func (s VSphere) IsValid() bool {
	return len(s.Username) > 0 && len(s.Password) > 0
}

type VMwareCloudDirector struct {
	ProviderPreset `json:",inline"`

	// The VMware Cloud Director user name.
	Username string `json:"username,omitempty"`
	// The VMware Cloud Director user password.
	Password string `json:"password,omitempty"`
	// The VMware Cloud Director API token.
	APIToken string `json:"apiToken,omitempty"`
	// The organizational virtual data center.
	VDC string `json:"vdc"`
	// The name of organization to use.
	Organization string `json:"organization"`
	// The name of organizational virtual data center network that will be associated with the VMs and vApp.
	// Deprecated: OVDCNetwork has been deprecated starting with KKP 2.25 and will be removed in KKP 2.27+. It is recommended to use OVDCNetworks instead.
	OVDCNetwork string `json:"ovdcNetwork,omitempty"`
	// OVDCNetworks is the list of organizational virtual data center networks that will be attached to the vApp and can be consumed the VMs.
	OVDCNetworks []string `json:"ovdcNetworks,omitempty"`
}

func (s VMwareCloudDirector) IsValid() bool {
	hasAnyNet := len(s.OVDCNetwork) > 0 || len(s.OVDCNetworks) > 0
	hasBothNets := len(s.OVDCNetwork) > 0 && len(s.OVDCNetworks) > 0
	hasCredentials := len(s.Username) > 0 && len(s.Password) > 0

	return true &&
		(hasCredentials || len(s.APIToken) > 0) &&
		len(s.VDC) > 0 &&
		len(s.Organization) > 0 &&
		hasAnyNet && !hasBothNets
}

type Baremetal struct {
	ProviderPreset `json:",inline"`

	Tinkerbell *Tinkerbell `json:"tinkerbell,omitempty"`
}

type Tinkerbell struct {
	// Kubeconfig is the cluster's kubeconfig file, encoded with base64.
	Kubeconfig string `json:"kubeconfig"`
}

func (s Tinkerbell) IsValid() bool {
	return len(s.Kubeconfig) > 0
}

func (s Baremetal) IsValid() bool {
	if s.Tinkerbell != nil {
		return s.Tinkerbell.IsValid()
	}
	return false
}

type AWS struct {
	ProviderPreset `json:",inline"`

	// The Access key ID used to authenticate against AWS.
	AccessKeyID string `json:"accessKeyID"`
	// The Secret Access Key used to authenticate against AWS.
	SecretAccessKey string `json:"secretAccessKey"`

	// Defines the ARN for an IAM role that should be assumed when handling resources on AWS. It will be used
	// to acquire temporary security credentials using an STS AssumeRole API operation whenever creating an AWS session.
	// +optional
	AssumeRoleARN string `json:"assumeRoleARN,omitempty"` //nolint:tagliatelle
	// An arbitrary string that may be needed when calling the STS AssumeRole API operation.
	// Using an external ID can help to prevent the "confused deputy problem".
	// +optional
	AssumeRoleExternalID string `json:"assumeRoleExternalID,omitempty"`

	// AWS VPC to use. Must be configured.
	VPCID string `json:"vpcID,omitempty"`
	// Route table to use. This can be configured, but if left empty will be
	// automatically filled in during reconciliation.
	RouteTableID string `json:"routeTableID,omitempty"`
	// Instance profile to use. This can be configured, but if left empty will be
	// automatically filled in during reconciliation.
	InstanceProfileName string `json:"instanceProfileName,omitempty"`
	// Security group to use. This can be configured, but if left empty will be
	// automatically filled in during reconciliation.
	SecurityGroupID string `json:"securityGroupID,omitempty"`
	// ARN to use. This can be configured, but if left empty will be
	// automatically filled in during reconciliation.
	ControlPlaneRoleARN string `json:"roleARN,omitempty"` //nolint:tagliatelle
}

func (s AWS) IsValid() bool {
	return len(s.AccessKeyID) > 0 && len(s.SecretAccessKey) > 0
}

type Openstack struct {
	ProviderPreset `json:",inline"`

	UseToken bool `json:"useToken,omitempty"`

	// Application credential ID to authenticate in combination with an application credential secret (which is not the user's password).
	ApplicationCredentialID string `json:"applicationCredentialID,omitempty"`
	// Application credential secret (which is not the user's password) to authenticate in combination with an application credential ID.
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Project, formally known as tenant.
	Project string `json:"project,omitempty"`
	// ProjectID, formally known as tenantID.
	ProjectID string `json:"projectID,omitempty"`
	// Domain holds the name of the identity service (keystone) domain.
	Domain string `json:"domain,omitempty"`

	// Network holds the name of the internal network When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.
	Network        string `json:"network,omitempty"`
	SecurityGroups string `json:"securityGroups,omitempty"`
	// FloatingIPPool holds the name of the public network The public network is reachable from the outside world and should provide the pool of IP addresses to choose from.
	FloatingIPPool string `json:"floatingIPPool,omitempty"`
	RouterID       string `json:"routerID,omitempty"`
	SubnetID       string `json:"subnetID,omitempty"`
}

func (s Openstack) IsValid() bool {
	if s.UseToken {
		return true
	}

	if len(s.ApplicationCredentialID) > 0 {
		return len(s.ApplicationCredentialSecret) > 0
	}

	return len(s.Username) > 0 &&
		len(s.Password) > 0 &&
		(len(s.Project) > 0 || len(s.ProjectID) > 0) &&
		len(s.Domain) > 0
}

// Deprecated: The Packet / Equinix Metal provider is deprecated and will be REMOVED IN VERSION 2.29.
// This provider is no longer supported. Migrate your configurations away from "packet" immediately.
// NOOP.
type Packet struct {
	ProviderPreset `json:",inline"`

	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectID"`

	BillingCycle string `json:"billingCycle,omitempty"`
}

func (s Packet) IsValid() bool {
	return len(s.APIKey) > 0 && len(s.ProjectID) > 0
}

type GCP struct {
	ProviderPreset `json:",inline"`

	// ServiceAccount is the Google Service Account (JSON format), encoded with base64.
	ServiceAccount string `json:"serviceAccount"`

	Network    string `json:"network,omitempty"`
	Subnetwork string `json:"subnetwork,omitempty"`
}

func (s GCP) IsValid() bool {
	return len(s.ServiceAccount) > 0
}

type Fake struct {
	ProviderPreset `json:",inline"`

	Token string `json:"token"`
}

func (s Fake) IsValid() bool {
	return len(s.Token) > 0
}

type Kubevirt struct {
	ProviderPreset `json:",inline"`

	// Kubeconfig is the cluster's kubeconfig file, encoded with base64.
	Kubeconfig string `json:"kubeconfig"`

	// VPCName  is a virtual network name dedicated to a single tenant within a KubeVirt
	VPCName string `json:"vpcName,omitempty"`

	// SubnetName is the name of a subnet that is smaller, segmented portion of a larger network, like a Virtual Private Cloud (VPC).
	SubnetName string `json:"subnetName,omitempty"`
}

func (s Kubevirt) IsValid() bool {
	return len(s.Kubeconfig) > 0
}

type Alibaba struct {
	ProviderPreset `json:",inline"`

	// The Access Key ID used to authenticate against Alibaba.
	AccessKeyID string `json:"accessKeyID"`
	// The Access Key Secret used to authenticate against Alibaba.
	AccessKeySecret string `json:"accessKeySecret"`
}

func (s Alibaba) IsValid() bool {
	return len(s.AccessKeyID) > 0 &&
		len(s.AccessKeySecret) > 0
}

type Anexia struct {
	ProviderPreset `json:",inline"`

	// Token is used to authenticate with the Anexia API.
	Token string `json:"token"`
}

func (s Anexia) IsValid() bool {
	return len(s.Token) > 0
}

type Nutanix struct {
	ProviderPreset `json:",inline"`

	// Optional: To configure a HTTP proxy to access Nutanix Prism Central.
	ProxyURL string `json:"proxyURL,omitempty"`
	// Username that is used to access the Nutanix Prism Central API.
	Username string `json:"username"`
	// Password corresponding to the provided user.
	Password string `json:"password"`

	// The name of the Nutanix cluster to which the resources and nodes are deployed to.
	ClusterName string `json:"clusterName"`
	// Optional: Nutanix project to use. If none is given,
	// no project will be used.
	ProjectName string `json:"projectName,omitempty"`

	// Prism Element Username for CSI driver.
	CSIUsername string `json:"csiUsername,omitempty"`

	// Prism Element Password for CSI driver.
	CSIPassword string `json:"csiPassword,omitempty"`

	// CSIEndpoint to access Nutanix Prism Element for CSI driver.
	CSIEndpoint string `json:"csiEndpoint,omitempty"`

	// CSIPort to use when connecting to the Nutanix Prism Element endpoint (defaults to 9440).
	CSIPort *int32 `json:"csiPort,omitempty"`
}

func (s Nutanix) IsValid() bool {
	return len(s.Username) > 0 && len(s.Password) > 0
}

type GKE struct {
	ProviderPreset `json:",inline"`

	ServiceAccount string `json:"serviceAccount"`
}

func (s GKE) IsValid() bool {
	return len(s.ServiceAccount) > 0
}

type EKS struct {
	ProviderPreset `json:",inline"`

	// The Access key ID used to authenticate against AWS.
	AccessKeyID string `json:"accessKeyID"`
	// The Secret Access Key used to authenticate against AWS.
	SecretAccessKey string `json:"secretAccessKey"`
	// Defines the ARN for an IAM role that should be assumed when handling resources on AWS. It will be used
	// to acquire temporary security credentials using an STS AssumeRole API operation whenever creating an AWS session.
	// required: false
	AssumeRoleARN string `json:"assumeRoleARN,omitempty"` //nolint:tagliatelle
	// An arbitrary string that may be needed when calling the STS AssumeRole API operation.
	// Using an external ID can help to prevent the "confused deputy problem".
	// required: false
	AssumeRoleExternalID string `json:"assumeRoleExternalID,omitempty"`
}

func (s EKS) IsValid() bool {
	return len(s.AccessKeyID) > 0 &&
		len(s.SecretAccessKey) > 0
}

type AKS struct {
	ProviderPreset `json:",inline"`

	// The Azure Active Directory Tenant used for the user cluster.
	TenantID string `json:"tenantID"`
	// The Azure Subscription used for the user cluster.
	SubscriptionID string `json:"subscriptionID"`
	// The service principal used to access Azure.
	ClientID string `json:"clientID"`
	// The client secret corresponding to the given service principal.
	ClientSecret string `json:"clientSecret"`
}

func (s AKS) IsValid() bool {
	return len(s.TenantID) > 0 &&
		len(s.SubscriptionID) > 0 &&
		len(s.ClientID) > 0 &&
		len(s.ClientSecret) > 0
}
