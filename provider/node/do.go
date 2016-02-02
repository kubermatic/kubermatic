package do

/*
import (
	"github.com/digitalocean/godo"
	"github.com/kubermatic/api/provider"
	"golang.org/x/oauth2"
)

var (
	_ oauth2.TokenSource       = (*clusterProvider)(nil)
	_ provider.ClusterProvider = (*clusterProvider)(nil)
)

type clusterProvider struct {
	accessToken string
}

func NewProvider(accessToken string) *clusterProvider {
	return &clusterProvider{accessToken}
}

func (p *clusterProvider) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: p.accessToken}, nil
}

func (p *clusterProvider) Clusters() ([]provider.Cluster, error) {
	oc := oauth2.NewClient(oauth2.NoContext, p)
	_ = godo.NewClient(oc)
	return nil, nil
}

func (p *clusterProvider) NewCluster(s provider.ClusterSpec) (provider.Cluster, error) {
	return nil, nil
}
*/