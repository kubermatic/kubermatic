package cluster

import (
	"fmt"
	"log"

	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func newTestController() (*fake.Clientset, *clusterController) {
	dcs, err := provider.DatacentersMeta("./fixtures/datacenters.yaml")
	if err != nil {
		log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", "./fixtures/datacenters.yaml", err))
	}

	// create CloudProviders
	cps := cloud.Providers(dcs)

	tprClient, err := extensions.WrapClientsetWithExtensions(&rest.Config{})
	if err != nil {
		log.Fatal(err)
	}

	clientSet := fake.NewSimpleClientset()
	cc, err := NewController("", clientSet, tprClient, cps, "", "localhost", true, "", "./../../addon-charts/")
	if err != nil {
		log.Fatal(err)
	}

	return clientSet, cc.(*clusterController)
}
