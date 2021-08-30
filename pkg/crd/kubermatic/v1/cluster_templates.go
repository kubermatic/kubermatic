/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	UserClusterTemplateScope    = "user"
	ProjectClusterTemplateScope = "project"
	GlobalClusterTemplateScope  = "global"
)

const (
	ClusterTemplateScopeLabelKey             = "scope"
	ClusterTemplateUserAnnotationKey         = "user"
	ClusterTemplateProjectLabelKey           = "project-id"
	ClusterTemplateHumanReadableNameLabelKey = "name"
)

const (
	// ClusterTemplateResourceName represents "Resource" defined in Kubernetes
	ClusterTemplateResourceName = "clustertemplates"

	// ClusterTemplateKindName represents "Kind" defined in Kubernetes
	ClusterTemplateKindName = "ClusterTemplate"
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTemplate is the object representing a cluster template.
type ClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ClusterLabels          map[string]string `json:"clusterLabels,omitempty"`
	InheritedClusterLabels map[string]string `json:"inheritedClusterLabels,omitempty"`
	Credential             string            `json:"credential"`
	UserSSHKeys            []string          `json:"userSshKeys"`
	Spec                   ClusterSpec       `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTemplateList specifies a list of cluster templates
type ClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterTemplate `json:"items"`
}
