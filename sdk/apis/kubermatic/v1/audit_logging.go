/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:validation:Enum="";metadata;recommended;minimal

// AuditPolicyPreset refers to a pre-defined set of audit policy rules. Supported values
// are `metadata`, `recommended` and `minimal`. See KKP documentation for what each policy preset includes.
type AuditPolicyPreset string

const (
	AuditPolicyMetadata    AuditPolicyPreset = "metadata"
	AuditPolicyRecommended AuditPolicyPreset = "recommended"
	AuditPolicyMinimal     AuditPolicyPreset = "minimal"
)

type AuditSidecarSettings struct {
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	Config    *AuditSidecarConfiguration   `json:"config,omitempty"`
	// ExtraEnvs are the additional environment variables that can be set for the audit logging sidecar.
	// Additional environment variables can be set and passed to the AuditSidecarConfiguration field
	// to allow passing variables to the fluent-bit configuration.
	// Only, `Value` field is supported for the environment variables; `ValueFrom` field is not supported.
	// By default, `CLUSTER_ID` is set as an environment variable in the audit-logging sidecar.
	ExtraEnvs []corev1.EnvVar `json:"extraEnvs,omitempty"`
}

// AuditSidecarConfiguration defines custom configuration for the fluent-bit sidecar deployed with a kube-apiserver.
// Also see https://docs.fluentbit.io/manual/v/1.8/administration/configuring-fluent-bit/configuration-file.
type AuditSidecarConfiguration struct {
	Service map[string]string   `json:"service,omitempty"`
	Filters []map[string]string `json:"filters,omitempty"`
	Outputs []map[string]string `json:"outputs,omitempty"`
}

// AuditLoggingSettings configures audit logging functionality.
type AuditLoggingSettings struct {
	// Enabled will enable or disable audit logging.
	Enabled bool `json:"enabled,omitempty"`
	// Optional: PolicyPreset can be set to utilize a pre-defined set of audit policy rules.
	PolicyPreset AuditPolicyPreset `json:"policyPreset,omitempty"`
	// Optional: Configures the fluent-bit sidecar deployed alongside kube-apiserver.
	SidecarSettings *AuditSidecarSettings `json:"sidecar,omitempty"`
	// Optional: Configures the webhook backend for audit logs.
	WebhookBackend *AuditWebhookBackendSettings `json:"webhookBackend,omitempty"`
}

// AuditWebhookBackendSettings configures webhook backend for audit logging functionality.
type AuditWebhookBackendSettings struct {
	// Required : AuditWebhookConfig contains reference to secret holding the audit webhook config file
	AuditWebhookConfig *corev1.SecretReference `json:"auditWebhookConfig"`
	// +kubebuilder:default="10s"
	AuditWebhookInitialBackoff string `json:"auditWebhookInitialBackoff,omitempty"`
}
