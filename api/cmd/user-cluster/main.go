package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	log "github.com/golang/glog"
	"k8s.io/client-go/util/homedir"

	kubermaticsignals "github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/flagopts"
)

// Opts represent combination of flags and ENV options
type Opts struct {
	Addons              flagopts.StringArray
	ClusterPath         string
	ClusterTimeout      time.Duration
	DeleteOnError       bool
	KubeconfPath        string
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
	}

	defaultKubeconfig, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		defaultKubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	flag.BoolVar(&runOpts.DeleteOnError, "delete", true, "delete on error")
	flag.DurationVar(&runOpts.ClusterTimeout, "cluster-timeout", 5*time.Minute, "cluster creation timeout")
	flag.DurationVar(&runOpts.NodesTimeout, "nodes-timeout", 10*time.Minute, "nodes creation timeout")
	flag.IntVar(&runOpts.Nodes, "nodes", 3, "number of worker nodes")
	flag.StringVar(&runOpts.ClusterPath, "cluster", "cluster.yaml", "path to Cluster yaml")
	flag.StringVar(&runOpts.KubeconfPath, "kubeconfig", "$KUBECONFIG or $HOME/.kube/config", "path to kubeconfig file")
	flag.StringVar(&runOpts.KubermaticNamespace, "namespace", "kubermatic", "namespace where kubermatic and it's configs deployed")
	flag.StringVar(&runOpts.NodePath, "node", "node.yaml", "path to Node yaml")
	flag.StringVar(&runOpts.Output, "output", "usercluster_kubeconfig", "path to generated usercluster kubeconfig")
	flag.Var(&runOpts.Addons, "addons", "comma separated list of addons")

	if err := flag.CommandLine.Set("logtostderr", "1"); err != nil {
		panic("can't set flag")
	}

	flag.Parse()

	if runOpts.KubeconfPath == "$KUBECONFIG or $HOME/.kube/config" {
		runOpts.KubeconfPath = defaultKubeconfig
	}

	log.Info("starting")

	stopCh := kubermaticsignals.SetupSignalHandler()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-stopCh:
			rootCancel()
			log.Info("user requested to stop the application")
		case <-rootCtx.Done():
			log.Info("context has been closed")
		}
	}()

	ctl, err := newClusterCreator(runOpts)

	if runOpts.DeleteOnError {
		//TODO: defer ctl.deleteCluster()
	}

	if err != nil {
		log.Exit(err)
	}

	err = ctl.create(rootCtx)
	if err != nil {
		log.Exit(err)
	}

	log.Info("cluster and machines created")
}
