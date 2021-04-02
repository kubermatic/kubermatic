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

package nodelocaldns

import (
	"bytes"
	"html/template"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	addonManagerModeKey = "addonmanager.kubernetes.io/mode"
	reconcileModeValue  = "Reconcile"
)

// ConfigMapCreator returns a ConfigMap containing the config for Node Local DNS cache
func ConfigMapCreator(dnsClusterIP string) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.NodeLocalDNSConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Labels == nil {
				cm.Labels = map[string]string{}
			}
			cm.Labels[addonManagerModeKey] = reconcileModeValue

			t, err := template.New("config").Parse(configTemplate)
			if err != nil {
				return nil, err
			}
			configBuf := bytes.Buffer{}
			if err := t.Execute(&configBuf, struct{ DNSClusterIP string }{dnsClusterIP}); err != nil {
				return nil, err
			}

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Data["Corefile"] = configBuf.String()
			return cm, nil
		}
	}
}

const (
	configTemplate = `
cluster.local:53 {
    errors
    cache {
            success 9984 30
            denial 9984 5
    }
    reload
    loop
    bind 169.254.20.10
    forward . {{ .DNSClusterIP }} {
            force_tcp
    }
    prometheus :9253
    health 169.254.20.10:8080
    }
in-addr.arpa:53 {
    errors
    cache 30
    reload
    loop
    bind 169.254.20.10
    forward . {{ .DNSClusterIP }} {
            force_tcp
    }
    prometheus :9253
    }
ip6.arpa:53 {
    errors
    cache 30
    reload
    loop
    bind 169.254.20.10
    forward . {{ .DNSClusterIP }} {
            force_tcp
    }
    prometheus :9253
    }
.:53 {
    errors
    cache 30
    reload
    loop
    bind 169.254.20.10
    forward . /etc/resolv.conf
    prometheus :9253
    }
  `
)
