package client

import (
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func New(secretLister corev1lister.SecretLister) *Provider {
	return &Provider{secretLister: secretLister}
}

type Provider struct {
	secretLister corev1lister.SecretLister
}

func (p *Provider) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	//Load the admin kubeconfig secret
	s, err := p.secretLister.Secrets(c.Status.NamespaceName).Get(resources.AdminKubeconfigSecretName)
	if err != nil {
		return nil, err
	}
	return s.Data["admin-kubeconfig"], nil
}

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
		c.Name,
		&clientcmd.ConfigOverrides{},
		nil,
	)
	return iconfig.ClientConfig()
}

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

func (p *Provider) GetMachineClient(c *kubermaticv1.Cluster) (machineclientset.Interface, error) {
	config, err := p.GetClientConfig(c)
	if err != nil {
		return nil, err
	}
	client, err := machineclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
