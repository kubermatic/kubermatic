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

package v1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// ConstraintResourceName represents "Resource" defined in Kubernetes.
	ConstraintResourceName = "constraints"

	// ConstraintKind represents "Kind" defined in Kubernetes.
	ConstraintKind = "Constraint"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Constraint specifies a kubermatic wrapper for the gatekeeper constraints.
type Constraint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the desired state for the constraint.
	Spec ConstraintSpec `json:"spec,omitempty"`
}

// ConstraintSpec specifies the data for the constraint.
type ConstraintSpec struct {
	// ConstraintType specifies the type of gatekeeper constraint that the constraint applies to
	ConstraintType string `json:"constraintType"`
	// Disabled  is the flag for disabling OPA constraints
	Disabled bool `json:"disabled,omitempty"`
	// Match contains the constraint to resource matching data
	Match Match `json:"match,omitempty"`
	// Parameters specifies the parameters used by the constraint template REGO.
	// It supports both the legacy rawJSON parameters, in which all the parameters are set in a JSON string, and regular
	// parameters like in Gatekeeper Constraints.
	// If rawJSON is set, during constraint syncing to the user cluster, the other parameters are ignored
	// Example with rawJSON parameters:
	//
	// parameters:
	//   rawJSON: '{"labels":["gatekeeper"]}'
	//
	// And with regular parameters:
	//
	// parameters:
	//   labels: ["gatekeeper"]
	//
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters Parameters `json:"parameters,omitempty"`
	// Selector specifies the cluster selection filters
	Selector ConstraintSelector `json:"selector,omitempty"`

	// EnforcementAction defines the action to take in response to a constraint being violated.
	// By default, EnforcementAction is set to deny as the default behavior is to deny admission requests with any violation.
	EnforcementAction string `json:"enforcementAction,omitempty"`
}

type Parameters map[string]json.RawMessage

// ConstraintSelector is the object holding the cluster selection filters.
type ConstraintSelector struct {
	// Providers is a list of cloud providers to which the Constraint applies to. Empty means all providers are selected.
	Providers []string `json:"providers,omitempty"`
	// LabelSelector selects the Clusters to which the Constraint applies based on their labels
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// Match contains the constraint to resource matching data.
type Match struct {
	// Kinds accepts a list of objects with apiGroups and kinds fields that list the groups/kinds of objects to which
	// the constraint will apply. If multiple groups/kinds objects are specified, only one match is needed for the resource to be in scope
	Kinds []Kind `json:"kinds,omitempty"`
	// Scope accepts *, Cluster, or Namespaced which determines if cluster-scoped and/or namespace-scoped resources are selected. (defaults to *)
	Scope string `json:"scope,omitempty"`
	// Namespaces is a list of namespace names. If defined, a constraint will only apply to resources in a listed namespace.
	Namespaces []string `json:"namespaces,omitempty"`
	// ExcludedNamespaces is a list of namespace names. If defined, a constraint will only apply to resources not in a listed namespace.
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	// LabelSelector is a standard Kubernetes label selector.
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
	// NamespaceSelector  is a standard Kubernetes namespace selector. If defined, make sure to add Namespaces to your
	// configs.config.gatekeeper.sh object to ensure namespaces are synced into OPA
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// Kind specifies the resource Kind and APIGroup.
type Kind struct {
	// Kinds specifies the kinds of the resources
	Kinds []string `json:"kinds,omitempty"`
	// APIGroups specifies the APIGroups of the resources
	APIGroups []string `json:"apiGroups,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ConstraintList specifies a list of constraints.
type ConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of Gatekeeper Constraints
	Items []Constraint `json:"items"`
}
