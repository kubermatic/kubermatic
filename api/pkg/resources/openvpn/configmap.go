package openvpn

import (
	"fmt"
	"net"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap returns a ConfigMap containing the openvpn config
func ConfigMap(data *resources.TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}

	cm.Name = resources.OpenVPNClientConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.GetLabels(name)

	var iroutes []string

	// iroute for pod network
	_, podNet, err := net.ParseCIDR(data.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}
	iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
		podNet.IP.String(),
		net.IP(podNet.Mask).String()))

	// iroute for service network
	_, serviceNet, err := net.ParseCIDR(data.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}
	iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
		serviceNet.IP.String(),
		net.IP(serviceNet.Mask).String()))

	_, nodeAccessNetwork, err := net.ParseCIDR(data.NodeAccessNetwork)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node access network %s: %v", data.NodeAccessNetwork, err)
	}
	iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
		nodeAccessNetwork.IP.String(),
		net.IP(nodeAccessNetwork.Mask).String()))

	// trailing newline
	iroutes = append(iroutes, "")
	cm.Data = map[string]string{
		"user-cluster-client": strings.Join(iroutes, "\n"),
	}

	return cm, nil
}
