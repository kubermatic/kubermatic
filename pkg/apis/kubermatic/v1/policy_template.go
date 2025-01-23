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
	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PolicyTemplateResourceName represents "Resource" defined in Kubernetes.
	PolicyTemplateResourceName = "policytemplates"

	// PolicyTemplateKind represents "Kind" defined in Kubernetes.
	PolicyTemplateKind = "PolicyTemplate"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Enforced",type=boolean,JSONPath=".spec.enforced",description="Whether the policy is mandatory"

// PolicyTemplate defines a reusable blueprint of a Kyverno policy.
type PolicyTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PolicyTemplateSpec `json:"spec,omitempty"`
}

type PolicyTemplateSpec struct {
	// Title is the title of the policy, specified as an annotation in the Kyverno policy
	Title string `json:"title"`

	// Description is the description of the policy, specified as an annotation in the Kyverno policy
	Description string `json:"description"`

	// Category is the category of the policy, specified as an annotation in the Kyverno policy
	//
	// +optional
	Category string `json:"category,omitempty"`

	// Severity indicates the severity level of the policy
	//
	// +optional
	Severity string `json:"severity,omitempty"`

	// Visibility specifies where the policy is visible.
	//
	// Can be one of: global, project, or cluster
	// +kubebuilder:validation:Enum=global;project;cluster
	Visibility string `json:"visibility"`

	// ProjectID is the ID of the project for which the policy template is created
	//
	// Relevant only for project visibility policies
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// Default determines whether we apply the policy (create policy binding)
	//
	// +optional
	Default bool `json:"default,omitempty"`

	// Enforced indicates whether this policy is mandatory
	//
	// If true, this policy is mandatory
	// A PolicyBinding referencing it cannot disable it
	Enforced bool `json:"enforced"`

	// KyvernoSpec is the Kyverno specification
	kyvernov1.Spec `json:",inline"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// PolicyTemplateList is a list of PolicyTemplate objects.
type PolicyTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of PolicyTemplate objects
	Items []PolicyTemplate `json:"items"`
}
