/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PolicyBindingResourceName represents "Resource" defined in Kubernetes.
	PolicyBindingResourceName = "policybindings"

	// PolicyBindingKindName represents "Kind" defined in Kubernetes.
	PolicyBindingKindName = "PolicyBinding"
)

// PolicyBindingConditionType defines the type of condition in PolicyBindingStatus.
//
// +kubebuilder:validation:Enum=Ready;TemplateValid;KyvernoPolicyApplied
type PolicyBindingConditionType string

// Condition types for PolicyBinding Status.
const (
	// PolicyBindingConditionReady indicates if the corresponding Kyverno policy is ready.
	PolicyBindingConditionReady PolicyBindingConditionType = "Ready"

	// PolicyBindingConditionTemplateValid indicates if the referenced PolicyTemplate is valid.
	PolicyBindingConditionTemplateValid PolicyBindingConditionType = "TemplateValid"

	// PolicyBindingConditionKyvernoPolicyApplied indicates whether the controller
	// successfully created/updated the required Kyverno Policy/ClusterPolicy resources.
	PolicyBindingConditionKyvernoPolicyApplied PolicyBindingConditionType = "KyvernoPolicyApplied"
)

// Annotation keys for PolicyBinding.
const (
	// AnnotationPolicyEnforced is added to PolicyBinding resources that were automatically created.
	AnnotationPolicyEnforced = "policy.kubermatic.k8c.io/enforced-by-template"

	// AnnotationPolicyDefault is added to PolicyBinding resources that were defaulted from the PolicyTemplate.
	AnnotationPolicyDefault = "policy.kubermatic.k8c.io/default-policy"
)

// +kubebuilder:resource:scope=Namespaced,categories=kubermatic,shortName=pb
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Template",type=string,JSONPath=".spec.policyTemplateRef.name"
// +kubebuilder:printcolumn:name="Enforced",type=boolean,JSONPath=".status.templateEnforced"
// +kubebuilder:printcolumn:name="Active",type=string,JSONPath=".status.active"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PolicyBinding binds a PolicyTemplate to specific clusters/projects and
// optionally enables or disables it (if the template is not enforced).
type PolicyBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyBindingSpec   `json:"spec,omitempty"`
	Status PolicyBindingStatus `json:"status,omitempty"`
}

// PolicyBindingSpec describes how and where to apply the referenced PolicyTemplate.
type PolicyBindingSpec struct {
	// PolicyTemplateRef references the PolicyTemplate by name
	//
	// +kubebuilder:validation:Required
	PolicyTemplateRef corev1.ObjectReference `json:"policyTemplateRef"`

	// KyvernoPolicyNamespace specifies the Kyverno namespace to deploy the Kyverno Policy into
	//
	// Relevant only if the referenced PolicyTemplate has spec.enforced=false.
	// If Template.NamespacedPolicy is true and this field is omitted, no Kyverno Policy resources will be created.
	//
	// +optional
	KyvernoPolicyNamespace *KyvernoPolicyNamespace `json:"kyvernoPolicyNamespace,omitempty"`
}

// KyvernoPolicyNamespace specifies the Kyverno namespace to deploy the Kyverno Policy into
// This is relevant only if a Kyverno Policy resource is created because Kyverno Policy is namespaced.
// For Kyverno ClusterPolicy, this field is ignored.
type KyvernoPolicyNamespace struct {
	// Name is the namespace to deploy the Kyverno Policy into.
	//
	// +kubebuilder:validation:Pattern:=`^(|[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`

	// Create defines whether the namespace should be created if it does not exist. Defaults to true
	//
	// +kubebuilder:default:=true
	Create bool `json:"create"`

	// Labels of the Kyverno namespace
	//
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations of the Kyverno namespace
	//
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PolicyBindingStatus is the status of the policy binding.
type PolicyBindingStatus struct {
	// ObservedGeneration is the generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// TemplateEnforced reflects the value of `spec.enforced` from PolicyTemplate
	//
	// +optional
	TemplateEnforced *bool `json:"templateEnforced,omitempty"`

	// Active reflects whether the Kyverno policy exists and is active in this User Cluster.
	//
	// +optional
	Active *bool `json:"active,omitempty"`

	// Conditions represents the latest available observations of the policy binding's current state
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// PolicyBindingList is a list of PolicyBinding objects.
type PolicyBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of PolicyBinding objects
	Items []PolicyBinding `json:"items"`
}
