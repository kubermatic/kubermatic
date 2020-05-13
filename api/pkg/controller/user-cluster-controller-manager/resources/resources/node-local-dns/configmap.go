package nodelocaldns

import (
	"bytes"
	"html/template"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	addonManagerModeKey = "addonmanager.kubernetes.io/mode"
	reconcilModeValue   = "Reconcile"
)

// ConfigMapCreator returns a ConfigMap containing the config for Node Local DNS cache
func ConfigMapCreator(dnsClusterIP string) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.NodeLocalDNSConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Labels == nil {
				cm.Labels = map[string]string{}
			}
			cm.Labels[addonManagerModeKey] = reconcilModeValue

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
    forward . /etc/resolv.conf {
            force_tcp
    }
    prometheus :9253
    }
  `
)
