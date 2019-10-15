package client

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserClusterConnectionProvider describes the interface available for accessing
// resources inside the user cluster
type UserClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster, ...ConfigOption) (ctrlruntimeclient.Client, error)
	GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
	GetViewerKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
	RevokeViewerKubeconfig(c *kubermaticv1.Cluster) error
}

// NewInternal returns a new instance of the client connection provider that
// only works from within the seed cluster but has the advantage that it doesn't leave
// the seed clusters network
func NewInternal(seedClient ctrlruntimeclient.Client) (UserClusterConnectionProvider, error) {
	return &provider{
		seedClient:         seedClient,
		useExternalAddress: false,
		restMapperCache:    restmapper.New(),
	}, nil
}

// NewExternal returns a new instance of the client connection provider
// that uses the external cluster address and hence works from everywhere.
// Use NewInternal if possible
func NewExternal(seedClient ctrlruntimeclient.Client) (UserClusterConnectionProvider, error) {
	return &provider{
		seedClient:         seedClient,
		useExternalAddress: true,
		restMapperCache:    restmapper.New(),
	}, nil
}

type provider struct {
	seedClient         ctrlruntimeclient.Client
	useExternalAddress bool

	// We keep the existing cluster mappings to avoid the discovery on each call to the API server
	restMapperCache *restmapper.Cache
}

// GetAdminKubeconfig returns the admin kubeconfig for the given cluster
func (p *provider) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	s := &corev1.Secret{}
	var err error
	if p.useExternalAddress {
		// Load the admin kubeconfig secret, it uses the external apiserver address
		err = p.seedClient.Get(context.Background(), types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.AdminKubeconfigSecretName}, s)
	} else {
		// Load the internal admin kubeconfig secret
		err = p.seedClient.Get(context.Background(), types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.InternalUserClusterAdminKubeconfigSecretName}, s)
	}
	if err != nil {
		return nil, err
	}
	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}
	return d, nil
}

// GetViewerKubeconfig returns the viewer kubeconfig for the given cluster
func (p *provider) GetViewerKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	s := &corev1.Secret{}

	if err := p.seedClient.Get(context.Background(), types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.ViewerKubeconfigSecretName}, s); err != nil {
		return nil, err
	}

	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}
	return d, nil
}

// RevokeViewerKubeconfig deletes viewer token to deploy new one and regenerate viewer-kubeconfig
func (p *provider) RevokeViewerKubeconfig(c *kubermaticv1.Cluster) error {
	s := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      resources.ViewerTokenSecretName,
			Namespace: c.Status.NamespaceName,
		},
	}

	if err := p.seedClient.Delete(context.Background(), s); err != nil {
		return err
	}
	return nil
}

// ConfigOption defines a function that applies additional configuration to restclient.Config in a generic way.
type ConfigOption func(*restclient.Config) *restclient.Config

// GetClientConfig returns the client config used for initiating a connection for the given cluster
func (p *provider) GetClientConfig(c *kubermaticv1.Cluster, options ...ConfigOption) (*restclient.Config, error) {
	b, err := p.GetAdminKubeconfig(c)
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.Load(b)
	if err != nil {
		return nil, err
	}

	iconfig := clientcmd.NewNonInteractiveClientConfig(
		*cfg,
		resources.KubeconfigDefaultContextKey,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	clientConfig, err := iconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Avoid blocking of the controller by increasing the QPS for user cluster interaction
	clientConfig.QPS = 20
	clientConfig.Burst = 50

	// apply all options
	for _, opt := range options {
		clientConfig = opt(clientConfig)
	}

	return clientConfig, err
}

// GetClient returns a dynamic client
func (p *provider) GetClient(c *kubermaticv1.Cluster, options ...ConfigOption) (ctrlruntimeclient.Client, error) {
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}

	return p.restMapperCache.Client(config)
}
