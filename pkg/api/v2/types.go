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

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

// ConstraintTemplate represents a gatekeeper ConstraintTemplate
// swagger:model ConstraintTemplate
type ConstraintTemplate struct {
	Name string `json:"name"`

	Spec   v1beta1.ConstraintTemplateSpec   `json:"spec"`
	Status v1beta1.ConstraintTemplateStatus `json:"status"`
}

// Constraint represents a gatekeeper Constraint
// swagger:model Constraint
type Constraint struct {
	Name string `json:"name"`

	Spec   v1.ConstraintSpec `json:"spec"`
	Status *ConstraintStatus `json:"status,omitempty"`
}

// ConstraintStatus represents a constraint status which holds audit info
type ConstraintStatus struct {
	Enforcement    string      `json:"enforcement,omitempty"`
	AuditTimestamp string      `json:"auditTimestamp,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
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

type PresetList struct {
	Items []Preset `json:"items"`
}

type Preset struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}
