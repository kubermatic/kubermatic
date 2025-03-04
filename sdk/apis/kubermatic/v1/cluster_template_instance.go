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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	// ClusterTemplateInstanceResourceName represents "Resource" defined in Kubernetes.
	ClusterTemplateInstanceResourceName = "clustertemplateinstances"

	// ClusterTemplateInstanceKindName represents "Kind" defined in Kubernetes.
	ClusterTemplateInstanceKindName = "ClusterTemplateInstance"
)

// ClusterTemplateInstanceOwnerAnnotationKey represents the user cluster owner.
const ClusterTemplateInstanceOwnerAnnotationKey = "owner"

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.projectID",name="ProjectID",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.clusterTemplateID",name="ClusterTemplateID",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.replicas",name="Replicas",type="integer"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// ClusterTemplateInstance is the object representing a cluster template instance.
type ClusterTemplateInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec specifies the data for cluster instances.
	Spec ClusterTemplateInstanceSpec `json:"spec,omitempty"`
}

// ClusterTemplateInstanceSpec specifies the data for cluster instances.
type ClusterTemplateInstanceSpec struct {
	ProjectID           string `json:"projectID"`
	ClusterTemplateID   string `json:"clusterTemplateID"`
	ClusterTemplateName string `json:"clusterTemplateName"`
	Replicas            int64  `json:"replicas"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ClusterTemplateInstanceList specifies a list of cluster template instances.
type ClusterTemplateInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of ClusterTemplateInstance objects.
	Items []ClusterTemplateInstance `json:"items"`
}
