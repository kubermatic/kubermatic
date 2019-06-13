package openvpn

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	v1 "k8s.io/api/core/v1"
)

const (
	Name = "openvpn"
)

// ClientConfigConfigMapCreator returns a ConfigMap containing the config for the OpenVPN client. It lives inside the user-cluster
func ClientConfigConfigMapCreator(hostname string, serverPort int) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.OpenVPNClientConfigConfigMapName, func(cm *v1.ConfigMap) (*v1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabel(Name, nil)

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
cipher AES-256-GCM
auth SHA1
keysize 256
status /run/openvpn-status
up '/bin/sh -c "/sbin/iptables -t nat -I POSTROUTING -s 10.20.0.0/24 -j MASQUERADE"'
log /dev/stdout
`, hostname, serverPort)

			cm.Data["config"] = config

			return cm, nil
		}
	}
}
