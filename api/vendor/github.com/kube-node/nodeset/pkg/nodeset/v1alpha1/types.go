/*
Copyright 2017 The Kubernetes Authors.

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
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type NodeClassResourceType string

const (
	NodeClassResourceFile      NodeClassResourceType = "File"
	NodeClassResourceReference NodeClassResourceType = "Reference"

	NodeClassResourcePlural = "nodeclasses"
	NodeSetResourcePlural   = "nodesets"

	NodeClassContentAnnotationKey     = "nodeset.k8s.io/nodeclass-content"
	NodeClassContentHashAnnotationKey = "nodeset.k8s.io/nodeclass-content-hash"
	NodeClassNameAnnotationKey        = "nodeset.k8s.io/node-class"

	NodeSetGenerationAnnotationKey = "nodeset.k8s.io/nodeset-generation"

	NodeSetNameLabelKey = "nodeset.k8s.io/nodeset"
	ControllerLabelKey  = "nodeset.k8s.io/controller"
)

type NodeClassResource struct {
	// Type is the type of the resource
	Type NodeClassResourceType `json:"type,omitempty"`

	// Name is the name of the resource
	// It can be used as a key for overriding
	Name string `json:"name,omitempty"`

	// For File type
	// Path is the full path of the file including filename
	// +optional
	Path string `json:"path,omitempty"`
	// Owner is the owner of the file
	// +optional
	Owner string `json:"owner,omitempty"`
	// Permission is the permission of the file
	// +optional
	Permission string `json:"permission,omitempty"`
	// Template is the template or content of the file
	// +optional
	Template string `json:"template,omitempty"`

	// For Reference type
	// Reference points to an Object in registry that the controller will use when
	// provisioning the node
	// Ex. a secret ref to be used as ssh credentials.
	// +optional
	Reference *corev1.ObjectReference `json:"reference,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeClass describes the parameters for a class of nodes that can be
// provisioned by the specific controller
type NodeClass struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// NodeController specifes which node controller should provision and
	// control this class of nodes
	NodeController string `json:"nodeController,omitempty"`

	// NodeLabels specifies a list of labels each node should get.
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// Config is free form data containing overridable parameters that the controller
	// will use
	// +optional
	Config runtime.RawExtension `json:"config,omitempty"`

	// Resources is a list of file-like resources that the controller can use
	// +optional
	Resources []NodeClassResource `json:"resources,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeClassList is a collection of node classes.
type NodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of NodeClasses
	Items []NodeClass `json:"items"`
}

// NodeSetSpec is the spec of a node set.
type NodeSetSpec struct {
	// NodeSelector is a selector of node labels which a node must have to be in the NodeSet.
	// More info: http://kubernetes.io/docs/user-guide/node-selection/README
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// NodeClass specifies which NodeClass the nodes created by this NodeSet should
	// belong to
	NodeClass string `json:"nodeClass,omitempty"`

	// NodeSetController specifies a NodeSet controller to be in charge of this NodeSet
	// If not specified, the default one that comes with the NodeController in NodeClass
	// will be used.
	// +optional
	NodeSetController string `json:"nodeSetController,omitempty"`

	// Replicas is the number of desired replicas.
	Replicas int32 `json:"replicas,omitempty"`

	// The maximum number of Nodes that can be unavailable during update
	// This can be an absolute number (ex. 3) or a percentage of total replicas
	// The absolute number will be calculated from percentage by rounding down.
	// This can't be 0 if MaxSurge is 0.
	// By default, a value of 0 is used
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// The maximum number of Nodes that can be created above the desired replicas.
	// This can be an absolute number (ex. 3) or a percentage of total replicas
	// The absolute number will be calculated from percentage by rounding up.
	// This can't be 0 if MaxUnavailble is 0.
	// A default of 1 is used if not set.
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`

	// Config contains free form structural data that overrides those in NodeClass.
	// +optional
	Config runtime.RawExtension `json:"config,omitempty"`
}

type NodeSetConditionType string

const (
	// This is added when the NodeSet controllers failed to create or delete a node during
	// reconciliation.
	NodeSetReplicaFailure NodeSetConditionType = "ReplicaFailure"
)

// NodeSetCondition describes the state of a NodeSet at a certain point.
type NodeSetCondition struct {
	// Type of NodeSet condition.
	Type NodeSetConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`

	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// NodeSetStatus is the status of a nodeset
type NodeSetStatus struct {
	// Replicas is the number of actual replicas.
	Replicas int32 `json:"replicas"`

	// The number of replicas with running status for this NodeSet.
	// +optional
	RunningReplicas int32 `json:"runningReplicas,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Represents the latest available observations of the NodeSet's current state.
	// +optional
	Conditions []NodeSetCondition `json:"conditions,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeSet is a set of nodes of the same class.
type NodeSet struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of desired behavior of the NodeSet.
	// +optional
	Spec NodeSetSpec `json:"spec,omitempty"`

	// Most recently observed status of the NodeSet.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// +optional
	Status NodeSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeSetList is a collection of nodesets.
type NodeSetList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of NodeSets
	Items []NodeSet `json:"items"`
}
