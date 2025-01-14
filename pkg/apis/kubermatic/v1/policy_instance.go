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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PolicyInstanceResourceName represents "Resource" defined in Kubernetes.
	PolicyInstanceResourceName = "policyinstances"

	// PolicyInstanceKindName represents "Kind" defined in Kubernetes.
	PolicyInstanceKindName = "PolicyInstance"

	// PolicyInstanceScopeCluster is the cluster scope of the policy instance
	PolicyInstanceScopeCluster = "cluster"

	// PolicyInstanceScopeNamespaced is the namespaced scope of the policy instance
	PolicyInstanceScopeNamespaced = "namespaced"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=".spec.enabled",description="Whether the policy is applied (only relevant if not enforced)"
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=".spec.scope",description="cluster or namespace"

// PolicyInstance binds a PolicyTemplate to specific clusters/projects and
// optionally enables or disables it (if the template is not enforced)
type PolicyInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PolicyInstanceSpec `json:"spec,omitempty"`
}

// PolicyInstanceSpec describes how and where to apply the referenced PolicyTemplate
type PolicyInstanceSpec struct {
	// PolicyTemplateRef references the PolicyTemplate by name
	PolicyTemplateRef ObjectReference `json:"policyTemplateRef"`

	// Enabled determines whether we apply the policy
	//
	// If the referenced template is enforced, "enabled" is ignored (or disallowed)
	// If the referenced template is not enforced, "enabled" determines whether we apply the policy
	Enabled bool `json:"enabled,omitempty"`

	// Scope specifies the Kyverno Policy to be created.
	//
	// The scope can be "cluster" or "namespaced", by default it is "cluster"
	// +kubebuilder:default=cluster
	Scope string `json:"scope,omitempty"`

	// Target specifies which clusters/projects to apply the policy to
	Target PolicyTargetSpec `json:"target,omitempty"`
}

// ObjectReference is a local reference to a PolicyTemplate
type ObjectReference struct {
	// Name is the name of the policy template
	Name string `json:"name"`
}

// PolicyTargetSpec indicates how to select projects/clusters in Kubermatic
type PolicyTargetSpec struct {
	// AllProjects indicates that the policy applies to all projects
	AllProjects bool `json:"allProjects,omitempty"`

	// ProjectSelector restricts the policy to projects matching these labels
	ProjectSelector *metav1.LabelSelector `json:"projectSelector,omitempty"`

	// AllClusters indicates that the policy applies to all clusters
	AllClusters bool `json:"allClusters,omitempty"`

	// ClusterSelector restricts the policy to clusters matching these labels
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// PolicyInstanceList is a list of PolicyInstance objects.
type PolicyInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of PolicyInstance objects
	Items []PolicyInstance `json:"items"`
}
