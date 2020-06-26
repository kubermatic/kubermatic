/*
Copyright 2019 The Machine Controller Authors.

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

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OperatingSystem defines the host operating system.
type OperatingSystem string

const (
	OperatingSystemCoreos  OperatingSystem = "coreos"
	OperatingSystemUbuntu  OperatingSystem = "ubuntu"
	OperatingSystemCentOS  OperatingSystem = "centos"
	OperatingSystemSLES    OperatingSystem = "sles"
	OperatingSystemRHEL    OperatingSystem = "rhel"
	OperatingSystemFlatcar OperatingSystem = "flatcar"
)

type CloudProvider string

const (
	CloudProviderAWS          CloudProvider = "aws"
	CloudProviderAzure        CloudProvider = "azure"
	CloudProviderDigitalocean CloudProvider = "digitalocean"
	CloudProviderGoogle       CloudProvider = "gce"
	CloudProviderHetzner      CloudProvider = "hetzner"
	CloudProviderKubeVirt     CloudProvider = "kubevirt"
	CloudProviderLinode       CloudProvider = "linode"
	CloudProviderOpenstack    CloudProvider = "openstack"
	CloudProviderPacket       CloudProvider = "packet"
	CloudProviderVsphere      CloudProvider = "vsphere"
	CloudProviderFake         CloudProvider = "fake"
	CloudProviderAlibaba      CloudProvider = "alibaba"
)

var (
	ErrOSNotSupported = errors.New("os not supported")

	// AllOperatingSystems is a slice containing all supported operating system identifiers.
	AllOperatingSystems = []OperatingSystem{
		OperatingSystemCoreos,
		OperatingSystemUbuntu,
		OperatingSystemCentOS,
		OperatingSystemSLES,
		OperatingSystemRHEL,
		OperatingSystemFlatcar,
	}

	// AllCloudProviders is a slice containing all supported cloud providers.
	AllCloudProviders = []CloudProvider{
		CloudProviderAWS,
		CloudProviderAzure,
		CloudProviderDigitalocean,
		CloudProviderGoogle,
		CloudProviderHetzner,
		CloudProviderKubeVirt,
		CloudProviderLinode,
		CloudProviderOpenstack,
		CloudProviderPacket,
		CloudProviderVsphere,
		CloudProviderFake,
		CloudProviderAlibaba,
	}
)

// DNSConfig contains a machine's DNS configuration
type DNSConfig struct {
	Servers []string `json:"servers"`
}

// NetworkConfig contains a machine's static network configuration
type NetworkConfig struct {
	CIDR    string    `json:"cidr"`
	Gateway string    `json:"gateway"`
	DNS     DNSConfig `json:"dns"`
}

type Config struct {
	SSHPublicKeys []string `json:"sshPublicKeys"`

	CloudProvider     CloudProvider        `json:"cloudProvider"`
	CloudProviderSpec runtime.RawExtension `json:"cloudProviderSpec"`

	OperatingSystem     OperatingSystem      `json:"operatingSystem"`
	OperatingSystemSpec runtime.RawExtension `json:"operatingSystemSpec"`

	// +optional
	Network *NetworkConfig `json:"network,omitempty"`

	// +optional
	OverwriteCloudConfig *string `json:"overwriteCloudConfig,omitempty"`
}

// GlobalObjectKeySelector is needed as we can not use v1.SecretKeySelector
// because it is not cross namespace
type GlobalObjectKeySelector struct {
	corev1.ObjectReference `json:",inline"`
	Key                    string `json:"key,omitempty"`
}

type GlobalSecretKeySelector GlobalObjectKeySelector
type GlobalConfigMapKeySelector GlobalObjectKeySelector

type ConfigVarString struct {
	Value           string                     `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

// This type only exists to have the same fields as ConfigVarString but
// not its funcs, so it can be used as target for json.Unmarshal without
// causing a recursion
type configVarStringWithoutUnmarshaller ConfigVarString

// MarshalJSON converts a configVarString to its JSON form, omitting empty strings.
// This is done to not have the json object cluttered with empty strings
// This will eventually hopefully be resolved within golang itself
// https://github.com/golang/go/issues/11939
func (configVarString ConfigVarString) MarshalJSON() ([]byte, error) {
	var secretKeyRefEmpty, configMapKeyRefEmpty bool
	if configVarString.SecretKeyRef.ObjectReference.Namespace == "" &&
		configVarString.SecretKeyRef.ObjectReference.Name == "" &&
		configVarString.SecretKeyRef.Key == "" {
		secretKeyRefEmpty = true
	}

	if configVarString.ConfigMapKeyRef.ObjectReference.Namespace == "" &&
		configVarString.ConfigMapKeyRef.ObjectReference.Name == "" &&
		configVarString.ConfigMapKeyRef.Key == "" {
		configMapKeyRefEmpty = true
	}

	if secretKeyRefEmpty && configMapKeyRefEmpty {
		return []byte(fmt.Sprintf(`"%s"`, configVarString.Value)), nil
	}

	buffer := bytes.NewBufferString("{")
	if !secretKeyRefEmpty {
		jsonVal, err := json.Marshal(configVarString.SecretKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`"secretKeyRef":%s`, string(jsonVal)))
	}

	if !configMapKeyRefEmpty {
		var leadingComma string
		if !secretKeyRefEmpty {
			leadingComma = ","
		}
		jsonVal, err := json.Marshal(configVarString.ConfigMapKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`%s"configMapKeyRef":%s`, leadingComma, jsonVal))
	}

	if configVarString.Value != "" {
		buffer.WriteString(fmt.Sprintf(`,"value":"%s"`, configVarString.Value))
	}

	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (configVarString *ConfigVarString) UnmarshalJSON(b []byte) error {
	if !bytes.HasPrefix(b, []byte("{")) {
		b = bytes.TrimPrefix(b, []byte(`"`))
		b = bytes.TrimSuffix(b, []byte(`"`))
		configVarString.Value = string(b)
		return nil
	}
	// This type must have the same fields as ConfigVarString but not
	// its UnmarshalJSON, otherwise we cause a recursion
	var cvsDummy configVarStringWithoutUnmarshaller
	err := json.Unmarshal(b, &cvsDummy)
	if err != nil {
		return err
	}
	configVarString.Value = cvsDummy.Value
	configVarString.SecretKeyRef = cvsDummy.SecretKeyRef
	configVarString.ConfigMapKeyRef = cvsDummy.ConfigMapKeyRef
	return nil
}

type ConfigVarBool struct {
	Value           bool                       `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

type configVarBoolWithoutUnmarshaller ConfigVarBool

// MarshalJSON encodes the configVarBool, omitting empty strings
// This is done to not have the json object cluttered with empty strings
// This will eventually hopefully be resolved within golang itself
// https://github.com/golang/go/issues/11939
func (configVarBool ConfigVarBool) MarshalJSON() ([]byte, error) {
	var secretKeyRefEmpty, configMapKeyRefEmpty bool
	if configVarBool.SecretKeyRef.ObjectReference.Namespace == "" &&
		configVarBool.SecretKeyRef.ObjectReference.Name == "" &&
		configVarBool.SecretKeyRef.Key == "" {
		secretKeyRefEmpty = true
	}

	if configVarBool.ConfigMapKeyRef.ObjectReference.Namespace == "" &&
		configVarBool.ConfigMapKeyRef.ObjectReference.Name == "" &&
		configVarBool.ConfigMapKeyRef.Key == "" {
		configMapKeyRefEmpty = true
	}

	if secretKeyRefEmpty && configMapKeyRefEmpty {
		return []byte(fmt.Sprintf("%v", configVarBool.Value)), nil
	}

	buffer := bytes.NewBufferString("{")
	if !secretKeyRefEmpty {
		jsonVal, err := json.Marshal(configVarBool.SecretKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`"secretKeyRef":%s`, string(jsonVal)))
	}

	if !configMapKeyRefEmpty {
		var leadingComma string
		if !secretKeyRefEmpty {
			leadingComma = ","
		}
		jsonVal, err := json.Marshal(configVarBool.ConfigMapKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`%s"configMapKeyRef":%s`, leadingComma, jsonVal))
	}

	buffer.WriteString(fmt.Sprintf(`,"value":%v}`, configVarBool.Value))

	return buffer.Bytes(), nil
}

func (configVarBool *ConfigVarBool) UnmarshalJSON(b []byte) error {
	if !bytes.HasPrefix(b, []byte("{")) {
		value, err := strconv.ParseBool(string(b))
		if err != nil {
			return fmt.Errorf("Error converting string to bool: '%v'", err)
		}
		configVarBool.Value = value
		return nil
	}
	var cvbDummy configVarBoolWithoutUnmarshaller
	err := json.Unmarshal(b, &cvbDummy)
	if err != nil {
		return err
	}
	configVarBool.Value = cvbDummy.Value
	configVarBool.SecretKeyRef = cvbDummy.SecretKeyRef
	configVarBool.ConfigMapKeyRef = cvbDummy.ConfigMapKeyRef
	return nil
}

func GetConfig(r clusterv1alpha1.ProviderSpec) (*Config, error) {
	if r.Value == nil {
		return nil, fmt.Errorf("machine.spec.providerSpec.value is nil")
	}
	p := new(Config)
	if len(r.Value.Raw) == 0 {
		return p, nil
	}
	if err := json.Unmarshal(r.Value.Raw, p); err != nil {
		return nil, err
	}
	return p, nil
}
