package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (

	// ConstraintResourceName represents "Resource" defined in Kubernetes
	ConstraintResourceName = "constraints"

	// ConstraintKind represents "Kind" defined in Kubernetes
	ConstraintKind = "Constraint"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Constraint struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	ConstraintSpec ConstraintSpec `json:"spec,omitempty"`
}

type ConstraintSpec struct {
	// type of gatekeeper constraint that the constraint applies to
	ConstraintType string `json:"constraintType"`
	Match          `json:"match,omitempty"`
	Parameters     Parameters `json:"parameters,omitempty"`
}

type Match struct {
	Kinds              []Kind               `json:"kinds,omitempty"`
	Scope              string               `json:"scope,omitempty"`
	Namespaces         []string             `json:"namespaces,omitempty"`
	ExcludedNamespaces []string             `json:"excludedNamespaces,omitempty"`
	LabelSelector      metav1.LabelSelector `json:"labelSelector,omitempty"`
	NamespaceSelector  metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type Kind struct {
	Kinds     string `json:"kinds,omitempty"`
	ApiGroups string `json:"apiGroups,omitempty"`
}

type Parameters interface {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConstraintList specifies a list of constraints
type ConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Constraint `json:"items"`
}
