package main

import (
	"context"
	"flag"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/golang/glog"

	kubermaticsignals "github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/flagopts"
)

// Opts represent combination of flags and ENV options
type Opts struct {
	Addons         flagopts.StringArray
	ClusterPath    string
	ClusterTimeout time.Duration
	Focus          string
	GinkgoBin      string
	GinkgoNoColor  bool
	GinkgoTimeout  time.Duration
	KubeconfPath   string
	MachinePath    string
	Nodes          int
	NodesTimeout   time.Duration
	Parallel       int
	Provider       string
	ReportsDir     string
	Skip           string
	TestBin        string
}

func main() {
	runOpts := Opts{
		Addons: flagopts.StringArray{"canal", "dns", "kube-proxy", "openvpn", "rbac"},
	}

	flag.StringVar(&runOpts.KubeconfPath, "kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&runOpts.ClusterPath, "kubermatic-cluster", "/manifests/cluster.yaml", "path to Cluster yaml")
	flag.StringVar(&runOpts.MachinePath, "kubermatic-machine", "/manifests/machine.yaml", "path to Machine yaml")
	flag.Var(&runOpts.Addons, "kubermatic-addons", "comma separated list of addons")
	flag.IntVar(&runOpts.Nodes, "kubermatic-nodes", 3, "number of worker nodes")
	flag.DurationVar(&runOpts.ClusterTimeout, "kubermatic-cluster-timeout", 3*time.Minute, "cluster creation timeout")
	flag.DurationVar(&runOpts.NodesTimeout, "kubermatic-nodes-timeout", 10*time.Minute, "nodes creation timeout")
	flag.StringVar(&runOpts.GinkgoBin, "ginkgo-bin", "/usr/local/bin/ginkgo", "path to ginkgo binary")
	flag.BoolVar(&runOpts.GinkgoNoColor, "ginkgo-nocolor", false, "don't show colors")
	flag.DurationVar(&runOpts.GinkgoTimeout, "ginkgo-timeout", 5400*time.Second, "ginkgo execution timeout")
	flag.StringVar(&runOpts.Focus, "ginkgo-focus", `\[Conformance\]`, "tests focus")
	flag.StringVar(&runOpts.Skip, "ginkgo-skip", `Flaky`, "skip those groups of tests")
	flag.IntVar(&runOpts.Parallel, "ginkgo-parallel", 3, "parallelism of tests")
	flag.StringVar(&runOpts.TestBin, "e2e-test-bin", "/usr/local/bin/e2e.test", "path to e2e.test binary")
	flag.StringVar(&runOpts.Provider, "e2e-provider", "local", "cloud provider to use in tests")
	flag.StringVar(&runOpts.ReportsDir, "e2e-results-dir", "/tmp/results", "directory to save test results")

	if err := flag.CommandLine.Set("logtostderr", "1"); err != nil {
		panic("can't set flag")
	}

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
		log.Exit(err)
	}

	err = ctl.run(rootCtx)
	if err != nil {
		log.Exit(err)
	}

	log.Info("e2e run done")
}
