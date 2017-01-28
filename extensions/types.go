package extensions

import (
	"time"

	"k8s.io/client-go/pkg/api/meta"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apimachinery/announced"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
)

const (
	// GroupName is the group for all our extension
	GroupName string = "kubermatic.io"
	// Version is the version of our extensions
	Version string = "v1"
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
		&SSHKey{},
		&apiv1.ListOptions{},
		&apiv1.DeleteOptions{},
	)
	return nil
}

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  GroupName,
			VersionPreferenceOrder:     []string{SchemeGroupVersion.Version},
			AddInternalObjectsToScheme: SchemeBuilder.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			SchemeGroupVersion.Version: SchemeBuilder.AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
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
func (e *ClusterAddon) GetObjectMeta() meta.Object {
	return &e.Metadata
}

//GetObjectKind returns the object typemeta information
func (el *ClusterAddonList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

//GetListMeta returns the list object metadata
func (el *ClusterAddonList) GetListMeta() metav1.List {
	return &el.Metadata
}

// SSHKey specifies a users SSHKey
type SSHKey struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        apiv1.ObjectMeta `json:"metadata"`

	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

//GetObjectKind returns the object typemeta information
func (sk *SSHKey) GetObjectKind() schema.ObjectKind {
	return &sk.TypeMeta
}

//GetListMeta returns the list object metadata
func (sk *SSHKey) GetListMeta() metav1.List {
	return &sk.Metadata
}
