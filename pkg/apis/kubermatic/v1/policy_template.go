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

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
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
	//
	// +optional
	Title string `json:"title,omitempty"`
	// Category is the category of the policy, specified as an annotation in the Kyverno policy
	//
	// +optional
	Category string `json:"category,omitempty"`
	// Description is the description of the policy, specified as an annotation in the Kyverno policy
	//
	// +optional
	Description string `json:"description,omitempty"`

	// Enforced indicates whether this policy is mandatory
	//
	// If true, this policy is mandatory
	// A PolicyInstance referencing it cannot disable it
	Enforced bool `json:"enforced"`

	// KyvernoSpec is the Kyverno specification
	KyvernoSpec kyvernov1.Spec `json:"kyvernoSpec,omitempty"`
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
