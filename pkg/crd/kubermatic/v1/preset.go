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
	"fmt"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProviderType string

const (
	ProviderDigitalocean ProviderType = "digitalocean"
	ProviderHetzner      ProviderType = "hetzner"
	ProviderAzure        ProviderType = "azure"
	ProviderVSphere      ProviderType = "vsphere"
	ProviderAWS          ProviderType = "aws"
	ProviderOpenstack    ProviderType = "openstack"
	ProviderPacket       ProviderType = "packet"
	ProviderGCP          ProviderType = "gcp"
	ProviderKubevirt     ProviderType = "kubevirt"
	ProviderAlibaba      ProviderType = "alibaba"
	ProviderAnexia       ProviderType = "anexia"
	ProviderFake         ProviderType = "fake"
)

func SupportedProviders() []ProviderType {
	return []ProviderType{
		ProviderDigitalocean,
		ProviderHetzner,
		ProviderAzure,
		ProviderVSphere,
		ProviderAWS,
		ProviderOpenstack,
		ProviderPacket,
		ProviderGCP,
		ProviderKubevirt,
		ProviderAlibaba,
		ProviderAnexia,
		ProviderFake,
	}
}

func IsProviderSupported(name string) bool {
	for _, provider := range SupportedProviders() {
		if strings.EqualFold(name, string(provider)) {
			return true
		}
	}

	return false
}

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
	Anexia       *Anexia       `json:"anexia,omitempty"`

	Fake *Fake `json:"fake,omitempty"`
	// see RequiredEmails
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`
	// RequiredEmails: specify emails and domains
	// RequiredEmailDomain is appended to RequiredEmails for backward compatibility.
	// e.g.:
	//   RequiredEmailDomain: "example.com"
	//   RequiredEmails: ["foo.com", "foo.bar@test.com"]
	// Result:
	//   *@example.com, *@foo.com and foo.bar@test.com can use the Preset
	RequiredEmails []string `json:"requiredEmails,omitempty"`
	Enabled        *bool    `json:"enabled,omitempty"`
}

func (s *PresetSpec) getProviderValue(providerType ProviderType) reflect.Value {
	spec := reflect.ValueOf(s).Elem()
	if spec.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	ignoreCaseCompare := func(name string) bool {
		return strings.EqualFold(name, string(providerType))
	}

	provider := reflect.Indirect(spec).FieldByNameFunc(ignoreCaseCompare)
	return provider
}

func (s *PresetSpec) SetPresetStatus(enabled bool) {
	s.Enabled = &enabled
}

func (s *PresetSpec) SetPresetProviderStatus(providerType ProviderType, enabled bool) {
	provider := s.getProviderValue(providerType)

	ignoreCaseCompare := func(name string) bool {
		return strings.EqualFold(name, "Enabled")
	}

	enabledField := reflect.Indirect(provider).FieldByNameFunc(ignoreCaseCompare)
	enabledField.Set(reflect.ValueOf(&enabled))
}

func (s PresetSpec) HasProvider(providerType ProviderType) (bool, reflect.Value) {
	provider := s.getProviderValue(providerType)
	return !provider.IsZero(), provider
}

func (s PresetSpec) GetPresetProvider(providerType ProviderType) *PresetProvider {
	hasProvider, providerField := s.HasProvider(providerType)
	if !hasProvider {
		return nil
	}

	presetSpecBaseFieldName := "PresetProvider"
	presetBaseField := reflect.Indirect(providerField).FieldByName(presetSpecBaseFieldName)
	presetBase := presetBaseField.Interface().(PresetProvider)
	return &presetBase
}

func (s PresetSpec) Validate(providerType ProviderType) error {
	hasProvider, providerField := s.HasProvider(providerType)
	if !hasProvider {
		return fmt.Errorf("missing provider configuration for: %s", providerType)
	}

	validateableType := reflect.TypeOf(new(Validateable)).Elem()
	if !providerField.Type().Implements(validateableType) {
		return fmt.Errorf("provider %s does not implement Validateable interface", providerField.Type().Name())
	}

	validateable := providerField.Interface().(Validateable)
	if !validateable.IsValid() {
		return fmt.Errorf("required fields missing for provider spec: %s", providerType)
	}

	return nil
}

func (s PresetSpec) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}

	return *s.Enabled
}

func (s PresetSpec) IsProviderEnabled(provider ProviderType) bool {
	presetProvider := s.GetPresetProvider(provider)
	return presetProvider != nil && presetProvider.IsEnabled()
}

func (s *PresetSpec) OverrideProvider(providerType ProviderType, spec *PresetSpec) {
	dest := s.getProviderValue(providerType)
	src := spec.getProviderValue(providerType)
	dest.Set(src)
}

type Validateable interface {
	IsValid() bool
}

type PresetProvider struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	Datacenter string `json:"datacenter,omitempty"`
}

func (s PresetProvider) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}

	return *s.Enabled
}

type Digitalocean struct {
	PresetProvider `json:",inline"`

	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`
}

