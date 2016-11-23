package kubernetes

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/util/rand"
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
	cfgs map[string]restclient.Config,
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
					Network: api.NetworkSpec{
						Flannel: api.FlannelNetworkSpec{
							CIDR: flannelCIDRADefault,
						},
					},
				},
			},
			Address: &api.ClusterAddress{
				URL:     cfg.Host,
				Token:   cfg.BearerToken,
				EtcdURL: strings.TrimRight(cfg.Host, "/") + ":2378",
			},
			Status: api.ClusterStatus{
				LastTransitionTime: time.Now(),
				Phase:              api.RunningClusterStatusPhase,
				ApiserverSSH:       secrets.ApiserverSSH[dcName],
			},
		}

		if ca, found := secrets.RootCAs[dcName]; found {
			c.Status.RootCA.Key = api.NewBytes(ca.Key)
			c.Status.RootCA.Cert = api.NewBytes(ca.Cert)
		}

		dc, found := dcs[dcName]
		if !found {
			log.Fatal(fmt.Errorf("cannot find kubeconfig ctx %q as datacenter", dcName))
		}
		p, err := provider.DatacenterCloudProviderName(&dc.Spec)
		if err != nil {
			log.Fatal(err)
		}

		if dc.Spec.Seed.Network.Flannel.CIDR != "" {
			c.Spec.Cloud.Network.Flannel.CIDR = dc.Spec.Seed.Network.Flannel.CIDR
		}

		switch p {
		case provider.BringYourOwnCloudProvider:
			c.Spec.Cloud.BringYourOwn = &api.BringYourOwnCloudSpec{
				PrivateIntf: dc.Spec.Seed.BringYourOwn.PrivateIntf,
			}
			if c.Status.RootCA.Key != nil && c.Status.RootCA.Cert != nil {
				clientCA, err := c.CreateKeyCert("seed-etcd-client-ca")
				if err != nil {
					log.Fatalf("failed to create a client ca for seed dc %q", dcName)
				}
				c.Spec.Cloud.BringYourOwn.ClientKeyCert = *clientCA
			}
		case provider.DigitaloceanCloudProvider:
			token, found := secrets.Tokens[dcName]
			if !found {
				log.Fatalf("cannot find dc %q in secret tokens", dcName)
			}
			c.Spec.Cloud.Digitalocean = &api.DigitaloceanCloudSpec{
				Token:   token,
				SSHKeys: dc.Spec.Seed.Digitalocean.SSHKeys,
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

func (p *seedProvider) NewCluster(user provider.User, spec *api.ClusterSpec) (*api.Cluster, error) {
	cluster := rand.String(9)

	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		return nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	return nil, errors.New("not implemented")
}

func (p *seedProvider) Cluster(user provider.User, cluster string) (*api.Cluster, error) {
	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		return nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	c, found := p.seeds[cluster]
	if !found {
		return nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
	}
	return c, nil
}

func (p *seedProvider) SetCloud(user provider.User, cluster string, cloud *api.CloudSpec) (*api.Cluster, error) {
	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		return nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	return nil, errors.New("not implemented")
}

func (p *seedProvider) Clusters(user provider.User) ([]*api.Cluster, error) {
	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		return nil, errors.New("forbidden to access clusters")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cs := make([]*api.Cluster, 0, len(p.seeds))
	for _, c := range p.seeds {
		cs = append(cs, c)
	}

	return cs, nil
}

func (p *seedProvider) DeleteCluster(user provider.User, cluster string) error {
	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		return kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	return errors.New("not implemented")
}
