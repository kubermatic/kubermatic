package cluster

import (
	"log"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	fake2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"k8s.io/client-go/informers"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

const TestClusterName = "fqpcvnc6v"
const TestDC = "europe-west3-c"
const TestExternalURL = "dev.kubermatic.io"
const TestExternalPort = 30000

func newTestController(kubeObjects []runtime.Object, kubermaticObjects []runtime.Object) *Controller {
	dcs := buildDatacenterMeta()
	cps := cloud.Providers(dcs)

	versions := buildMasterVerionsMap()
	updates := buildMasterUpdates()

	kubeClient := kubefake.NewSimpleClientset(kubeObjects...)
	kubermaticClient := fake2.NewSimpleClientset(kubermaticObjects...)

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute*5)
	kubermaticInformerFactory := externalversions.NewSharedInformerFactory(kubermaticClient, time.Minute*5)

	controller, err := NewController(
		kubeClient,
		kubermaticClient,
		versions,
		updates,
		"./../../master-resources/",
		TestExternalURL,
		"",
		TestDC,
		dcs,
		cps,
		ControllerMetrics{},

		kubermaticInformerFactory.Kubermatic().V1().Clusters(),
		kubermaticInformerFactory.Etcd().V1beta2().EtcdClusters(),
		kubeInformerFactory.Core().V1().Namespaces(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeInformerFactory.Core().V1().Services(),
		kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
		kubeInformerFactory.Core().V1().ConfigMaps(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		kubeInformerFactory.Extensions().V1beta1().Deployments(),
		kubeInformerFactory.Extensions().V1beta1().Ingresses(),
		kubeInformerFactory.Rbac().V1beta1().ClusterRoleBindings(),
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
