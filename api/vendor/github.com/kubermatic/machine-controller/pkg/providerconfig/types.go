package providerconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var (
	ErrOSNotSupported = errors.New("os not supported")
)

type OperatingSystem string

const (
	OperatingSystemCoreos OperatingSystem = "coreos"
	OperatingSystemUbuntu OperatingSystem = "ubuntu"
	OperatingSystemCentOS OperatingSystem = "centos"
)

type CloudProvider string

const (
	CloudProviderAWS          CloudProvider = "aws"
	CloudProviderAzure        CloudProvider = "azure"
	CloudProviderDigitalocean CloudProvider = "digitalocean"
	CloudProviderOpenstack    CloudProvider = "openstack"
	CloudProviderHetzner      CloudProvider = "hetzner"
	CloudProviderVsphere      CloudProvider = "vsphere"
	CloudProviderFake         CloudProvider = "fake"
	CloudProviderKubeVirt     CloudProvider = "kubevirt"
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

// GlobaObjectKeySelector is needed as we can not use v1.SecretKeySelector
// because it is not cross namespace
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

// MarshalJSON converts a configVarString to its JSON form, ompitting empty strings.
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
		return "", fmt.Errorf("secret '%s' in namespace '%s' has no key '%s'", configVar.SecretKeyRef.Name, configVar.SecretKeyRef.Namespace, configVar.SecretKeyRef.Key)
	}

	// We need all three of these to fetch and use a configmap
	if configVar.ConfigMapKeyRef.Name != "" && configVar.ConfigMapKeyRef.Namespace != "" && configVar.ConfigMapKeyRef.Key != "" {
		configMap, err := configVarResolver.kubeClient.CoreV1().ConfigMaps(configVar.ConfigMapKeyRef.Namespace).Get(configVar.ConfigMapKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("error retrieving configmap '%s' from namespace '%s': '%v'", configVar.ConfigMapKeyRef.Name, configVar.ConfigMapKeyRef.Namespace, err)
		}
		if val, ok := configMap.Data[configVar.ConfigMapKeyRef.Key]; ok {
			return val, nil
		}
		return "", fmt.Errorf("configmap '%s' in namespace '%s' has no key '%s'", configVar.ConfigMapKeyRef.Name, configVar.ConfigMapKeyRef.Namespace, configVar.ConfigMapKeyRef.Key)
	}

	return configVar.Value, nil
}

// GetConfigVarStringValueOrEnv tries to get the value from ConfigVarString, when it fails, it falls back to
// getting the value from an environment variable specified by envVarName parameter
func (configVarResolver *ConfigVarResolver) GetConfigVarStringValueOrEnv(configVar ConfigVarString, envVarName string) (string, error) {
	cfgVar, err := configVarResolver.GetConfigVarStringValue(configVar)
	if err == nil && len(cfgVar) > 0 {
		return cfgVar, err
	}

	envVal, envValFound := os.LookupEnv(envVarName)
	if !envValFound {
		return "", fmt.Errorf("all machanisms(value, secret, configMap) of getting the value failed, including reading from environment variable = %s which was not set", envVarName)
	}
	return envVal, nil
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

func (configVarResolver *ConfigVarResolver) GetConfigVarBoolValueOrEnv(configVar ConfigVarBool, envVarName string) (bool, error) {
	cvs := ConfigVarString{Value: strconv.FormatBool(configVar.Value), SecretKeyRef: configVar.SecretKeyRef}
	stringVal, err := configVarResolver.GetConfigVarStringValue(cvs)
	if err != nil {
		return false, err
	}
	if stringVal == "" {
		envVal, envValFound := os.LookupEnv(envVarName)
		if !envValFound {
			return false, fmt.Errorf("all machanisms(value, secret, configMap) of getting the value failed, including reading from environment variable = %s which was not set", envVarName)
		}
		stringVal = envVal
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

func GetConfig(r clusterv1alpha1.ProviderConfig) (*Config, error) {
	if r.Value == nil {
		return nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
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
