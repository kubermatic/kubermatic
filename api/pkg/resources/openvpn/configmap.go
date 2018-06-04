package openvpn

import (
	"fmt"
	"net"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configTpl = `iroute %s %s
iroute %s %s
`
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

	podNetIP, podNet, err := net.ParseCIDR(data.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}

	serviceNetIP, serviceNet, err := net.ParseCIDR(data.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}

	cm.Data = map[string]string{
		"user-cluster-client": fmt.Sprintf(
			configTpl,
			podNetIP.String(),
			net.IP(podNet.Mask).String(),
			serviceNetIP.String(),
			net.IP(serviceNet.Mask).String(),
		),
	}

	return cm, nil
}
