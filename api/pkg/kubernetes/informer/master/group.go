package master

import (
	"time"

	crdClient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	crdinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
)

// Group is a group of informers needed for kubermatic
type Group struct {
	SSHKeyInformer v1.UserSSHKeyInformer
}

// New returns a instance of Group
func New(crdClient crdClient.Interface) *Group {
	g := Group{}
	crdInformers := crdinformers.NewSharedInformerFactory(crdClient, 5*time.Minute)
	g.SSHKeyInformer = crdInformers.Kubermatic().V1().UserSSHKeies()

	return &g
}

// HasSynced tells if the all informers of the group have synced
func (g *Group) HasSynced() bool {
	return g.SSHKeyInformer.Informer().HasSynced()

}

// Run starts all informers of the group
func (g *Group) Run(stopCh <-chan struct{}) {
	go g.SSHKeyInformer.Informer().Run(stopCh)
}
