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
	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// ConstraintTemplateResourceName represents "Resource" defined in Kubernetes
	ConstraintTemplateResourceName = "constrainttemplates"

	// ConstraintTemplateKind represents "Kind" defined in Kubernetes
	ConstraintTemplateKind = "ConstraintTemplate"
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConstraintTemplate is the object representing a kubermatic wrapper for a gatekeeper constraint template.
type ConstraintTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConstraintTemplateSpec `json:"spec"`
}

// ConstraintTemplateSpec is the object representing the gatekeeper constraint template spec and kubermatic related spec
type ConstraintTemplateSpec struct {
	CRD      v1beta1.CRD                `json:"crd,omitempty"`
	Targets  []v1beta1.Target           `json:"targets,omitempty"`
	Selector ConstraintTemplateSelector `json:"selector,omitempty"`
}

// ConstraintTemplateSelector is the object holding the cluster selection filters
type ConstraintTemplateSelector struct {
	// Providers is a list of cloud providers to which the Constraint Template applies to. Empty means all providers are selected.
	Providers []string
	// LabelSelector selects the Clusters to which the Constraint Template applies based on their labels
	LabelSelector metav1.LabelSelector
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConstraintTemplateList specifies a list of constraint templates
type ConstraintTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ConstraintTemplate `json:"items"`
}
