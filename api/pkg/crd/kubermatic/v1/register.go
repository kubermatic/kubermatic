package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	if err := AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to add kubermatic scheme: %v", err))
	}
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// GroupName is the group name use in this package
const GroupName = "kubermatic.k8s.io"
const GroupVersion = "v1"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&UserSSHKey{},
		&UserSSHKeyList{},
		&Cluster{},
		&ClusterList{},
		&User{},
		&UserList{},
		&Project{},
		&ProjectList{},
		&Addon{},
		&AddonList{},
		&UserProjectBinding{},
		&UserProjectBindingList{},
		&Seed{},
		&SeedList{},
		&KubermaticSetting{},
		&KubermaticSettingList{},
		&AddonConfig{},
		&AddonConfigList{},
		&Preset{},
		&PresetList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
