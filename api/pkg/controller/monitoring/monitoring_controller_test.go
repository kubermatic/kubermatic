package monitoring

import (
	"log"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

const (
	TestClusterName = "fqpcvnc6v"
	TestDC          = "regular-do1"
)

func newTestController(kubeObjects []runtime.Object, kubermaticObjects []runtime.Object) *Controller {
	dcs := buildDatacenterMeta()

	kubeClient := kubefake.NewSimpleClientset(kubeObjects...)
	kubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjects...)

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute*5)
	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, time.Minute*5)

	controller, err := New(
		kubeClient,
		client.New(kubeInformerFactory.Core().V1().Secrets().Lister()),
		TestDC,
		dcs,
		"",
		"",
		"192.0.2.0/24",
		"5Gi",
		"",
		false,
		false,
		"",
		[]byte{},

		kubermaticInformerFactory.Kubermatic().V1().Clusters(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		kubeInformerFactory.Core().V1().ConfigMaps(),
		kubeInformerFactory.Rbac().V1().Roles(),
		kubeInformerFactory.Rbac().V1().RoleBindings(),
		kubeInformerFactory.Core().V1().Services(),
		kubeInformerFactory.Apps().V1().StatefulSets(),
		kubeInformerFactory.Rbac().V1().ClusterRoleBindings(),
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Core().V1().Secrets(),
	)
	if err != nil {
		log.Fatal(err)
	}

	kubeInformerFactory.Start(wait.NeverStop)
	kubermaticInformerFactory.Start(wait.NeverStop)

	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticInformerFactory.WaitForCacheSync(wait.NeverStop)

	return controller
}

func buildDatacenterMeta() map[string]provider.DatacenterMeta {
	seedAlias := "alias-europe-west3-c"
	return map[string]provider.DatacenterMeta{
		"us-central1": {
			Location: "us-central",
			Country:  "US",
			Private:  false,
			IsSeed:   true,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"us-central1-byo": {
			Location: "us-central",
			Country:  "US",
			Private:  false,
			Seed:     "us-central1",
			Spec: provider.DatacenterSpec{
				BringYourOwn: &provider.BringYourOwnSpec{},
			},
		},
		"private-do1": {
			Location: "US ",
			Seed:     "us-central1",
			Country:  "NL",
			Private:  true,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"regular-do1": {
			Location: "Amsterdam",
			Seed:     "us-central1",
			Country:  "NL",
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"dns-override-do2": {
			Location:         "Amsterdam",
			Seed:             "us-central1",
			Country:          "NL",
			SeedDNSOverwrite: &seedAlias,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams3",
				},
			},
		},
	}
}
