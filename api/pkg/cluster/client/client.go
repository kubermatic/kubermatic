package client

import (
	"context"
	"fmt"
	"sync"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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
}

// NewInternal returns a new instance of the client connection provider that
// only works from within the seed cluster but has the advantage that it doesn't leave
// the seed clusters network
func NewInternal(seedClient ctrlruntimeclient.Client) (UserClusterConnectionProvider, error) {
	return &provider{
		seedClient:         seedClient,
		useExternalAddress: false,
		clusterRESTMapper:  map[string]meta.RESTMapper{},
	}, nil
}

// NewExternal returns a new instance of the client connection provider that
// that uses the external cluster address and hence works from everywhere.
// Use NewInternal if possible
func NewExternal(seedClient ctrlruntimeclient.Client) (UserClusterConnectionProvider, error) {
	return &provider{
		seedClient:         seedClient,
		useExternalAddress: true,
		clusterRESTMapper:  map[string]meta.RESTMapper{},
	}, nil
}

type provider struct {
	seedClient         ctrlruntimeclient.Client
	useExternalAddress bool

	mapperLock sync.Mutex
	// We keep the existing cluster mappings to avoid the discovery on each call to the API server
	clusterRESTMapper map[string]meta.RESTMapper
}

func (p *provider) mapper(c *kubermaticv1.Cluster, config *restclient.Config) (meta.RESTMapper, error) {
	p.mapperLock.Lock()
	defer p.mapperLock.Unlock()

	if mapper, found := p.clusterRESTMapper[c.Name]; found {
		return mapper, nil
	}

	mapper, err := restmapper.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, err
	}
	log.Logger.Infow("mapper created", "cluster", c.Name)
	p.clusterRESTMapper[c.Name] = mapper

	return mapper, nil
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

	mapper, err := p.mapper(c, config)
	if err != nil {
		log.Logger.Errorf("failed to get the REST mapper for the client: %v", err)
		return ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	}

	dynamicClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}
	return dynamicClient, nil
}
