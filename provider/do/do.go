package do

import (
	"github.com/digitalocean/godo"
	cloud "github.com/kubermatic/api/provider"
	"golang.org/x/oauth2"
)

var (
	_ oauth2.TokenSource    = (*provider)(nil)
	_ cloud.ClusterProvider = (*provider)(nil)
)

type provider struct {
	accessToken string
}

func NewProvider(accessToken string) *provider {
	return &provider{accessToken}
}

func (p *provider) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: p.accessToken}, nil
}

func (p *provider) Clusters() ([]cloud.Cluster, error) {
	oc := oauth2.NewClient(oauth2.NoContext, p)
	_ = godo.NewClient(oc)
	return nil, nil
}

func (p *provider) NewCluster(s cloud.ClusterSpec) (cloud.Cluster, error) {
	return nil, nil
}
