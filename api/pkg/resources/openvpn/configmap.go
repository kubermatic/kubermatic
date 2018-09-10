package openvpn

import (
	"fmt"
	"net"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerClientConfigsConfigMap returns a ConfigMap containing the ClientConfig for the OpenVPN server. It lives inside the seed-cluster
func ServerClientConfigsConfigMap(data resources.ConfigMapDataProvider, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	cm := existing
	if cm == nil {
		cm = &corev1.ConfigMap{}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	cm.Name = resources.OpenVPNClientConfigsConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.BaseAppLabel(name, nil)

	var iroutes []string

	// iroute for pod network
	if len(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks) < 1 {
		return nil, fmt.Errorf("cluster.Spec.ClusterNetwork.Pods.CIDRBlocks must contain at least one entry")
	}
	_, podNet, err := net.ParseCIDR(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}
	iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
		podNet.IP.String(),
		net.IP(podNet.Mask).String()))

	// iroute for service network
	if len(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks) < 1 {
		return nil, fmt.Errorf("cluster.Spec.ClusterNetwork.Services.CIDRBlocks must contain at least one entry")
	}
	_, serviceNet, err := net.ParseCIDR(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}
	iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
		serviceNet.IP.String(),
		net.IP(serviceNet.Mask).String()))

	_, nodeAccessNetwork, err := net.ParseCIDR(data.NodeAccessNetwork())
	if err != nil {
		return nil, fmt.Errorf("failed to parse node access network %s: %v", data.NodeAccessNetwork(), err)
	}
	iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
		nodeAccessNetwork.IP.String(),
		net.IP(nodeAccessNetwork.Mask).String()))

	// trailing newline
	iroutes = append(iroutes, "")
	cm.Data["user-cluster-client"] = strings.Join(iroutes, "\n")

	return cm, nil
}

// ClientConfigConfigMap returns a ConfigMap containing the config for the OpenVPN client. It lives inside the user-cluster
func ClientConfigConfigMap(data resources.ConfigMapDataProvider, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	cm := existing
	if cm == nil {
		cm = &corev1.ConfigMap{}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	cm.Name = resources.OpenVPNClientConfigConfigMapName
	cm.Namespace = metav1.NamespaceSystem
	cm.Labels = resources.BaseAppLabel(name, nil)

	openvpnSvc, err := data.ServiceLister().Services(data.Cluster().Status.NamespaceName).Get(resources.OpenVPNServerServiceName)
	if err != nil {
		return nil, err
	}

	config := fmt.Sprintf(`client
proto tcp
dev kube
dev-type tun
auth-nocache
remote %s %d
nobind
ca '/etc/openvpn/certs/ca.crt'
cert '/etc/openvpn/certs/client.crt'
key '/etc/openvpn/certs/client.key'
remote-cert-tls server
script-security 2
link-mtu 1432
cipher AES-256-GCM
auth SHA1
keysize 256
status /run/openvpn-status
up '/bin/sh -c "/sbin/iptables -t nat -I POSTROUTING -s 10.20.0.0/24 -j MASQUERADE"'
log /dev/stdout
`, data.Cluster().Address.ExternalName, openvpnSvc.Spec.Ports[0].NodePort)

	cm.Data["config"] = config

	return cm, nil
}
