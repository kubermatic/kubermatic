package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PresetList is the type representing a PresetList
type PresetList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of presets
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md
	Items []Preset `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Preset is the type representing a Preset
type Preset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PresetSpec `json:"spec"`
}

// Presets specifies default presets for supported providers
type PresetSpec struct {
	Digitalocean Digitalocean `json:"digitalocean,omitempty"`
	Hetzner      Hetzner      `json:"hetzner,omitempty"`
	Azure        Azure        `json:"azure,omitempty"`
	VSphere      VSphere      `json:"vsphere,omitempty"`
	AWS          AWS          `json:"aws,omitempty"`
	Openstack    Openstack    `json:"openstack,omitempty"`
	Packet       Packet       `json:"packet,omitempty"`
	GCP          GCP          `json:"gcp,omitempty"`
	Kubevirt     Kubevirt     `json:"kubevirt,omitempty"`

	Fake                Fake   `json:"fake,omitempty"`
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`
}

type Digitalocean struct {
	Credentials []DigitaloceanPresetCredentials `json:"credentials,omitempty"`
}

type Hetzner struct {
	Credentials []HetznerPresetCredentials `json:"credentials,omitempty"`
}

type Azure struct {
	Credentials []AzurePresetCredentials `json:"credentials,omitempty"`
}

type VSphere struct {
	Credentials []VSpherePresetCredentials `json:"credentials,omitempty"`
}

type AWS struct {
	Credentials []AWSPresetCredentials `json:"credentials,omitempty"`
}

type Openstack struct {
	Credentials []OpenstackPresetCredentials `json:"credentials,omitempty"`
}

type Packet struct {
	Credentials []PacketPresetCredentials `json:"credentials,omitempty"`
}

type GCP struct {
	Credentials []GCPPresetCredentials `json:"credentials,omitempty"`
}

type Fake struct {
	Credentials []FakePresetCredentials `json:"credentials,omitempty"`
}

type Kubevirt struct {
	Credentials []KubevirtPresetCredentials `json:"credentials,omitempty"`
}

// DigitaloceanPresetCredentials defines Digitalocean credential
type DigitaloceanPresetCredentials struct {
	Name  string `json:"name"`
	Token string `json:"token"` // Token is used to authenticate with the DigitalOcean API.
}

type HetznerPresetCredentials struct {
	Name  string `json:"name"`
	Token string `json:"token"` // Token is used to authenticate with the Hetzner API.
}

type AzurePresetCredentials struct {
	Name           string `json:"name"`
	TenantID       string `json:"tenantId"`
	SubscriptionID string `json:"subscriptionId"`
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`

	ResourceGroup  string `json:"resourceGroup,omitempty"`
	VNetName       string `json:"vnet,omitempty"`
	SubnetName     string `json:"subnet,omitempty"`
	RouteTableName string `json:"routeTable,omitempty"`
	SecurityGroup  string `json:"securityGroup,omitempty"`
}

// VSpherePresetCredentials credentials represents a credential for accessing vSphere
type VSpherePresetCredentials struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`

	VMNetName string `json:"vmNetName,omitempty"`
}

type AWSPresetCredentials struct {
	Name            string `json:"name"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`

	VPCID               string `json:"vpcId,omitempty"`
	RouteTableID        string `json:"routeTableId,omitempty"`
	InstanceProfileName string `json:"instanceProfileName,omitempty"`
	SecurityGroupID     string `json:"securityGroupID,omitempty"`
}

// OpenstackPresetCredentials specifies access data to an openstack cloud.
type OpenstackPresetCredentials struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
	TenantID string `json:"tenantID"`
	Domain   string `json:"domain"`

	Network        string `json:"network,,omitempty"`
	SecurityGroups string `json:"securityGroups,omitempty"`
	FloatingIPPool string `json:"floatingIpPool,omitempty"`
	RouterID       string `json:"routerID,omitempty"`
	SubnetID       string `json:"subnetID,omitempty"`
}

// PacketPresetCredentials specifies access data to a Packet cloud.
type PacketPresetCredentials struct {
	Name      string `json:"name"`
	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectId"`

	BillingCycle string `json:"billingCycle,omitempty"`
}

// GCPPresetCredentials specifies access data to GCP.
type GCPPresetCredentials struct {
	Name           string `json:"name"`
	ServiceAccount string `json:"serviceAccount"`

	Network    string `json:"network,omitempty"`
	Subnetwork string `json:"subnetwork,omitempty"`
}

// KubevirtPresetCredentials specifies access data to Kubevirt.
type KubevirtPresetCredentials struct {
	Name       string `json:"name,omitempty"`
	Kubeconfig string `json:"kubeconfig,omitempty"`
}

// FakePresetCredentials defines fake credential for tests
type FakePresetCredentials struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}
