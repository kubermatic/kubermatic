package kubernetes

import (
	"errors"
	"log"
	"sync"

	"fmt"
	"strings"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

var _ provider.KubernetesProvider = (*seedProvider)(nil)

type seedProvider struct {
	mu    sync.Mutex
	cps   map[string]provider.CloudProvider
	seeds map[string]*api.Cluster
}

// NewSeedProvider creates a new seed provider object
func NewSeedProvider(
	dcs map[string]provider.DatacenterMeta,
	cps map[string]provider.CloudProvider,
	cfgs map[string]client.Config,
	secrets *Secrets,
) provider.KubernetesProvider {
	seeds := map[string]*api.Cluster{}
	for dcName, cfg := range cfgs {
		c := api.Cluster{
			Metadata: api.Metadata{
				Name: dcName,
				UID:  dcName,
			},
			Spec: api.ClusterSpec{
				HumanReadableName: dcName,
				Cloud: &api.CloudSpec{
					DC: dcName,
				},
			},
			Address: &api.ClusterAddress{
				URL:     cfg.Host,
				Token:   cfg.BearerToken,
				EtcdURL: strings.Replace(cfg.Host, "://", "://etcd-", 1),
			},
			Status: api.ClusterStatus{},
		}

		dc, found := dcs[dcName]
		if !found {
			log.Fatal(fmt.Errorf("cannot find kubeconfig ctx %q as datacenter", dcName))
		}
		p, err := provider.DatacenterCloudProviderName(&dc.Spec)
		if err != nil {
			log.Fatal(err)
		}
		switch p {
		case provider.BringYourOwnCloudProvider:
			c.Spec.Cloud.BringYourOwn = &api.BringYourOwnCloudSpec{
				PrivateIntf: dc.Spec.BringYourOwn.Seed.PrivateIntf,
			}
		case provider.DigitaloceanCloudProvider:
			token, found := secrets.Tokens[dcName]
			if !found {
				log.Fatal("cannot find dc %q in secret tokens", dcName)
			}
			c.Spec.Cloud.Digitalocean = &api.DigitaloceanCloudSpec{
				Token:   token,
				SSHKeys: dc.Spec.Digitalocean.Seed.SSHKeys,
			}
		default:
			log.Fatalf("unsupported cloud provider %q for seed dc %q", p, dcName)
		}

		seeds[dcName] = &c
	}

	return &seedProvider{
		cps:   cps,
		seeds: seeds,
	}
}

func (p *seedProvider) NewCluster(user, cluster string, spec *api.ClusterSpec) (*api.Cluster, error) {
	return nil, errors.New("not implemented")
}

func (p *seedProvider) Cluster(user, cluster string) (*api.Cluster, error) {
	if user != "seeds" {
		return nil, kerrors.NewNotFound("cluster", cluster)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	c, found := p.seeds[cluster]
	if !found {
		return nil, kerrors.NewNotFound("cluster", cluster)
	}
	return c, nil
}

func (p *seedProvider) SetCloud(user, cluster string, cloud *api.CloudSpec) (*api.Cluster, error) {
	return nil, errors.New("not implemented")
}

func (p *seedProvider) Clusters(user string) ([]*api.Cluster, error) {
	if user != "seeds" {
		return []*api.Cluster{}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cs := make([]*api.Cluster, 0, len(p.seeds))
	for _, c := range p.seeds {
		cs = append(cs, c)
	}

	return cs, nil
}

func (p *seedProvider) DeleteCluster(user, cluster string) error {
	return errors.New("not implemented")
}
