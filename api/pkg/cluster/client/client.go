package client

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	admissionregistrationclientset "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	aggregationclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserClusterConnectionProvider describes the interface available for accessing
// resources inside the user cluster
type UserClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster, ...ConfigOption) (kubernetes.Interface, error)
	GetApiextensionsClient(*kubermaticv1.Cluster, ...ConfigOption) (apiextensionsclientset.Interface, error)
	GetAdmissionRegistrationClient(*kubermaticv1.Cluster, ...ConfigOption) (admissionregistrationclientset.AdmissionregistrationV1beta1Interface, error)
	GetKubeAggregatorClient(*kubermaticv1.Cluster, ...ConfigOption) (aggregationclientset.Interface, error)
	GetDynamicClient(*kubermaticv1.Cluster, ...ConfigOption) (ctrlruntimeclient.Client, error)
	GetAdminKubeconfig(*kubermaticv1.Cluster) ([]byte, error)
}

// NewInternal returns a new instance of the client connection provider that
// only works from within the seed cluster but has the advantage that it doesn't leave
// the seed clusters network
func NewInternal(secretLister corev1lister.SecretLister) (UserClusterConnectionProvider, error) {
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add clusterv1alpha1 to scheme: %v", err)
	}
	return &provider{secretLister: secretLister, useExternalAddress: false}, nil
}

// NewExternal returns a new instance of the client connection provider that
// that uses the external cluster address and hence works from everywhere.
// Use NewInternal if possible
func NewExternal(secretLister corev1lister.SecretLister) (UserClusterConnectionProvider, error) {
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add clusterv1alpha1 to scheme: %v", err)
	}
	return &provider{secretLister: secretLister, useExternalAddress: true}, nil
}

type provider struct {
	secretLister       corev1lister.SecretLister
	useExternalAddress bool
}

// GetAdminKubeconfig returns the admin kubeconfig for the given cluster
func (p *provider) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	var s *corev1.Secret
	var err error
	if p.useExternalAddress {
		// Load the admin kubeconfig secret, it uses the external apiserver address
		s, err = p.secretLister.Secrets(c.Status.NamespaceName).Get(resources.AdminKubeconfigSecretName)
	} else {
		// Load the internal admin kubeconfig secret
		s, err = p.secretLister.Secrets(c.Status.NamespaceName).Get(resources.InternalUserClusterAdminKubeconfigSecretName)
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

// GetClient returns a kubernetes client to interact with the given cluster
func (p *provider) GetClient(c *kubermaticv1.Cluster, options ...ConfigOption) (kubernetes.Interface, error) {
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetApiextensionsClient returns a client to interact with apiextension resources for the given cluster
func (p *provider) GetApiextensionsClient(c *kubermaticv1.Cluster, options ...ConfigOption) (apiextensionsclientset.Interface, error) {
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}
	return apiextensionsclientset.NewForConfig(config)
}

// GetAdmissionRegistrationClient returns a client to interact with admissionregistration resources
func (p *provider) GetAdmissionRegistrationClient(c *kubermaticv1.Cluster, options ...ConfigOption) (admissionregistrationclientset.AdmissionregistrationV1beta1Interface, error) {
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}
	return admissionregistrationclientset.NewForConfig(config)
}

// GetKubeAggregatorClient returns a client to interact with the aggregation API for the given cluster
func (p *provider) GetKubeAggregatorClient(c *kubermaticv1.Cluster, options ...ConfigOption) (aggregationclientset.Interface, error) {
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}
	return aggregationclientset.NewForConfig(config)
}

// GetDynamicClient returns a dynamic client
func (p *provider) GetDynamicClient(c *kubermaticv1.Cluster, options ...ConfigOption) (ctrlruntimeclient.Client, error) {
	config, err := p.GetClientConfig(c, options...)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}
	return dynamicClient, nil
}
