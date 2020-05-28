package kubernetes

import (
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

// kubernetesImpersonationClient gives kubernetes client set that uses user impersonation
type kubernetesImpersonationClient func(impCfg restclient.ImpersonationConfig) (kubernetes.Interface, error)

// kubermaticImpersonationClient gives kubermatic client set that uses user impersonation
type kubermaticImpersonationClient func(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error)

// impersonationClient gives runtime controller client that uses user impersonation
type impersonationClient func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}

// createKubermaticImpersonationClientWrapperFromUserInfo is a helper method that spits back kubermatic client that uses user impersonation
func createKubermaticImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient kubermaticImpersonationClient) (kubermaticclientv1.KubermaticV1Interface, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}

// createKubernetesImpersonationClientWrapperFromUserInfo is a helper method that spits back kubernetes client that uses user impersonation
func createKubernetesImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient kubernetesImpersonationClient) (kubernetes.Interface, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}

// createImpersonationClientWrapperFromUserInfo is a helper method that spits back controller runtime client that uses user impersonation
func createImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient impersonationClient) (ctrlruntimeclient.Client, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}

// NewKubermaticImpersonationClient creates a new default impersonation client
// that knows how to create KubermaticV1Interface client for a impersonated user
//
// Note:
// It is usually not desirable to create many RESTClient thus in the future we might
// consider storing RESTClients in a pool for the given group name
func NewKubermaticImpersonationClient(cfg *restclient.Config) *DefaultKubermaticImpersonationClient {
	return &DefaultKubermaticImpersonationClient{cfg}
}

// DefaultKubermaticImpersonationClient knows how to create impersonated client set
type DefaultKubermaticImpersonationClient struct {
	cfg *restclient.Config
}

// CreateImpersonatedKubermaticClientSet actually creates impersonated kubermatic client set for the given user.
func (d *DefaultKubermaticImpersonationClient) CreateImpersonatedKubermaticClientSet(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
	config := *d.cfg
	config.Impersonate = impCfg
	return kubermaticclientv1.NewForConfig(&config)
}

// NewKubernetesImpersonationClient creates a new default impersonation client
// that knows how to create kubernetes Interface client for a impersonated user
func NewKubernetesImpersonationClient(cfg *restclient.Config) *DefaultKubernetesImpersonationClient {
	return &DefaultKubernetesImpersonationClient{cfg}
}

// DefaultKubermaticImpersonationClient knows how to create impersonated client set
type DefaultKubernetesImpersonationClient struct {
	cfg *restclient.Config
}

// CreateImpersonatedKubernetesClientSet actually creates impersonated kubernetes client set for the given user.
func (d *DefaultKubernetesImpersonationClient) CreateImpersonatedKubernetesClientSet(impCfg restclient.ImpersonationConfig) (kubernetes.Interface, error) {
	config := *d.cfg
	config.Impersonate = impCfg
	return kubernetes.NewForConfig(&config)
}

// NewImpersonationClient creates a new default impersonation client
// that knows how to create Interface client for a impersonated user
func NewImpersonationClient(cfg *restclient.Config, restMapper meta.RESTMapper) *DefaultImpersonationClient {
	return &DefaultImpersonationClient{
		cfg:        cfg,
		restMapper: restMapper,
	}
}

// DefaultImpersonationClient knows how to create impersonated client set
type DefaultImpersonationClient struct {
	cfg        *restclient.Config
	restMapper meta.RESTMapper
}

// CreateImpersonatedClient actually creates impersonated client set for the given user.
func (d *DefaultImpersonationClient) CreateImpersonatedClient(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
	config := *d.cfg
	config.Impersonate = impCfg

	return ctrlruntimeclient.New(&config, ctrlruntimeclient.Options{Mapper: d.restMapper})
}
