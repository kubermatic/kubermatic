package client

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

// New returns a new instance of the client connection provider
func New(secretLister corev1lister.SecretLister) *Provider {
	return &Provider{secretLister: secretLister}
}

// Provider offers functions to interact with a user cluster
type Provider struct {
	secretLister corev1lister.SecretLister
}

// GetAdminKubeconfig returns the admin kubeconfig for the given cluster
func (p *Provider) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	//Load the admin kubeconfig secret
	s, err := p.secretLister.Secrets(c.Status.NamespaceName).Get(resources.AdminKubeconfigSecretName)
	if err != nil {
		return nil, err
	}
	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}
	return d, nil
}

// GetClientConfig returns the client config used for initiating a connection for the given cluster
func (p *Provider) GetClientConfig(c *kubermaticv1.Cluster) (*restclient.Config, error) {
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
	return iconfig.ClientConfig()
}

// GetClient returns a kubernetes client to interact with the given cluster
func (p *Provider) GetClient(c *kubermaticv1.Cluster) (kubernetes.Interface, error) {
	config, err := p.GetClientConfig(c)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetMachineClient returns a client to interact with machine resources for the given cluster
func (p *Provider) GetMachineClient(c *kubermaticv1.Cluster) (clusterv1alpha1clientset.Interface, error) {
	config, err := p.GetClientConfig(c)
	if err != nil {
		return nil, err
	}
	client, err := clusterv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetApiextensionsClient returns a client to interact with apiextension resources for the given cluster
func (p *Provider) GetApiextensionsClient(c *kubermaticv1.Cluster) (apiextensionsclientset.Interface, error) {
	config, err := p.GetClientConfig(c)
	if err != nil {
		return nil, err
	}
	return apiextensionsclientset.NewForConfig(config)
}
