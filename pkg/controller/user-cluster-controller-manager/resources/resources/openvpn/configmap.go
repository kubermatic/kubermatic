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

package openvpn

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	v1 "k8s.io/api/core/v1"
)

const (
	Name = "openvpn"
)

// ClientConfigConfigMapCreator returns a ConfigMap containing the config for the OpenVPN client. It lives inside the user-cluster
func ClientConfigConfigMapCreator(hostname string, serverPort uint32) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.OpenVPNClientConfigConfigMapName, func(cm *v1.ConfigMap) (*v1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(Name, nil)

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
