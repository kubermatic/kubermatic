package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"

	k8csignals "github.com/kubermatic/kubermatic/api/pkg/signals"
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

func lookupEnv(key, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return val
}

func main() {
	log.Print("starting")

	var runOpts Opts

	flag.IntVar(&runOpts.Nodes,
		"nodes", 1, "number of worker nodes")
	flag.StringArrayVar(&runOpts.Addons,
		"addons",
		[]string{"canal", "dns", "kube-proxy", "openvpn", "rbac"},
		"coma separated list of addons")
	flag.StringVar(&runOpts.ClusterPath,
		"cluster", "/manifests/cluster.yaml", "path to Cluster yaml")
	flag.StringVar(&runOpts.KubeconfPath,
		"kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&runOpts.MachinePath,
		"machine", "/manifests/machine.yaml", "path to Machine yaml")
	flag.DurationVar(&runOpts.ClusterTimeout,
		"cluster-timeout", 3*time.Minute, "cluster creation timeout")
	flag.DurationVar(&runOpts.NodesTimeout,
		"nodes-timeout", 10*time.Minute, "nodes creation timeout")

	runOpts.GinkgoBin = lookupEnv("E2E_GINKGO", "/usr/local/bin/ginkgo")
	runOpts.TestBin = lookupEnv("E2E_TEST", "/usr/local/bin/e2e.test")
	runOpts.Focus = lookupEnv("E2E_FOCUS", "[Conformance]")
	runOpts.Skip = lookupEnv("E2E_SKIP", "Alpha|Kubectl|[(Disruptive|Feature:[^]]+|Flaky)]")
	runOpts.Provider = lookupEnv("E2E_PROVIDER", "local")
	runOpts.Parallel = lookupEnv("E2E_PARALLEL", "1")
	runOpts.ReportsDir = lookupEnv("E2E_REPORTS_DIR", "/tmp/results")
	flag.Parse()

	spew.Dump(runOpts)

	stopCh := k8csignals.SetupSignalHandler()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-stopCh:
			rootCancel()
			log.Print("user requested to stop the application")
		case <-rootCtx.Done():
			log.Print("parent context has been closed - propagating the request")
		}
	}()

	ctl, err := newE2ETestsController(runOpts)
	if err != nil {
		log.Fatal(err)
	}

	err = ctl.run(rootCtx)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("e2e run done")
}
