package main

import (
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/client-go/kubernetes"

	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

type TestContext struct {
	clusterLister          kubermaticv1lister.ClusterLister
	kubermaticClient       kubermaticclientset.Interface
	kubeClient             kubernetes.Interface
	clusterClientProvider  *clusterclient.Provider
	dcs                    map[string]provider.DatacenterMeta
	nodeCount              int
	workerName             string
	deleteClustersWhenDone bool
	workingDir             string
	testBinRoot            string

	cluster *kubermaticv1.Cluster
	node    *kubermaticapiv1.Node

	controlPlaneWaitTimeout time.Duration

	clusterContext struct {
		kubeconfig       string
		kubeClient       kubernetes.Interface
		clusterAPIClient clusterv1alpha1clientset.Interface
	}
}
