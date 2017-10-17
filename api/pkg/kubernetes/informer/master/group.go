package master

import (
	"time"

	crdClient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	crdinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/master/informers/externalversions/kubermatic/v1"
)

// Group is a group of informers needed for kubermatic
type Group struct {
	SSHKeyInformer  v1.UserSSHKeyInformer
	ClusterInformer v1.ClusterInformer
}

// New returns a instance of Group
func New(crdClient crdClient.Interface) *Group {
	g := Group{}
	crdInformers := crdinformers.NewSharedInformerFactory(crdClient, 5*time.Minute)
	g.SSHKeyInformer = crdInformers.Kubermatic().V1().UserSSHKeies()
	g.ClusterInformer = crdInformers.Kubermatic().V1().Clusters()

	return &g
}

// HasSynced tells if the all informers of the group have synced
func (g *Group) HasSynced() bool {
	return g.SSHKeyInformer.Informer().HasSynced() &&
		g.ClusterInformer.Informer().HasSynced()

}

// Run starts all informers of the group
func (g *Group) Run(stopCh <-chan struct{}) {
	go g.SSHKeyInformer.Informer().Run(stopCh)
	go g.ClusterInformer.Informer().Run(stopCh)
}
