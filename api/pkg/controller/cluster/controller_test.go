package cluster

import (
	"log"

	"github.com/kubermatic/kubermatic/api"
	mastercrdfake "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned/fake"
	seedcrdfake "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned/fake"
	masterinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/master"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const TestDC string = "testdc"
const TestExternalURL string = "localhost"
const TestExternalPort int = 8443

type fake struct {
	controller   *controller
	kubeclient   *kubefake.Clientset
	seedclient   *seedcrdfake.Clientset
	masterclient *mastercrdfake.Clientset
}

func newTestController(
	kubeObjects []runtime.Object,
	seedCrdObjects []runtime.Object,
	masterCrdObjects []runtime.Object,
) *fake {
	// create datacenters
	dcs := buildDatacenterMeta()
	// create CloudProviders
	cps := cloud.Providers(dcs)

	versions := buildMasterVerionsMap()
	updates := buildMasterUpdates()

	kubeClient := kubefake.NewSimpleClientset(kubeObjects...)

	seedCrdClient := seedcrdfake.NewSimpleClientset(seedCrdObjects...)
	masterCrdClient := mastercrdfake.NewSimpleClientset(masterCrdObjects...)

	seedInformerGroup := seedinformer.New(kubeClient, seedCrdClient)
	masterInformerGroup := masterinformer.New(masterCrdClient)

	cc, err := NewController(
		TestDC,
		kubeClient,
		seedCrdClient,
		masterCrdClient,
		cps,
		versions,
		updates,
		"./../../master-resources/",
		TestExternalURL,
		"user1",
		TestExternalPort,
		dcs,
		masterInformerGroup,
		seedInformerGroup,
	)
	if err != nil {
		log.Fatal(err)
	}

	masterInformerGroup.Run(wait.NeverStop)
	seedInformerGroup.Run(wait.NeverStop)

	cache.WaitForCacheSync(wait.NeverStop, masterInformerGroup.HasSynced, seedInformerGroup.HasSynced)

	return &fake{
		controller:   cc.(*controller),
		kubeclient:   kubeClient,
		masterclient: masterCrdClient,
		seedclient:   seedCrdClient,
	}
}

func buildMasterVerionsMap() map[string]*api.MasterVersion {
	return map[string]*api.MasterVersion{
		"1.5.2": {
			Name:                       "1.5.2",
			ID:                         "1.5.2",
			Default:                    false,
			AllowedNodeVersions:        []string{"1.3.0"},
			EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
			EtcdClusterYaml:            "etcd-cluster.yaml",
			ApiserverDeploymentYaml:    "apiserver-dep.yaml",
			ControllerDeploymentYaml:   "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:    "scheduler-dep.yaml",
			Values: map[string]string{
				"k8s-version":  "v1.5.2",
				"etcd-version": "3.0.14-kubeadm",
			},
		},
		"1.5.3": {
			Name:                       "1.5.3",
			ID:                         "1.5.3",
			Default:                    true,
			AllowedNodeVersions:        []string{"1.3.0"},
			EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
			EtcdClusterYaml:            "etcd-cluster.yaml",
			ApiserverDeploymentYaml:    "apiserver-dep.yaml",
			ControllerDeploymentYaml:   "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:    "scheduler-dep.yaml",
			Values: map[string]string{
				"k8s-version":  "v1.5.3",
				"etcd-version": "3.0.14-kubeadm",
			},
		},
		"v1.6.0-rc.1": {
			Name:                       "v1.6.0-rc.1",
			ID:                         "v1.6.0-rc.1",
			Default:                    false,
			AllowedNodeVersions:        []string{"1.4.0"},
			EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
			EtcdClusterYaml:            "etcd-cluster.yaml",
			ApiserverDeploymentYaml:    "apiserver-dep.yaml",
			ControllerDeploymentYaml:   "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:    "scheduler-dep.yaml",
			Values: map[string]string{
				"k8s-version":  "v1.6.0-rc.1",
				"etcd-version": "3.0.14-kubeadm",
			},
		},
	}
}

func buildMasterUpdates() []api.MasterUpdate {
	return []api.MasterUpdate{
		{
			From:            "1.5.*",
			To:              "1.5.2",
			Automatic:       true,
			RollbackAllowed: true,
			Enabled:         true,
			Visible:         true,
			Promote:         true,
		},
		{
			From:            "1.4.6",
			To:              "1.5.1",
			Automatic:       true,
			RollbackAllowed: true,
			Enabled:         true,
			Visible:         true,
			Promote:         true,
		},
		{
			From:            "1.4.*",
			To:              "1.4.6",
			Automatic:       true,
			RollbackAllowed: true,
			Enabled:         true,
			Visible:         true,
			Promote:         true,
		},
	}
}

func buildDatacenterMeta() map[string]provider.DatacenterMeta {
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
	}
}
