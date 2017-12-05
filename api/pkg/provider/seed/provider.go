package seed

import (
	"errors"
	"fmt"

	seedcrdclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	// ErrDCNotFound tells that the given datacenter was not found
	ErrDCNotFound = errors.New("datacenter not found")
)

// NewFromConfig returns a new instance of Provider
func NewFromConfig(config *clientcmdapi.Config) (*Provider, error) {
	dcs := map[string]*DatacenterInteractor{}

	for dc := range config.Contexts {
		ctxConfig := clientcmd.NewNonInteractiveClientConfig(
			*config,
			dc,
			&clientcmd.ConfigOverrides{},
			nil,
		)
		clientConfig, err := ctxConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to get clientconfig for context %s: %v", dc, err)
		}

		client := kubernetes.NewForConfigOrDie(clientConfig)
		crdClient := seedcrdclientset.NewForConfigOrDie(clientConfig)
		informerGroup := seed.New(client, crdClient)
		go informerGroup.Run(wait.NeverStop)
		cache.WaitForCacheSync(wait.NeverStop, informerGroup.HasSynced)
		dcs[dc] = &DatacenterInteractor{
			crdClient:     crdClient,
			client:        client,
			informerGroup: informerGroup,
		}
	}

	return &Provider{dcs: dcs}, nil
}

// NewDatacenterIteractor returns a new DatacenterInteractor
func NewDatacenterIteractor(client kubernetes.Interface, crdClient seedcrdclientset.Interface, informerGroup *seed.Group) *DatacenterInteractor {
	return &DatacenterInteractor{
		crdClient:     crdClient,
		client:        client,
		informerGroup: informerGroup,
	}
}

// NewProvider returns a new Provider with the given datacenters
func NewProvider(dcs map[string]*DatacenterInteractor) *Provider {
	return &Provider{dcs: dcs}
}

// DatacenterInteractor groups clients for a seed cluster
type DatacenterInteractor struct {
	client        kubernetes.Interface
	crdClient     seedcrdclientset.Interface
	informerGroup *seed.Group
}

// Provider is a wrapper around a list of datacenters
type Provider struct {
	dcs map[string]*DatacenterInteractor
}

// GetClient returns the default kubernetes client
func (p *Provider) GetClient(dc string) (kubernetes.Interface, error) {
	i, found := p.dcs[dc]
	if !found {
		return nil, ErrDCNotFound
	}
	return i.client, nil
}

// GetCRDClient returns the client to interact with custom resources
func (p *Provider) GetCRDClient(dc string) (seedcrdclientset.Interface, error) {
	i, found := p.dcs[dc]
	if !found {
		return nil, ErrDCNotFound
	}
	return i.crdClient, nil
}

// GetInformerGroup returns the informer group
func (p *Provider) GetInformerGroup(dc string) (*seed.Group, error) {
	i, found := p.dcs[dc]
	if !found {
		return nil, ErrDCNotFound
	}
	return i.informerGroup, nil
}
