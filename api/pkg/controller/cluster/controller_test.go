package cluster

import (
	"fmt"
	"log"

	"github.com/kubermatic/kubermatic/api"
	crdfakeclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	kfake "k8s.io/client-go/kubernetes/fake"
)

const TestDC string = "testdc"
const TestExternalURL string = "localhost"
const TestExternalPort int = 8443

func newTestController() (*crdfakeclient.Clientset, *controller) {
	dcs, err := provider.LoadDatacentersMeta("./fixtures/datacenters.yaml")
	if err != nil {
		log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", "./fixtures/datacenters.yaml", err))
	}

	// create CloudProviders
	cps := cloud.Providers(dcs)

	versions := buildMasterVerionsMap()
	updates := buildMasterUpdates()

	kubeClient := kfake.NewSimpleClientset()
	crdClient := crdfakeclient.NewSimpleClientset()
	informerGroup := seedinformer.New(kubeClient, crdClient)
	cc, err := NewController(
		TestDC,
		kubeClient,
		crdClient,
		cps,
		versions,
		updates,
		"./../../master-resources/",
		TestExternalURL,
		"user1",
		TestExternalPort,
		dcs,
		informerGroup,
	)
	if err != nil {
		log.Fatal(err)
	}

	return crdClient, cc.(*controller)
}

func buildMasterVerionsMap() map[string]*api.MasterVersion {
	return map[string]*api.MasterVersion{
		"1.5.2": &api.MasterVersion{
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
		"1.5.3": &api.MasterVersion{
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
		"v1.6.0-rc.1": &api.MasterVersion{
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
