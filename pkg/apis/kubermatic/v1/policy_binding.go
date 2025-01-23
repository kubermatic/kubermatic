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

// Condition reasons for PolicyBinding.
const (
	// PolicyAppliedSuccessfully indicates the policy was successfully applied.
	PolicyAppliedSuccessfully = "PolicyAppliedSuccessfully"

	// PolicyApplicationFailed indicates the policy application failed.
	PolicyApplicationFailed = "PolicyApplicationFailed"

	// PolicyTemplateNotFound indicates the referenced template doesn't exist.
	PolicyTemplateNotFound = "PolicyTemplateNotFound"
)

// Condition types for PolicyBinding.
const (
	// PolicyReady indicates if the policy has been successfully applied.
	PolicyReadyCondition = "Ready"

	// PolicyEnforced indicates if the policy is currently being enforced.
	PolicyEnforcedCondition = "Enforced"
)

// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=".spec.enabled",description="Whether the policy is applied (only relevant if not enforced)"
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=".spec.scope",description="cluster or namespace"

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
	PolicyTemplateRef corev1.ObjectReference `json:"policyTemplateRef"`

	// NamespacedPolicy is a boolean to indicate if the policy binding is namespaced
	NamespacedPolicy bool `json:"namespacedPolicy,omitempty"`

	// Scope specifies the scope of the policy.
	// Can be one of: global, project, or cluster
	//
	// +kubebuilder:validation:Enum=global;project;cluster
	Scope string `json:"scope"`

	// Target specifies which clusters/projects to apply the policy to
	Target PolicyTargetSpec `json:"target,omitempty"`
}

// PolicyTargetSpec indicates how to select projects/clusters in Kubermatic.
type PolicyTargetSpec struct {
	// Projects is a list of projects to apply the policy to
	Projects ResourceSelector `json:"projects,omitempty"`

	// Clusters is a list of clusters to apply the policy to
	Clusters ResourceSelector `json:"clusters,omitempty"`
}

// ResourceSelector is a struct that contains the label selector, name, and selectAll fields.
type ResourceSelector struct {
	// LabelSelector is a label selector to select the resources (projects/clusters)
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	// Name is a list of names to select the resources (projects/clusters)
	Name []string `json:"name,omitempty"`
	// SelectAll is a boolean to select all the resources (projects/clusters) from cluster admins.
	SelectAll bool `json:"selectAll,omitempty"`
}

// PolicyBindingStatus is the status of the policy binding.
type PolicyBindingStatus struct {
	// ObservedGeneration is the generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

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
