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
	Digitalocean *Digitalocean `json:"digitalocean,omitempty"`
	Hetzner      *Hetzner      `json:"hetzner,omitempty"`
	Azure        *Azure        `json:"azure,omitempty"`
	VSphere      *VSphere      `json:"vsphere,omitempty"`
	AWS          *AWS          `json:"aws,omitempty"`
	Openstack    *Openstack    `json:"openstack,omitempty"`
	Packet       *Packet       `json:"packet,omitempty"`
	GCP          *GCP          `json:"gcp,omitempty"`
	Kubevirt     *Kubevirt     `json:"kubevirt,omitempty"`

	Fake                *Fake  `json:"fake,omitempty"`
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`
}

type Digitalocean struct {
	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`
}

type Hetzner struct {
	// Token is used to authenticate with the Hetzner API.
	Token string `json:"token"`
}

type Azure struct {
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

type VSphere struct {
	Username string `json:"username"`
	Password string `json:"password"`

	VMNetName string `json:"vmNetName,omitempty"`
}

type AWS struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`

	VPCID               string `json:"vpcId,omitempty"`
	RouteTableID        string `json:"routeTableId,omitempty"`
	InstanceProfileName string `json:"instanceProfileName,omitempty"`
	SecurityGroupID     string `json:"securityGroupID,omitempty"`
	ControlPlaneRoleARN string `json:"roleARN,omitempty"`
}

type Openstack struct {
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

type Packet struct {
	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectId"`

	BillingCycle string `json:"billingCycle,omitempty"`
}

type GCP struct {
	ServiceAccount string `json:"serviceAccount"`

	Network    string `json:"network,omitempty"`
	Subnetwork string `json:"subnetwork,omitempty"`
}

type Fake struct {
	Token string `json:"token"`
}

type Kubevirt struct {
	Kubeconfig string `json:"kubeconfig"`
}
