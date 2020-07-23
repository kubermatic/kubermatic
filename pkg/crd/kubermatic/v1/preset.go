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
	Alibaba      *Alibaba      `json:"alibaba,omitempty"`

	Fake                *Fake  `json:"fake,omitempty"`
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`
}

type Digitalocean struct {
	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`

	Datacenter string `json:"datacenter,omitempty"`
}

type Hetzner struct {
	// Token is used to authenticate with the Hetzner API.
	Token string `json:"token"`

	Datacenter string `json:"datacenter,omitempty"`
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

	Datacenter string `json:"datacenter,omitempty"`
}

type VSphere struct {
	Username string `json:"username"`
	Password string `json:"password"`

	VMNetName        string `json:"vmNetName,omitempty"`
	Datastore        string `json:"datastore,omitempty"`
	DatastoreCluster string `json:"datastoreCluster,omitempty"`
	Datacenter       string `json:"datacenter,omitempty"`
}

type AWS struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`

	VPCID               string `json:"vpcId,omitempty"`
	RouteTableID        string `json:"routeTableId,omitempty"`
	InstanceProfileName string `json:"instanceProfileName,omitempty"`
	SecurityGroupID     string `json:"securityGroupID,omitempty"`
	ControlPlaneRoleARN string `json:"roleARN,omitempty"`

	Datacenter string `json:"datacenter,omitempty"`
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

	Datacenter string `json:"datacenter,omitempty"`
}

type Packet struct {
	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectId"`

	BillingCycle string `json:"billingCycle,omitempty"`

	Datacenter string `json:"datacenter,omitempty"`
}

type GCP struct {
	ServiceAccount string `json:"serviceAccount"`

	Network    string `json:"network,omitempty"`
	Subnetwork string `json:"subnetwork,omitempty"`

	Datacenter string `json:"datacenter,omitempty"`
}

type Fake struct {
	Token string `json:"token"`

	Datacenter string `json:"datacenter,omitempty"`
}

type Kubevirt struct {
	Kubeconfig string `json:"kubeconfig"`

	Datacenter string `json:"datacenter,omitempty"`
}

type Alibaba struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`

	Datacenter string `json:"datacenter,omitempty"`
}
