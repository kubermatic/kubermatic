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

package apiserver

import (
	"bytes"
	"html/template"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

var auditPolicies = map[kubermaticv1.AuditPolicyPreset]string{
	kubermaticv1.AuditPolicyMetadata: `# policyPreset: metadata
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: Metadata
`,
	kubermaticv1.AuditPolicyRecommended: `# policyPreset: recommended
apiVersion: audit.k8s.io/v1
kind:  Policy
omitStages:
  - "RequestReceived"
rules:
  - level: RequestResponse
    verbs: ["create", "delete", "update", "patch"]
    resources:
      # log all changes to workloads (Pods, Deployments, etc)
      - group: ""
        resources: ["pods"]
      - group: "apps"
        resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
      # log all changes to machines and higher level machine objects
      - group: "cluster.k8s.io"
        resources: ["machines", "machinesets", "machinedeployments"]
      # log all changes to Gatekeeper templates
      - group: "templates.gatekeeper.sh"
      # this secret controls SSH access to nodes if user-ssh-keys-agent is enabled
      # and it is included in audit logging because of that
      - group: ""
        resources: ["secrets"]
        resourceNames: ["usersshkeys"]
  # log extended information for requests that access pods via shell or network proxying
  - level: RequestResponse
    resources:
      - group: ""
        resources: ["pods/exec", "pods/portforward", "pods/proxy", "services/proxy"]
  # every other request will be logged at the metadata level
  - level: Metadata
    omitStages:
      - "RequestReceived"
`,
	kubermaticv1.AuditPolicyMinimal: `# policyPreset: minimal
apiVersion: audit.k8s.io/v1
kind: Policy
omitStages:
  - "RequestReceived"
rules:
  - level: RequestResponse
    verbs: ["create", "delete", "update", "patch"]
    resources:
      # log all changes to workloads (Pods, Deployments, etc)
      - group: ""
        resources: ["pods"]
      - group: "apps"
        resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
      # log all changes to machines and higher level machine objects
      - group: "cluster.k8s.io"
        resources: ["machines", "machinesets", "machinedeployments"]
      # log all changes to Gatekeeper templates
      - group: "templates.gatekeeper.sh"
      # this secret controls SSH access to nodes if user-ssh-keys-agent is enabled
      # and it is included in audit logging because of that
      - group: ""
        resources: ["secrets"]
        resourceNames: ["usersshkeys"]
  # log extended information for requests that access pods in an way (shell or network)
  - level: Request
    resources:
      - group: ""
        resources: ["pods/exec", "pods/portforward", "pods/proxy", "services/proxy"]
  - level: Metadata
    resources:
      # log metadata for access to pod logs
      - group: ""
        resources: ["pods/log"]
      # log metadata for all requests that are related to Secrets and ConfigMaps
      - group: ""
        resources: ["secrets", "configmaps"]
`,
}

const fluentBitConfigTemplate = `
{{- if .Service }}
[SERVICE]
{{- range $key, $value := .Service }}
    {{ $key }}      {{ $value }}
{{- end }}
{{- end }}

[INPUT]
    Name    tail
    Path    /var/log/kubernetes/audit/audit.log
    DB      /var/log/kubernetes/audit/fluentbit.db

{{- if .Filters }}
{{- range $filter := .Filters }}
[FILTER]
{{- range $key, $value := $filter }}
    {{ $key }}      {{ $value }}
{{- end }}

{{- end }}
{{- end }}

{{- if .Outputs }}
{{- range $output := .Outputs }}
[OUTPUT]
{{- range $key, $value := $output }}
    {{ $key }}      {{ $value }}
{{- end }}

{{- end }}
{{- else }}
[OUTPUT]
    Name    stdout
    Match   *
{{- end }}

`

func AuditConfigMapReconciler(data *resources.TemplateData) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return resources.AuditConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			// set the audit policy preset so we generate a ConfigMap in any case.
			// It won't be used if audit logging and audit webhook are not enabled
			preset := kubermaticv1.AuditPolicyPreset("")
			if data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled && data.Cluster().Spec.AuditLogging.PolicyPreset != "" {
				preset = data.Cluster().Spec.AuditLogging.PolicyPreset
			}

			if data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.WebhookBackend != nil && data.Cluster().Spec.AuditLogging.PolicyPreset != "" {
				preset = data.Cluster().Spec.AuditLogging.PolicyPreset
			}

			// if the policyPreset field is empty, only update the ConfigMap on creation
			if preset != "" || cm.Data == nil {
				// if the preset is empty, set it to 'metadata' to generate a valid audit policy
				if preset == "" {
					preset = kubermaticv1.AuditPolicyMetadata
				}

				cm.Data = map[string]string{
					"policy.yaml": auditPolicies[preset],
				}
			}
			return cm, nil
		}
	}
}

// FluentBitSecretReconciler returns a reconciling.NamedSecretReconcilerFactory for a secret that contains
// fluent-bit configuration for the audit-logs sidecar.
func FluentBitSecretReconciler(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.FluentBitSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}

			config := &kubermaticv1.AuditSidecarConfiguration{}

			var err error
			config, err = data.ParseFluentBitRecords()
			if err != nil {
				return nil, err
			}

			t, err := template.New("fluent-bit.conf").Parse(fluentBitConfigTemplate)
			if err != nil {
				return nil, err
			}

			configBuf := bytes.Buffer{}
			if err := t.Execute(&configBuf, config); err != nil {
				return nil, err
			}

			secret.Data["fluent-bit.conf"] = configBuf.Bytes()

			return secret, nil
		}
	}
}
