package client

import (
	"context"
	"fmt"

	openshiftuserclusterresources "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/resources/resources/openshift"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewInternal returns a new instance of the client connection provider that
// only works from within the seed cluster but has the advantage that it doesn't leave
// the seed clusters network
func NewInternal(seedClient ctrlruntimeclient.Client) (*Provider, error) {
	return &Provider{
		seedClient:         seedClient,
		useExternalAddress: false,
		restMapperCache:    restmapper.New(),
	}, nil
}

// NewExternal returns a new instance of the client connection provider
// that uses the external cluster address and hence works from everywhere.
// Use NewInternal if possible
func NewExternal(seedClient ctrlruntimeclient.Client) (*Provider, error) {
	return &Provider{
		seedClient:         seedClient,
		useExternalAddress: true,
		restMapperCache:    restmapper.New(),
	}, nil
}

type Provider struct {
	seedClient         ctrlruntimeclient.Client
	useExternalAddress bool

	// We keep the existing cluster mappings to avoid the discovery on each call to the API server
	restMapperCache *restmapper.Cache

	// Can be set for tests
	overrideGetClientFunc func(*kubermaticv1.Cluster, ...ConfigOption) (ctrlruntimeclient.Client, error)
}

// GetAdminKubeconfig returns the admin kubeconfig for the given cluster
func (p *Provider) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	s := &corev1.Secret{}
	if err := p.seedClient.Get(context.Background(), types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.InternalUserClusterAdminKubeconfigSecretName}, s); err != nil {
		return nil, err
	}
	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	if p.useExternalAddress {
		return setExternalAddress(c, d)
	}

	return d, nil
}

// GetViewerKubeconfig returns the viewer kubeconfig for the given cluster
func (p *Provider) GetViewerKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	s := &corev1.Secret{}

	if err := p.seedClient.Get(context.Background(), types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.ViewerKubeconfigSecretName}, s); err != nil {
		return nil, err
	}

	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	if p.useExternalAddress {
		return setExternalAddress(c, d)
	}

	return d, nil
}

func setExternalAddress(c *kubermaticv1.Cluster, config []byte) ([]byte, error) {
	cfg, err := clientcmd.Load(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}
	for _, cluster := range cfg.Clusters {
		cluster.Server = c.Address.URL
	}
	data, err := clientcmd.Write(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %v", err)
	}

	return data, nil
}

// RevokeViewerKubeconfig deletes viewer token to deploy new one and regenerate viewer-kubeconfig
func (p *Provider) RevokeViewerKubeconfig(c *kubermaticv1.Cluster) error {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ViewerTokenSecretName,
			Namespace: c.Status.NamespaceName,
		},
	}

	if err := p.seedClient.Delete(context.Background(), s); err != nil {
		return err
	}
	return nil
}

func (p *Provider) RevokeAdminKubeconfig(c *kubermaticv1.Cluster) error {
	isOpenshift := c.Annotations["kubermatic.io/openshift"] != ""
	ctx := context.Background()
	if !isOpenshift {
		oldCluster := c.DeepCopy()
		c.Address.AdminToken = kuberneteshelper.GenerateToken()
		if err := p.seedClient.Patch(ctx, c, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to patch cluster with new token: %v", err)
		}
		return nil
	}
	userClusterClient, err := p.GetClient(c)
	if err != nil {
		return fmt.Errorf("failed to get usercluster client: %v", err)
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceSystem,
			Name:      openshiftuserclusterresources.TokenOwnerServiceAccountName,
		},
	}
	if err := userClusterClient.Delete(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to remove the token owner: %v", err)
	}
	return nil
}

// ConfigOption defines a function that applies additional configuration to restclient.Config in a generic way.
type ConfigOption func(*restclient.Config) *restclient.Config

// GetClientConfig returns the client config used for initiating a connection for the given cluster
func (p *Provider) GetClientConfig(c *kubermaticv1.Cluster, options ...ConfigOption) (*restclient.Config, error) {
	b, err := p.GetAdminKubeconfig(c)
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.Load(b)
	if err != nil {
		return nil, err
	}

	if p.useExternalAddress {
		for _, cluster := range cfg.Clusters {
			cluster.Server = c.Address.URL
		}
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
func (p *Provider) GetClient(c *kubermaticv1.Cluster, options ...ConfigOption) (ctrlruntimeclient.Client, error) {
	if p.overrideGetClientFunc != nil {
		return p.overrideGetClientFunc(c, options...)
	}
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}

	return p.restMapperCache.Client(config)
}
