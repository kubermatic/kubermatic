package resources

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

type Data struct {
	ImageRepository string

	Cluster    *kubermaticv1.Cluster
	Version    *apiv1.MasterVersion
	Datacenter *provider.DatacenterMeta

	SecretLister    corev1lister.SecretLister
	ConfigMapLister corev1lister.ConfigMapLister
	ServiceLister   corev1lister.ServiceLister
}

func (d *Data) GetClusterRef() metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(d.Cluster, gv.WithKind("Cluster"))
}

// Int32 returns a pointer to of the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// Int64 returns a pointer to of the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Bool returns a pointer to of the bool value passed in.
func Bool(v bool) *bool {
	return &v
}
