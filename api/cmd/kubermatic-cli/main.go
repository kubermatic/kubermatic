package main

import (
	"context"
	"flag"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/util/flagopts"

	"k8s.io/client-go/kubernetes/scheme"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Opts represent combination of flags and ENV options
type Opts struct {
	Addons              flagopts.StringArray
	ClusterPath         string
	ClusterTimeout      time.Duration
	Context             string
	DeleteOnError       bool
	Kubeconf            flagopts.KubeconfigFlag
	KubermaticNamespace string
	NodePath            string
	Nodes               int
	NodesTimeout        time.Duration
	Output              string
}

func main() {
	runOpts := Opts{
		Addons: flagopts.StringArray{
			"canal",
			"dns",
			"kube-proxy",
			"openvpn",
			"rbac",
			"kubelet-configmap",
			"default-storage-class",
			"metrics-server",
		},
		Kubeconf: flagopts.NewKubeconfig(),
	}

	flag.BoolVar(&runOpts.DeleteOnError, "delete-on-error", true, "try to delete cluster on error")
	flag.DurationVar(&runOpts.ClusterTimeout, "cluster-timeout", 5*time.Minute, "cluster creation timeout")
	flag.DurationVar(&runOpts.NodesTimeout, "nodes-timeout", 10*time.Minute, "nodes creation timeout")
	flag.IntVar(&runOpts.Nodes, "nodes", 3, "number of worker nodes")
	flag.StringVar(&runOpts.ClusterPath, "cluster", "cluster.yaml", "path to Cluster yaml")
	flag.StringVar(&runOpts.Context, "context", "", "kubernetes context")
	flag.StringVar(&runOpts.KubermaticNamespace, "namespace", "kubermatic", "namespace where kubermatic and it's configs deployed")
	flag.StringVar(&runOpts.NodePath, "node", "node.yaml", "path to Node yaml")
	flag.StringVar(&runOpts.Output, "output", "usercluster_kubeconfig", "path to generated usercluster kubeconfig")
	flag.Var(&runOpts.Addons, "addons", "comma separated list of addons")
	flag.Var(&runOpts.Kubeconf, "kubeconfig", "path to kubeconfig file")

	if err := flag.CommandLine.Set("logtostderr", "1"); err != nil {
		glog.Fatal("can't set flag `logtostderr` to `1`")
	}

	flag.Parse()

	// Required to be able to use cluster-api types with the dynamic client
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatalf("failed to register clusterv1alpha1 scheme: %v", err)
	}

	glog.Info("starting")

	stopCh := signals.SetupSignalHandler()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-stopCh:
			rootCancel()
			glog.Info("user requested to stop the application")
		case <-rootCtx.Done():
			glog.Info("context has been closed")
		}
	}()

	ctl, err := newClusterCreator(runOpts)
	if err != nil {
		glog.Exit(err)
	}

	if err = ctl.create(rootCtx); err != nil {
		if runOpts.DeleteOnError {
			if errd := ctl.delete(); errd != nil {
				glog.Errorf("can't delete cluster %s: %+v", ctl.clusterName, err)
			}
		}
		glog.Exit(err)
	}

	glog.Info("cluster and machines created")
}
