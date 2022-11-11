package cni

import (
	"bytes"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

var ciliumValues = `
cni:
  exclusive: false
hubble:
  relay:
    enabled: false
  tls:
    auto:
      method: cronJob
  ui:
    enabled: false
ipam:
  operator:
    clusterPoolIPv4MaskSize: 24
    clusterPoolIPv4PodCIDR: 172.25.0.0/16
kubeProxyReplacement: disabled
operator:
  replicas: 1
`

func getCiliumValues(cluster *kubermaticv1.Cluster) runtime.RawExtension {
	// TODO: merge with https://pkg.go.dev/go.uber.org/config/internal/merge

	decoder := kyaml.NewYAMLToJSONDecoder(bytes.NewBufferString(ciliumValues))
	raw := runtime.RawExtension{}
	if err := decoder.Decode(&raw); err != nil {
		return runtime.RawExtension{} // TODO: return error
	}

	return raw
}
