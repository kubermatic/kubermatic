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
	templatesv1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// ConstraintTemplateResourceName represents "Resource" defined in Kubernetes.
	ConstraintTemplateResourceName = "constrainttemplates"

	// ConstraintTemplateKind represents "Kind" defined in Kubernetes.
	ConstraintTemplateKind = "ConstraintTemplate"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// ConstraintTemplate is the object representing a kubermatic wrapper for a gatekeeper constraint template.
type ConstraintTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec specifies the gatekeeper constraint template and KKP related spec.
	Spec ConstraintTemplateSpec `json:"spec,omitempty"`
}

// ConstraintTemplateSpec is the object representing the gatekeeper constraint template spec and kubermatic related spec.
type ConstraintTemplateSpec struct {
	CRD     templatesv1.CRD      `json:"crd,omitempty"`
	Targets []templatesv1.Target `json:"targets,omitempty"`

	// Selector configures which clusters this constraint template is applied to.
	Selector ConstraintTemplateSelector `json:"selector,omitempty"`
}

// ConstraintTemplateSelector is the object holding the cluster selection filters.
type ConstraintTemplateSelector struct {
	// Providers is a list of cloud providers to which the Constraint Template applies to. Empty means all providers are selected.
	Providers []string `json:"providers,omitempty"`
	// LabelSelector selects the Clusters to which the Constraint Template applies based on their labels
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ConstraintTemplateList specifies a list of constraint templates.
type ConstraintTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of ConstraintTemplate objects.
	Items []ConstraintTemplate `json:"items"`
}
