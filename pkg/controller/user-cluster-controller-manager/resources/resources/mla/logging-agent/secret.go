/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package loggingagent

import (
	"bytes"
	"html/template"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	MLAGatewayURL string
	TLSCertFile   string
	TLSKeyFile    string
	TLSCACertFile string
}

func SecretReconciler(config Config) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.MLALoggingAgentSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			t, err := template.New("agent").Parse(configTemplate)
			if err != nil {
				return nil, err
			}
			configBuf := bytes.Buffer{}
			if err := t.Execute(&configBuf, config); err != nil {
				return nil, err
			}
			secret.Data["config.alloy"] = configBuf.Bytes()
			secret.Labels = resources.BaseAppLabels(appName, nil)
			return secret, nil
		}
	}
}

const (
	configTemplate = `
discovery.kubernetes "logs_default_kubernetes_pods_app_kubernetes_io_name" {
        role = "pod"
}

discovery.relabel "logs_default_kubernetes_pods_app_kubernetes_io_name" {
        targets = discovery.kubernetes.logs_default_kubernetes_pods_app_kubernetes_io_name.targets

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_name"]
                target_label  = "app"
        }

        rule {
                source_labels = ["app"]
                regex         = ""
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_component"]
                target_label  = "component"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_node_name"]
                target_label  = "node_name"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace"]
                target_label  = "namespace"
        }

        rule {
                source_labels = ["namespace", "app"]
                separator     = "/"
                target_label  = "job"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_name"]
                target_label  = "pod"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_container_name"]
                target_label  = "container"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace", "__meta_kubernetes_pod_name", "__meta_kubernetes_pod_uid"]
                separator     = "_"
                target_label  = "__path__"
                replacement   = "/var/log/pods/$1/*/*.log"
        }
}

local.file_match "logs_default_kubernetes_pods_app_kubernetes_io_name" {
        path_targets = discovery.relabel.logs_default_kubernetes_pods_app_kubernetes_io_name.output
}

loki.process "logs_default_kubernetes_pods_app_kubernetes_io_name" {
        forward_to = [loki.write.logs_default.receiver]

        stage.cri { }
}

loki.source.file "logs_default_kubernetes_pods_app_kubernetes_io_name" {
        targets               = local.file_match.logs_default_kubernetes_pods_app_kubernetes_io_name.targets
        forward_to            = [loki.process.logs_default_kubernetes_pods_app_kubernetes_io_name.receiver]
        legacy_positions_file = "/run/grafana-agent/positions.yaml"
}

discovery.kubernetes "logs_default_kubernetes_pods_app" {
        role = "pod"
}

discovery.relabel "logs_default_kubernetes_pods_app" {
        targets = discovery.kubernetes.logs_default_kubernetes_pods_app.targets

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_name"]
                regex         = ".+"
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app"]
                target_label  = "app"
        }

        rule {
                source_labels = ["app"]
                regex         = ""
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_label_component"]
                target_label  = "component"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_node_name"]
                target_label  = "node_name"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace"]
                target_label  = "namespace"
        }

        rule {
                source_labels = ["namespace", "app"]
                separator     = "/"
                target_label  = "job"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_name"]
                target_label  = "pod"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_container_name"]
                target_label  = "container"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace", "__meta_kubernetes_pod_name", "__meta_kubernetes_pod_uid"]
                separator     = "_"
                target_label  = "__path__"
                replacement   = "/var/log/pods/$1/*/*.log"
        }
}

local.file_match "logs_default_kubernetes_pods_app" {
        path_targets = discovery.relabel.logs_default_kubernetes_pods_app.output
}

loki.process "logs_default_kubernetes_pods_app" {
        forward_to = [loki.write.logs_default.receiver]

        stage.cri { }
}

loki.source.file "logs_default_kubernetes_pods_app" {
        targets               = local.file_match.logs_default_kubernetes_pods_app.targets
        forward_to            = [loki.process.logs_default_kubernetes_pods_app.receiver]
        legacy_positions_file = "/run/grafana-agent/positions.yaml"
}

discovery.kubernetes "logs_default_kubernetes_pods_direct_controllers" {
        role = "pod"
}

discovery.relabel "logs_default_kubernetes_pods_direct_controllers" {
        targets = discovery.kubernetes.logs_default_kubernetes_pods_direct_controllers.targets

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_name", "__meta_kubernetes_pod_label_app"]
                separator     = ""
                regex         = ".+"
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_controller_name"]
                regex         = "[0-9a-z-.]+-[0-9a-f]{8,10}"
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_controller_name"]
                target_label  = "app"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_node_name"]
                target_label  = "node_name"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace"]
                target_label  = "namespace"
        }

        rule {
                source_labels = ["namespace", "app"]
                separator     = "/"
                target_label  = "job"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_name"]
                target_label  = "pod"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_container_name"]
                target_label  = "container"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace", "__meta_kubernetes_pod_name", "__meta_kubernetes_pod_uid"]
                separator     = "_"
                target_label  = "__path__"
                replacement   = "/var/log/pods/$1/*/*.log"
        }
}

local.file_match "logs_default_kubernetes_pods_direct_controllers" {
        path_targets = discovery.relabel.logs_default_kubernetes_pods_direct_controllers.output
}

loki.process "logs_default_kubernetes_pods_direct_controllers" {
        forward_to = [loki.write.logs_default.receiver]

        stage.cri { }
}

loki.source.file "logs_default_kubernetes_pods_direct_controllers" {
        targets               = local.file_match.logs_default_kubernetes_pods_direct_controllers.targets
        forward_to            = [loki.process.logs_default_kubernetes_pods_direct_controllers.receiver]
        legacy_positions_file = "/run/grafana-agent/positions.yaml"
}

discovery.kubernetes "logs_default_kubernetes_pods_indirect_controller" {
        role = "pod"
}

discovery.relabel "logs_default_kubernetes_pods_indirect_controller" {
        targets = discovery.kubernetes.logs_default_kubernetes_pods_indirect_controller.targets

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_name", "__meta_kubernetes_pod_label_app"]
                separator     = ""
                regex         = ".+"
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_controller_name"]
                regex         = "[0-9a-z-.]+-[0-9a-f]{8,10}"
                action        = "keep"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_controller_name"]
                regex         = "([0-9a-z-.]+)-[0-9a-f]{8,10}"
                target_label  = "app"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_node_name"]
                target_label  = "node_name"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace"]
                target_label  = "namespace"
        }

        rule {
                source_labels = ["namespace", "app"]
                separator     = "/"
                target_label  = "job"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_name"]
                target_label  = "pod"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_container_name"]
                target_label  = "container"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace", "__meta_kubernetes_pod_name", "__meta_kubernetes_pod_uid"]
                separator     = "_"
                target_label  = "__path__"
                replacement   = "/var/log/pods/$1/*/*.log"
        }
}

local.file_match "logs_default_kubernetes_pods_indirect_controller" {
        path_targets = discovery.relabel.logs_default_kubernetes_pods_indirect_controller.output
}

loki.process "logs_default_kubernetes_pods_indirect_controller" {
        forward_to = [loki.write.logs_default.receiver]

        stage.cri { }
}

loki.source.file "logs_default_kubernetes_pods_indirect_controller" {
        targets               = local.file_match.logs_default_kubernetes_pods_indirect_controller.targets
        forward_to            = [loki.process.logs_default_kubernetes_pods_indirect_controller.receiver]
        legacy_positions_file = "/run/grafana-agent/positions.yaml"
}

discovery.kubernetes "logs_default_kubernetes_other" {
        role = "pod"
}

discovery.relabel "logs_default_kubernetes_other" {
        targets = discovery.kubernetes.logs_default_kubernetes_other.targets

        rule {
                source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_name", "__meta_kubernetes_pod_label_app"]
                separator     = ""
                regex         = ".+"
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_controller_name"]
                regex         = ".+"
                action        = "drop"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_name"]
                target_label  = "app"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_label_component"]
                target_label  = "component"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_node_name"]
                target_label  = "node_name"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace"]
                target_label  = "namespace"
        }

        rule {
                source_labels = ["namespace", "app"]
                separator     = "/"
                target_label  = "job"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_name"]
                target_label  = "pod"
        }

        rule {
                source_labels = ["__meta_kubernetes_pod_container_name"]
                target_label  = "container"
        }

        rule {
                source_labels = ["__meta_kubernetes_namespace", "__meta_kubernetes_pod_name", "__meta_kubernetes_pod_uid"]
                separator     = "_"
                target_label  = "__path__"
                replacement   = "/var/log/pods/$1/*/*.log"
        }
}

local.file_match "logs_default_kubernetes_other" {
        path_targets = discovery.relabel.logs_default_kubernetes_other.output
}

loki.process "logs_default_kubernetes_other" {
        forward_to = [loki.write.logs_default.receiver]

        stage.cri { }
}

loki.source.file "logs_default_kubernetes_other" {
        targets               = local.file_match.logs_default_kubernetes_other.targets
        forward_to            = [loki.process.logs_default_kubernetes_other.receiver]
        legacy_positions_file = "/run/grafana-agent/positions.yaml"
}

loki.write "logs_default" {
        endpoint {
                url = "{{ .MLAGatewayURL }}"

                tls_config {
                        cert_file= "{{ .TLSCertFile }}"
                        key_file= "{{ .TLSKeyFile }}"
                        ca_file= "{{ .TLSCACertFile }}"
                }
        }
        external_labels = {}
}
`
)

func ClientCertificateReconciler(ca *resources.ECDSAKeyPair) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.MLALoggingAgentCertificatesSecretName,
			certificates.GetECDSAClientCertificateReconciler(
				resources.MLALoggingAgentCertificatesSecretName,
				resources.MLALoggingAgentCertificateCommonName,
				[]string{},
				resources.MLALoggingAgentClientCertSecretKey,
				resources.MLALoggingAgentClientKeySecretKey,
				func() (*resources.ECDSAKeyPair, error) { return ca, nil })
	}
}
