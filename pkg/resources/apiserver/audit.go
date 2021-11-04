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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

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
  # log all changes to workloads (Pods, Deployments, etc) in full, including request and response
  - level: RequestResponse
    verbs: ["create", "delete", "update", "patch"]
    resources:
      - group: ""
        resources: ["pods"]
      - group: "apps"
        resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
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
  # log all changes to workloads (Pods, Deployments, etc) in full, including request and response
  - level: RequestResponse
    verbs: ["create", "delete", "update", "patch"]
    resources:
      - group: ""
        resources: ["pods"]
      - group: "apps"
        resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
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

func AuditConfigMapCreator(data *resources.TemplateData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.AuditConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			// set the audit policy preset so we generate a ConfigMap in any case.
			// It won't be used if audit logging is not enabled
			preset := kubermaticv1.AuditPolicyMetadata
			if data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled && data.Cluster().Spec.AuditLogging.PolicyPreset != "" {
				preset = data.Cluster().Spec.AuditLogging.PolicyPreset
			}

			cm.Data = map[string]string{
				"policy.yaml": auditPolicies[preset],
			}
			return cm, nil
		}
	}
}
