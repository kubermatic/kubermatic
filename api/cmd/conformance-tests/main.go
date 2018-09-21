package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"

	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubermaticsignals "github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var supportedVersions = []*semver.Version{
	semver.MustParse("v1.9.10"),
	semver.MustParse("v1.10.8"),
	semver.MustParse("v1.11.3"),
}

// Opts represent combination of flags and ENV options
type Opts struct {
	namePrefix                   string
	controlPlaneReadyWaitTimeout time.Duration
	deleteClusterAfterTests      bool
	kubeconfigPath               string
	nodeCount                    int
	nodeReadyWaitTimeout         time.Duration
	reportsRoot                  string
	clusterLister                kubermaticv1lister.ClusterLister
	kubermaticClient             kubermaticclientset.Interface
	clusterClientProvider        *clusterclient.Provider
	dcFile                       string
	testBinPath                  string
	dcs                          map[string]provider.DatacenterMeta
	cleanupOnStart               bool
	clusterParallelCount         int

	secrets secrets
}

type secrets struct {
	AWS struct {
		AccessKeyID     string
		SecretAccessKey string
	}
	Digitalocean struct {
		Token string
	}
	Hetzner struct {
		Token string
	}
	OpenStack struct {
		Domain   string
		Tenant   string
		Username string
		Password string
	}
}

const (
	defaultTimeout = 30 * time.Minute

	controlPlaneReadyPollPeriod = 5 * time.Second
	nodesReadyPollPeriod        = 5 * time.Second
)

func main() {
	opts := Opts{}

	flag.StringVar(&opts.kubeconfigPath, "kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&opts.namePrefix, "name-prefix", "", "prefix used for all cluster names")
	flag.StringVar(&opts.testBinPath, "test-bin-path", "/opt/kube-test/", "Rootpath for the test binaries")
	flag.IntVar(&opts.nodeCount, "kubermatic-nodes", 3, "number of worker nodes")
	flag.IntVar(&opts.clusterParallelCount, "kubermatic-parallel-clusters", 5, "number of clusters to test in parallel")
	flag.StringVar(&opts.dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&opts.reportsRoot, "reports-root", "/opt/reports", "Root for reports")
	flag.BoolVar(&opts.cleanupOnStart, "cleanup-on-start", false, "Cleans up all clusters on start and exit afterwards - must be used with name-prefix.")
	flag.DurationVar(&opts.controlPlaneReadyWaitTimeout, "kubermatic-cluster-timeout", defaultTimeout, "cluster creation timeout")
	flag.DurationVar(&opts.nodeReadyWaitTimeout, "kubermatic-nodes-timeout", defaultTimeout, "nodes creation timeout")
	flag.BoolVar(&opts.deleteClusterAfterTests, "kubermatic-delete-cluster", true, "delete test cluster at the exit")

	flag.StringVar(&opts.secrets.AWS.AccessKeyID, "aws-access-key-id", "", "AWS: AccessKeyID")
	flag.StringVar(&opts.secrets.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS: SecretAccessKey")
	flag.StringVar(&opts.secrets.Digitalocean.Token, "digitalocean-token", "", "Digitalocean: API Token")
	flag.StringVar(&opts.secrets.Hetzner.Token, "hetzner-token", "", "Hetzner: API Token")
	flag.StringVar(&opts.secrets.OpenStack.Domain, "openstack-domain", "", "OpenStack: Domain")
	flag.StringVar(&opts.secrets.OpenStack.Tenant, "openstack-tenant", "", "OpenStack: Tenant")
	flag.StringVar(&opts.secrets.OpenStack.Username, "openstack-username", "", "OpenStack: Username")
	flag.StringVar(&opts.secrets.OpenStack.Password, "openstack-password", "", "OpenStack: Password")

	if err := flag.CommandLine.Set("logtostderr", "1"); err != nil {
		fmt.Printf("failed to set logtostderr flag: %v\n", err)
		os.Exit(1)
	}
	flag.Parse()

	stopCh := kubermaticsignals.SetupSignalHandler()
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

	dcs, err := provider.LoadDatacentersMeta(opts.dcFile)
	if err != nil {
		glog.Fatalf("failed to load datacenter yaml %q: %v", opts.dcFile, err)
	}
	opts.dcs = dcs

	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfigPath)
	if err != nil {
		glog.Fatal(err)
	}

	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)

	if opts.cleanupOnStart {
		if opts.namePrefix == "" {
			log.Fatalf("cleanup-on-start was specified but name-prefix is empty")
		}

		clusterList, err := kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}
		for _, cluster := range clusterList.Items {
			if strings.HasPrefix(cluster.Name, opts.namePrefix) {
				p := metav1.DeletePropagationBackground
				opts := metav1.DeleteOptions{PropagationPolicy: &p}
				fmt.Println("Deleting cluster " + cluster.Name + "...")
				if err = kubermaticClient.KubermaticV1().Clusters().Delete(cluster.Name, &opts); err != nil {
					log.Fatalf("failed to delete cluster %s: %v", cluster.Name, err)
				}
			}
		}

		fmt.Println("Cleaned up all old clusters")
		os.Exit(0)
	}

	kubeClient := kubernetes.NewForConfigOrDie(config)
	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, informer.DefaultInformerResyncPeriod)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informer.DefaultInformerResyncPeriod)

	opts.kubermaticClient = kubermaticClient
	opts.clusterLister = kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister()

	opts.clusterClientProvider = clusterclient.New(kubeInformerFactory.Core().V1().Secrets().Lister())

	kubermaticInformerFactory.Start(rootCtx.Done())
	kubeInformerFactory.Start(rootCtx.Done())
	kubermaticInformerFactory.WaitForCacheSync(rootCtx.Done())
	kubeInformerFactory.WaitForCacheSync(rootCtx.Done())

	glog.Info("Starting E2E tests...")

	var scenarios []testScenario
	scenarios = append(scenarios, getAWSScenarios()...)
	scenarios = append(scenarios, getDigitaloceanScenarios()...)
	scenarios = append(scenarios, getHetznerScenarios()...)
	scenarios = append(scenarios, getOpenStackScenarios()...)

	runner := newRunner(scenarios, &opts)
	hadFailure, err := runner.Run()
	if err != nil {
		glog.Fatalf("failed to execute the scenarios: %v", err)
	}

	if hadFailure {
		glog.Infof("Some tests failed.")
		os.Exit(1)
	}
}
