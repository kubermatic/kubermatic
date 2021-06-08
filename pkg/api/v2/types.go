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

package v2

import (
	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	crdapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

// ConstraintTemplate represents a gatekeeper ConstraintTemplate
// swagger:model ConstraintTemplate
type ConstraintTemplate struct {
	Name string `json:"name"`

	Spec   crdapiv1.ConstraintTemplateSpec  `json:"spec"`
	Status v1beta1.ConstraintTemplateStatus `json:"status"`
}

// Constraint represents a gatekeeper Constraint
// swagger:model Constraint
type Constraint struct {
	Name string `json:"name"`

	Spec   crdapiv1.ConstraintSpec `json:"spec"`
	Status *ConstraintStatus       `json:"status,omitempty"`
}

// ConstraintStatus represents a constraint status which holds audit info
type ConstraintStatus struct {
	Enforcement    string      `json:"enforcement,omitempty"`
	AuditTimestamp string      `json:"auditTimestamp,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
	Synced         *bool       `json:"synced,omitempty"`
}

// Violation represents a gatekeeper constraint violation
type Violation struct {
	EnforcementAction string `json:"enforcementAction,omitempty"`
	Kind              string `json:"kind,omitempty"`
	Message           string `json:"message,omitempty"`
	Name              string `json:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
}

// GatekeeperConfig represents a gatekeeper config
// swagger:model GatekeeperConfig
type GatekeeperConfig struct {
	Spec GatekeeperConfigSpec `json:"spec"`
}

type GatekeeperConfigSpec struct {
	// Configuration for syncing k8s objects
	Sync Sync `json:"sync,omitempty"`

	// Configuration for validation
	Validation Validation `json:"validation,omitempty"`

	// Configuration for namespace exclusion
	Match []MatchEntry `json:"match,omitempty"`

	// Configuration for readiness tracker
	Readiness ReadinessSpec `json:"readiness,omitempty"`
}

type Sync struct {
	// If non-empty, entries on this list will be replicated into OPA
	SyncOnly []GVK `json:"syncOnly,omitempty"`
}

type Validation struct {
	// List of requests to trace. Both "user" and "kinds" must be specified
	Traces []Trace `json:"traces,omitempty"`
}

type Trace struct {
	// Only trace requests from the specified user
	User string `json:"user,omitempty"`
	// Only trace requests of the following GroupVersionKind
	Kind GVK `json:"kind,omitempty"`
	// Also dump the state of OPA with the trace. Set to `All` to dump everything.
	Dump string `json:"dump,omitempty"`
}

type MatchEntry struct {
	// Namespaces which will be excluded
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	// Processes which will be excluded in the given namespaces (sync, webhook, audit, *)
	Processes []string `json:"processes,omitempty"`
}

type ReadinessSpec struct {
	// enables stats for gatekeeper audit
	StatsEnabled bool `json:"statsEnabled,omitempty"`
}

// GVK group version kind of a resource
type GVK struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// PresetList represents a list of presets
// swagger:model PresetList
type PresetList struct {
	Items []Preset `json:"items"`
}

// Preset represents a preset
// swagger:model Preset
type Preset struct {
	Name      string           `json:"name"`
	Enabled   bool             `json:"enabled"`
	Providers []PresetProvider `json:"providers"`
}

// PresetProvider represents a preset provider
// swagger:model PresetProvider
type PresetProvider struct {
	Name    crdapiv1.ProviderType `json:"name"`
	Enabled bool                  `json:"enabled"`
}

// Alertmanager represents an Alertmanager Configuration
// swagger:model Alertmanager
type Alertmanager struct {
	Spec AlertmanagerSpec `json:"spec"`
}

type AlertmanagerSpec struct {
	// Config contains the alertmanager configuration in YAML
	Config []byte `json:"config"`
}

// SeedSettings represents settings for a Seed cluster
// swagger:model SeedSettings
type SeedSettings struct {
	// the Seed level MLA (Monitoring, Logging, and Alerting) stack settings
	MLA MLA `json:"mla"`
}

type MLA struct {
	// whether the user cluster MLA (Monitoring, Logging & Alerting) stack is enabled in the seed
	UserClusterMLAEnabled bool `json:"user_cluster_mla_enabled"`
}

// ClusterTemplate represents a ClusterTemplate object
// swagger:model ClusterTemplate
type ClusterTemplate struct {
	Name string `json:"name"`
	ID   string `json:"id"`

	ProjectID      string                `json:"projectID,omitempty"`
	User           string                `json:"user,omitempty"`
	Scope          string                `json:"scope"`
	Cluster        *apiv1.Cluster        `json:"cluster,omitempty"`
	NodeDeployment *apiv1.NodeDeployment `json:"nodeDeployment,omitempty"`
}

// ClusterTemplateList represents a ClusterTemplate list
// swagger:model ClusterTemplateList
type ClusterTemplateList []ClusterTemplate

// RuleGroup represents a rule group of recording and alerting rules.
// swagger:model RuleGroup
type RuleGroup struct {
	// contains the RuleGroup data. Ref: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#rule_group
	Data []byte `json:"data"`
	// the type of this ruleGroup applies to. It can be `Metrics`.
	Type crdapiv1.RuleGroupType `json:"type"`
}