func (s Digitalocean) IsValid() bool {
	return len(s.Token) > 0
}

type Hetzner struct {
	PresetProvider `json:",inline"`

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
	PresetProvider `json:",inline"`

	TenantID       string `json:"tenantId"`
	SubscriptionID string `json:"subscriptionId"`
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`

	ResourceGroup     string `json:"resourceGroup,omitempty"`
	VNetResourceGroup string `json:"vnetResourceGroup,omitempty"`
	VNetName          string `json:"vnet,omitempty"`
	SubnetName        string `json:"subnet,omitempty"`
	RouteTableName    string `json:"routeTable,omitempty"`
	SecurityGroup     string `json:"securityGroup,omitempty"`
}

func (s Azure) IsValid() bool {
	return len(s.TenantID) > 0 &&
		len(s.SubscriptionID) > 0 &&
		len(s.ClientID) > 0 &&
		len(s.ClientSecret) > 0
}

type VSphere struct {
	PresetProvider `json:",inline"`

	Username string `json:"username"`
	Password string `json:"password"`

	VMNetName        string `json:"vmNetName,omitempty"`
	Datastore        string `json:"datastore,omitempty"`
	DatastoreCluster string `json:"datastoreCluster,omitempty"`
}

func (s VSphere) IsValid() bool {
	return len(s.Username) > 0 &&
		len(s.Password) > 0
}

type AWS struct {
	PresetProvider `json:",inline"`

	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`

	VPCID               string `json:"vpcId,omitempty"`
	RouteTableID        string `json:"routeTableId,omitempty"`
	InstanceProfileName string `json:"instanceProfileName,omitempty"`
	SecurityGroupID     string `json:"securityGroupID,omitempty"`
	ControlPlaneRoleARN string `json:"roleARN,omitempty"`
}

func (s AWS) IsValid() bool {
	return len(s.AccessKeyID) > 0 &&
		len(s.SecretAccessKey) > 0
}

type Openstack struct {
	PresetProvider `json:",inline"`

	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
	TenantID string `json:"tenantID"`
	Domain   string `json:"domain"`

	Network        string `json:"network,omitempty"`
	SecurityGroups string `json:"securityGroups,omitempty"`
	FloatingIPPool string `json:"floatingIpPool,omitempty"`
	RouterID       string `json:"routerID,omitempty"`
	SubnetID       string `json:"subnetID,omitempty"`
}

func (s Openstack) IsValid() bool {
	return len(s.Username) > 0 &&
		len(s.Password) > 0 &&
		(len(s.Tenant) > 0 || len(s.TenantID) > 0) &&
		len(s.Domain) > 0
}

type Packet struct {
	PresetProvider `json:",inline"`

	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectId"`

	BillingCycle string `json:"billingCycle,omitempty"`
}

func (s Packet) IsValid() bool {
	return len(s.APIKey) > 0 &&
		len(s.ProjectID) > 0
}

type GCP struct {
	PresetProvider `json:",inline"`

	ServiceAccount string `json:"serviceAccount"`

	Network    string `json:"network,omitempty"`
	Subnetwork string `json:"subnetwork,omitempty"`
}

func (s GCP) IsValid() bool {
	return len(s.ServiceAccount) > 0
}

type Fake struct {
	PresetProvider `json:",inline"`

	Token string `json:"token"`
}

func (s Fake) IsValid() bool {
	return len(s.Token) > 0
}

type Kubevirt struct {
	PresetProvider `json:",inline"`

	Kubeconfig string `json:"kubeconfig"`
}

func (s Kubevirt) IsValid() bool {
	return len(s.Kubeconfig) > 0
}

type Alibaba struct {
	PresetProvider `json:",inline"`

	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
}

func (s Alibaba) IsValid() bool {
	return len(s.AccessKeyID) > 0 &&
		len(s.AccessKeySecret) > 0
}

type Anexia struct {
	PresetProvider `json:",inline"`

	// Token is used to authenticate with the Anexia API.
	Token string `json:"token"`
}

func (s Anexia) IsValid() bool {
	return len(s.Token) > 0
}
