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
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// PolicyTemplateResourceName represents "Resource" defined in Kubernetes.
	PolicyTemplateResourceName = "policytemplates"

	// PolicyTemplateKind represents "Kind" defined in Kubernetes.
	PolicyTemplateKind = "PolicyTemplate"
)

const (
	// PolicyTemplateSeedCleanupFinalizer indicates that synced policy template on seed clusters need cleanup.
	PolicyTemplateSeedCleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-policy-template"
)

// PolicyTemplateVisibilityType defines the primary intended audience/scope for the template.
//
// +kubebuilder:validation:Enum=Global;Project;Cluster
type PolicyTemplateVisibilityType string

const (
	// PolicyTemplateVisibilityGlobal is the visibility of the policy template when it is created in the global scope.
	PolicyTemplateVisibilityGlobal = "Global"

	// PolicyTemplateVisibilityProject is the visibility of the policy template when it is created in the project scope.
	PolicyTemplateVisibilityProject = "Project"

	// PolicyTemplateVisibilityCluster is the visibility of the policy template when it is created in the cluster scope.
	PolicyTemplateVisibilityCluster = "Cluster"
)

// +kubebuilder:resource:scope=Cluster,categories=kubermatic,shortName=pt
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Visibility",type=string,JSONPath=".spec.visibility"
// +kubebuilder:printcolumn:name="ProjectID",type=string,JSONPath=".spec.projectID"
// +kubebuilder:printcolumn:name="Enforced",type=boolean,JSONPath=".spec.enforced"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
	// +kubebuilder:validation:Enum=Global;Project;Cluster
	// +kubebuilder:validation:Required
	Visibility string `json:"visibility"`

	// ProjectID is the ID of the project for which the policy template is created
	//
	// Relevant only for project visibility policies
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// Default determines whether we apply the policy (create policy binding) by default
	//
	// +optional
	Default bool `json:"default,omitempty"`

	// Enforced indicates whether this policy is mandatory
	//
	// If true, this policy is mandatory
	// A PolicyBinding referencing it cannot disable it
	Enforced bool `json:"enforced"`

	// Target allows selection of projects and clusters where this template applies,
	// If 'Target' itself is omitted, the scope defaults based on 'Visibility' and 'ProjectID':
	//
	// +optional
	Target *PolicyTemplateTarget `json:"target,omitempty"`

	// PolicySpec is the policy spec of the Kyverno Policy we want to apply on the cluster.
	//
	// The structure of this spec should follow the rules defined in Kyverno
	// [Writing Policies Docs](https://kyverno.io/docs/writing-policies/).
	//
	// For example, a simple policy spec could be defined as:
	//
	//    policySpec:
	//      validationFailureAction: Audit
	//      background: true
	//      rules:
	//      - name: check-for-labels
	//        match:
	//          any:
	//          - resources:
	//              kinds:
	//              - Pod
	//        validate:
	//          message: "The label `app.kubernetes.io/name` is required."
	//          pattern:
	//            metadata:
	//              labels:
	//                app.kubernetes.io/name: "?*"
	//
	// There are also further examples of Kyverno policies in the
	// [Kyverno Policies Examples](https://kyverno.io/policies/).
	PolicySpec runtime.RawExtension `json:"policySpec"`
}

// PolicyTemplateTarget allows specifying label selectors for Projects and Clusters.
type PolicyTemplateTarget struct {
	// ProjectSelector filters KKP Projects based on their labels.
	//
	// +optional
	ProjectSelector *metav1.LabelSelector `json:"projectSelector,omitempty"`

	// ClusterSelector filters individual KKP Cluster resources based on their labels.
	//
	// +optional
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`
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
