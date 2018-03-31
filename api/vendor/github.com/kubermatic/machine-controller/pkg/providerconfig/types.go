package providerconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrOSNotSupported = errors.New("os not supported")
)

type OperatingSystem string

const (
	OperatingSystemCoreos OperatingSystem = "coreos"
	OperatingSystemUbuntu OperatingSystem = "ubuntu"
)

type CloudProvider string

const (
	CloudProviderAWS          CloudProvider = "aws"
	CloudProviderDigitalocean CloudProvider = "digitalocean"
	CloudProviderOpenstack    CloudProvider = "openstack"
	CloudProviderHetzner      CloudProvider = "hetzner"
)

type Config struct {
	SSHPublicKeys []string `json:"sshPublicKeys"`

	CloudProvider     CloudProvider        `json:"cloudProvider,omitempty"`
	CloudProviderSpec runtime.RawExtension `json:"cloudProviderSpec,omitempty"`

	OperatingSystem     OperatingSystem      `json:"operatingSystem"`
	OperatingSystemSpec runtime.RawExtension `json:"operatingSystemSpec"`
}

// We can not use v1.SecretKeySelector because it is not cross namespace
type GlobaObjectKeySelector struct {
	v1.ObjectReference `json:",inline"`
	Key                string `json:"key"`
}

type GlobalSecretKeySelector GlobaObjectKeySelector
type GlobalConfigMapKeySelector GlobaObjectKeySelector

type ConfigVarString struct {
	Value           string                     `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

// This type only exists to have the same fields as ConfigVarString but
// not its funcs, so it can be used as target for json.Unmarshal without
// causing a recursion
type configVarStringWithoutUnmarshaller ConfigVarString

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

func (configVarBool *ConfigVarBool) UnmarshalJSON(b []byte) error {
	if !bytes.HasPrefix(b, []byte("{")) {
		b = bytes.TrimPrefix(b, []byte(`"`))
		b = bytes.TrimSuffix(b, []byte(`"`))
		value, err := strconv.ParseBool(string(b))
		if err != nil {
			return fmt.Errorf("Error converting string to bool: '%v'", err)
		}
		configVarBool.Value = value
		return nil
	}
	var cvsDummy configVarStringWithoutUnmarshaller
	err := json.Unmarshal(b, &cvsDummy)
	// Assume error was caused by `Value` being a bool, not a string
	if err != nil {
		var cvbDummy configVarBoolWithoutUnmarshaller
		err := json.Unmarshal(b, &cvbDummy)
		if err != nil {
			return err
		}
		configVarBool.Value = cvbDummy.Value
		configVarBool.SecretKeyRef = cvbDummy.SecretKeyRef
		configVarBool.ConfigMapKeyRef = cvsDummy.ConfigMapKeyRef
		return nil
	}
	value, err := strconv.ParseBool(cvsDummy.Value)
	if err != nil {
		return fmt.Errorf("Error converting string value to bool: '%v'", err)
	}
	configVarBool.Value = value
	configVarBool.SecretKeyRef = cvsDummy.SecretKeyRef
	return nil
}

type ConfigVarResolver struct {
	kubeClient kubernetes.Interface
}

func (configVarResolver *ConfigVarResolver) GetConfigVarStringValue(configVar ConfigVarString) (string, error) {
	// We need all three of these to fetch and use a secret
	if configVar.SecretKeyRef.Name != "" && configVar.SecretKeyRef.Namespace != "" && configVar.SecretKeyRef.Key != "" {
		secret, err := configVarResolver.kubeClient.CoreV1().Secrets(
			configVar.SecretKeyRef.Namespace).Get(configVar.SecretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("error retrieving secret '%s' from namespace '%s': '%v'", configVar.SecretKeyRef.Name, configVar.SecretKeyRef.Namespace, err)
		}
		if val, ok := secret.Data[configVar.SecretKeyRef.Key]; ok {
			return string(val), nil
		}
		return "", fmt.Errorf("secret '%s' in namespace '%s' has no key '%s'!", configVar.SecretKeyRef.Name, configVar.SecretKeyRef.Namespace, configVar.SecretKeyRef.Key)
	}

	// We need all three of these to fetch and use a configmap
	if configVar.ConfigMapKeyRef.Name != "" && configVar.ConfigMapKeyRef.Namespace != "" && configVar.ConfigMapKeyRef.Key != "" {
		configMap, err := configVarResolver.kubeClient.CoreV1().ConfigMaps(configVar.ConfigMapKeyRef.Namespace).Get(configVar.ConfigMapKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("error retrieving configmap '%s' from namespace '%s': '%v'", configVar.ConfigMapKeyRef.Name, configVar.ConfigMapKeyRef.Namespace, err)
		}
		if val, ok := configMap.Data[configVar.ConfigMapKeyRef.Key]; ok {
			return string(val), nil
		}
		return "", fmt.Errorf("configmap '%s' in namespace '%s' has no key '%s'!", configVar.ConfigMapKeyRef.Name, configVar.ConfigMapKeyRef.Namespace, configVar.ConfigMapKeyRef.Key)
	}

	return configVar.Value, nil
}

func (configVarResolver *ConfigVarResolver) GetConfigVarBoolValue(configVar ConfigVarBool) (bool, error) {
	cvs := ConfigVarString{Value: strconv.FormatBool(configVar.Value), SecretKeyRef: configVar.SecretKeyRef}
	stringVal, err := configVarResolver.GetConfigVarStringValue(cvs)
	if err != nil {
		return false, err
	}
	boolVal, err := strconv.ParseBool(stringVal)
	if err != nil {
		return false, err
	}
	return boolVal, nil
}

func NewConfigVarResolver(kubeClient kubernetes.Interface) *ConfigVarResolver {
	return &ConfigVarResolver{kubeClient: kubeClient}
}

func GetConfig(r runtime.RawExtension) (*Config, error) {
	p := new(Config)
	if len(r.Raw) == 0 {
		return p, nil
	}
	if err := json.Unmarshal(r.Raw, p); err != nil {
		return nil, err
	}
	return p, nil
}
