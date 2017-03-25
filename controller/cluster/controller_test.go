package cluster

import (
	"fmt"
	"log"

	"github.com/kubermatic/api"
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

	versions := buildMasterVerionsMap()
	updates := buildMasterUpdates()

	clientSet := fake.NewSimpleClientset()
	cc, err := NewController("", clientSet, tprClient, cps, versions, updates, "./../../master-resources/", "localhost", true, "./../../addon-charts/")
	if err != nil {
		log.Fatal(err)
	}

	return clientSet, cc.(*clusterController)
}

func buildMasterVerionsMap() map[string]*api.MasterVersion {
	return map[string]*api.MasterVersion{
		"1.5.2": &api.MasterVersion{
			Name:                     "1.5.2",
			ID:                       "1.5.2",
			Default:                  false,
			AllowedNodeVersions:      []string{"1.3.0"},
			EtcdDeploymentYaml:       "etcd-dep.yaml",
			ApiserverDeploymentYaml:  "apiserver-dep.yaml",
			ControllerDeploymentYaml: "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:  "scheduler-dep.yaml",
			Values: map[string]string{
				"k8s-version":  "v1.5.2",
				"etcd-version": "3.0.14-kubeadm",
			},
		},
		"1.5.3": &api.MasterVersion{
			Name:                     "1.5.3",
			ID:                       "1.5.3",
			Default:                  true,
			AllowedNodeVersions:      []string{"1.3.0"},
			EtcdDeploymentYaml:       "etcd-dep.yaml",
			ApiserverDeploymentYaml:  "apiserver-dep.yaml",
			ControllerDeploymentYaml: "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:  "scheduler-dep.yaml",
			Values: map[string]string{
				"k8s-version":  "v1.5.3",
				"etcd-version": "3.0.14-kubeadm",
			},
		},
		"v1.6.0-rc.1": &api.MasterVersion{
			Name:                     "v1.6.0-rc.1",
			ID:                       "v1.6.0-rc.1",
			Default:                  false,
			AllowedNodeVersions:      []string{"1.4.0"},
			EtcdDeploymentYaml:       "etcd-dep.yaml",
			ApiserverDeploymentYaml:  "apiserver-dep.yaml",
			ControllerDeploymentYaml: "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:  "scheduler-dep.yaml",
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
