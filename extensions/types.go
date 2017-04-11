package extensions

import (
	"time"

	apitypes "github.com/kubermatic/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	// GroupName is the group for all our extension
	GroupName string = "kubermatic.io"
	// Version is the version of our extensions
	Version string = "v1"
)

const (
	// SSHKeyTPRName is the names of the TPR storing SSH keys
	SSHKeyTPRName string = "usersshkeies"

	// SSHKeyTPRNamespace is the name of the namespace the TPR is created in
	SSHKeyTPRNamespace string = "default"
)

const (
	// NodeTPRName is the names of the TPR storing Nodes
	NodeTPRName string = "clnodes"
)

var (
	// SchemeGroupVersion is the combination of group name and version for the kubernetes client
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}
	// SchemeBuilder provides scheme information about our extensions
	SchemeBuilder = runtime.NewSchemeBuilder(addTypes)
)

func addTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&ClusterAddon{},
		&ClusterAddonList{},
		&ClNode{},
		&ClNodeList{},
		&apiv1.ListOptions{},
	)
	m := map[string]runtime.Object{
		"UserSshKey":     &UserSSHKey{},
		"UserSshKeyList": &UserSSHKeyList{},
	}
	for k, v := range m {
		scheme.AddKnownTypeWithName(
			schema.GroupVersionKind{
				Group:   SchemeGroupVersion.Group,
				Version: SchemeGroupVersion.Version,
				Kind:    k,
			},
			v,
		)
	}

	return nil
}

// AddonPhase is the life cycle phase of a add on.
type AddonPhase string

const (
	// PendingAddonStatusPhase means that the cluster controller hasn't picked the addon up
	PendingAddonStatusPhase AddonPhase = "Pending"

	// InstallingAddonStatusPhase means that the cluster controller has picked the addon up
	InstallingAddonStatusPhase AddonPhase = "Installing"

	// FailedAddonStatusPhase means that the cluster controller failed to install the add on
	FailedAddonStatusPhase AddonPhase = "Failed"

	// RunningAddonStatusPhase means that the add on is up and running
	RunningAddonStatusPhase AddonPhase = "Running"
)

// ClusterAddon specifies a cluster addon
type ClusterAddon struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        apiv1.ObjectMeta `json:"metadata"`
	Name            string           `json:"name"`
	Phase           AddonPhase       `json:"phase"`
	Version         int32
	Deployed        time.Time
	ReleaseName     string
	Attempt         int8
}

// ClusterAddonList specifies a list of cluster addons
type ClusterAddonList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []ClusterAddon `json:"items"`
}

//GetObjectKind returns the object typemeta information
func (e *ClusterAddon) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

//GetObjectMeta returns the object metadata
func (e *ClusterAddon) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

// ClNode contains node information to be saved
type ClNode struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        apiv1.ObjectMeta `json:"metadata"`

	Spec   apitypes.NodeSpec   `json:"spec"`
	Status apitypes.NodeStatus `json:"status,omitempty"`
}

// ClNodeList specifies a list of nodes
type ClNodeList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []ClNode `json:"items"`
}

// GetObjectKind returns the object typemeta information
func (e *ClNode) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

// GetObjectMeta returns the object metadata
func (e *ClNode) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

// GetObjectKind returns the object typemeta information
func (el *ClNodeList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

// GetListMeta returns the list object metadata
func (el *ClNodeList) GetListMeta() metav1.List {
	return &el.Metadata
}

//GetObjectKind returns the object typemeta information
func (el *ClusterAddonList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

//GetListMeta returns the list object metadata
func (el *ClusterAddonList) GetListMeta() metav1.List {
	return &el.Metadata
}

// UserSSHKey specifies a users UserSSHKey
type UserSSHKey struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        apiv1.ObjectMeta `json:"metadata"`

	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

//GetObjectKind returns the object typemeta information
func (sk *UserSSHKey) GetObjectKind() schema.ObjectKind {
	return &sk.TypeMeta
}

//GetListMeta returns the list object metadata
func (sk *UserSSHKey) GetListMeta() metav1.List {
	return &sk.Metadata
}

func (sk *UserSSHKey) addLabel(key string, value string) {
	lbs := sk.Metadata.Labels
	if lbs == nil {
		lbs = map[string]string{}
	}
	lbs[key] = value
	sk.Metadata.SetLabels(lbs)
}

// UserSSHKeyList specifies a users UserSSHKey
type UserSSHKeyList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []UserSSHKey `json:"items"`
}

//GetObjectKind returns the object typemeta information
func (kl *UserSSHKeyList) GetObjectKind() schema.ObjectKind {
	return &kl.TypeMeta
}

//GetListMeta returns the list object metadata
func (kl *UserSSHKeyList) GetListMeta() metav1.List {
	return &kl.Metadata
}
