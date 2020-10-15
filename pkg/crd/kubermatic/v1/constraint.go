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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (

	// ConstraintResourceName represents "Resource" defined in Kubernetes
	ConstraintResourceName = "constraints"

	// ConstraintKind represents "Kind" defined in Kubernetes
	ConstraintKind = "Constraint"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Constraint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConstraintSpec `json:"spec,omitempty"`
}

type ConstraintSpec struct {
	// type of gatekeeper constraint that the constraint applies to
	ConstraintType string `json:"constraintType"`
	Match          `json:"match,omitempty"`
	Parameters     Parameters `json:"parameters,omitempty"`
}

type Match struct {
	Kinds              []Kind               `json:"kinds,omitempty"`
	Scope              string               `json:"scope,omitempty"`
	Namespaces         []string             `json:"namespaces,omitempty"`
	ExcludedNamespaces []string             `json:"excludedNamespaces,omitempty"`
	LabelSelector      metav1.LabelSelector `json:"labelSelector,omitempty"`
	NamespaceSelector  metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type Kind struct {
	Kinds     string `json:"kinds,omitempty"`
	APIGroups string `json:"apiGroups,omitempty"`
}

type Parameters struct {
	RawJSON string `json:"rawJSON,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConstraintList specifies a list of constraints
type ConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Constraint `json:"items"`
}
