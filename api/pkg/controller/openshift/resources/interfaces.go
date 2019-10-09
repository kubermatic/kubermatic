package resources

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Openshift data contains all data required for Openshift control plane components
// It should be as small as possible
type openshiftData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetPodTemplateLabelsWithContext(context.Context, string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	GetApiserverExternalNodePort(context.Context) (int32, error)
	NodePortRange(context.Context) string
	ClusterIPByServiceName(name string) (string, error)
	ImageRegistry(string) string
	NodeAccessNetwork() string
	GetClusterRef() metav1.OwnerReference
	GetRootCA() (*triple.KeyPair, error)
	GetRootCAWithContext(context.Context) (*triple.KeyPair, error)
	DC() *kubermaticv1.Datacenter
	HasEtcdOperatorService() (bool, error)
	EtcdDiskSize() resource.Quantity
	NodeLocalDNSCacheEnabled() bool
	KubermaticAPIImage() string
	DNATControllerImage() string
	GetOauthExternalNodePort() (int32, error)
	Client() (ctrlruntimeclient.Client, error)
	ExternalURL() string
	Seed() *kubermaticv1.Seed
}
