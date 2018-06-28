package main

import (
	"context"
	goflag "flag"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/golang/glog"
	flag "github.com/spf13/pflag"

	kubermaticsignals "github.com/kubermatic/kubermatic/api/pkg/signals"
)

// Opts represent combination of flags and ENV options
type Opts struct {
	Addons         []string
	ClusterPath    string
	Focus          string
	GinkgoBin      string
	KubeconfPath   string
	MachinePath    string
	Nodes          int
	Parallel       string
	Provider       string
	ReportsDir     string
	Skip           string
	TestBin        string
	ClusterTimeout time.Duration
	NodesTimeout   time.Duration
}

func main() {
	var runOpts Opts

	flag.StringVar(&runOpts.KubeconfPath, "kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&runOpts.ClusterPath, "cluster", "/manifests/cluster.yaml", "path to Cluster yaml")
	flag.StringVar(&runOpts.MachinePath, "machine", "/manifests/machine.yaml", "path to Machine yaml")
	flag.StringArrayVar(&runOpts.Addons, "addons", []string{"canal", "dns", "kube-proxy", "openvpn", "rbac"}, "comma separated list of addons")
	flag.IntVar(&runOpts.Nodes, "nodes", 1, "number of worker nodes")
	flag.DurationVar(&runOpts.ClusterTimeout, "cluster-timeout", 3*time.Minute, "cluster creation timeout")
	flag.DurationVar(&runOpts.NodesTimeout, "nodes-timeout", 10*time.Minute, "nodes creation timeout")
	flag.StringVar(&runOpts.GinkgoBin, "ginkgo-bin", "/usr/local/bin/ginkgo", "path to ginkgo binary")
	flag.StringVar(&runOpts.Focus, "ginkgo-focus", "[Conformance]", "tests focus")
	flag.StringVar(&runOpts.Skip, "ginkgo-skip", "Alpha|Kubectl|[(Disruptive|Feature:[^]]+|Flaky)]", "skip those groups of tests")
	flag.StringVar(&runOpts.Parallel, "ginkgo-parallel", "1", "parallelism of tests")
	flag.StringVar(&runOpts.TestBin, "e2e-test-bin", "/usr/local/bin/e2e.test", "path to e2e.test binary")
	flag.StringVar(&runOpts.Provider, "e2e-provider", "local", "cloud provider to use in tests")
	flag.StringVar(&runOpts.ReportsDir, "e2e-results-dir", "/tmp/results", "directory to save test results")

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.CommandLine.SortFlags = false

	flag.Parse()

	log.Info("starting")
	spew.Dump(runOpts)

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

	ctl, err := newE2ETestRunner(runOpts)
	if err != nil {
		log.Fatal(err)
	}

	err = ctl.run(rootCtx)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("e2e run done")
}
