package cluster

import (
	"log"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	mastercrdfake "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned/fake"
	seedcrdclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	seedcrdfake "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned/fake"
	masterinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/master"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/seed"
	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const TestDC = "testdc"
const TestExternalURL = "localhost"
const TestExternalPort = 30000

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
	seedInformerGroup.Run(wait.NeverStop)
	seedProvider := NewFakeProvider(kubeClient, seedCrdClient, seedInformerGroup)
	masterInformerGroup := masterinformer.New(masterCrdClient)

	cc, err := NewController(
		seedProvider,
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
		ControllerMetrics{},
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

func buildMasterVerionsMap() map[string]*apiv1.MasterVersion {
	return map[string]*apiv1.MasterVersion{
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

func buildMasterUpdates() []apiv1.MasterUpdate {
	return []apiv1.MasterUpdate{
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
	}
}

func NewFakeProvider(client kubernetes.Interface, crdClient seedcrdclientset.Interface, informerGroup *seedinformer.Group) *seed.Provider {
	dcs := map[string]*seed.DatacenterInteractor{
		TestDC: seed.NewDatacenterIteractor(client, crdClient, informerGroup),
	}
	return seed.NewProvider(dcs)
}
