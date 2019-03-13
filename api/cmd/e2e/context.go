package main

import (
	"context"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TestContext struct {
	ctx                    context.Context
	clusterLister          kubermaticv1lister.ClusterLister
	client                 ctrlruntimeclient.Client
	clusterClientProvider  clusterclient.UserClusterConnectionProvider
	dcs                    map[string]provider.DatacenterMeta
	deleteClustersWhenDone bool
	workingDir             string
	testBinRoot            string

	cluster        *kubermaticv1.Cluster
	nodeDeployment *kubermaticapiv1.NodeDeployment

	clusterContext struct {
		kubeconfig string
		client     ctrlruntimeclient.Client
	}
}
