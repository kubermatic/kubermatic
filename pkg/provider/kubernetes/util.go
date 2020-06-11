package kubernetes

import (
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/api/meta"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

// impersonationClient gives runtime controller client that uses user impersonation
type impersonationClient func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}

// createImpersonationClientWrapperFromUserInfo is a helper method that spits back controller runtime client that uses user impersonation
func createImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient impersonationClient) (ctrlruntimeclient.Client, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
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
