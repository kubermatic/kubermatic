package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	kubermaticsignals "github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//TODO: Move Kubernetes versions into this as well
type excludeSelector struct {
	// The value in this map is never used, we use the keys only to have a simple set mechanism
	Distributions map[providerconfig.OperatingSystem]bool
}

// Opts represent combination of flags and ENV options
type Opts struct {
	namePrefix                     string
	providers                      sets.String
	controlPlaneReadyWaitTimeout   time.Duration
	deleteClusterAfterTests        bool
	kubeconfigPath                 string
	nodeCount                      int
	nodeReadyWaitTimeout           time.Duration
	publicKeys                     [][]byte
	reportsRoot                    string
	clusterLister                  kubermaticv1lister.ClusterLister
	kubermaticClient               kubermaticclientset.Interface
	kubeClient                     kubernetes.Interface
	clusterClientProvider          *clusterclient.Provider
	dcFile                         string
	repoRoot                       string
	dcs                            map[string]provider.DatacenterMeta
	cleanupOnStart                 bool
	clusterParallelCount           int
	workerName                     string
	homeDir                        string
	runKubermaticControllerManager bool
	versions                       []*semver.Semver
	log                            *logrus.Entry
	excludeSelector                excludeSelector
	excludeSelectorRaw             string

	secrets secrets
}

type secrets struct {
	AWS struct {
		AccessKeyID     string
		SecretAccessKey string
	}
	Azure struct {
		ClientID       string
		ClientSecret   string
		TenantID       string
		SubscriptionID string
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
	VSphere struct {
		Username string
		Password string
	}
}

const (
	defaultTimeout                 = 30 * time.Minute
	defaultUserClusterPollInterval = 10 * time.Second
	defaultAPIRetries              = 100

	controlPlaneReadyPollPeriod = 5 * time.Second
	nodesReadyPollPeriod        = 5 * time.Second
)

var (
	providers  string
	pubKeyPath string
	debug      bool
	sversions  string
)

func main() {
	mainLog := logrus.New()
	mainLog.SetLevel(logrus.InfoLevel)

	opts := Opts{
		providers:  sets.NewString(),
		publicKeys: [][]byte{},
		versions:   []*semver.Semver{},
	}

	usr, err := user.Current()
	if err != nil {
		mainLog.Fatal(err)
	}
	pubkeyPath := path.Join(usr.HomeDir, ".ssh/id_rsa.pub")

	flag.StringVar(&opts.kubeconfigPath, "kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&providers, "providers", "aws,digitalocean,openstack,hetzner,vsphere,azure", "comma separated list of providers to test")
	flag.StringVar(&opts.namePrefix, "name-prefix", "", "prefix used for all cluster names")
	flag.StringVar(&opts.repoRoot, "repo-root", "/opt/kube-test/", "Root path for the different kubernetes repositories")
	flag.IntVar(&opts.nodeCount, "kubermatic-nodes", 3, "number of worker nodes")
	flag.IntVar(&opts.clusterParallelCount, "kubermatic-parallel-clusters", 5, "number of clusters to test in parallel")
	flag.StringVar(&opts.dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&opts.reportsRoot, "reports-root", "/opt/reports", "Root for reports")
	flag.BoolVar(&opts.cleanupOnStart, "cleanup-on-start", false, "Cleans up all clusters on start and exit afterwards - must be used with name-prefix.")
	flag.DurationVar(&opts.controlPlaneReadyWaitTimeout, "kubermatic-cluster-timeout", defaultTimeout, "cluster creation timeout")
	flag.DurationVar(&opts.nodeReadyWaitTimeout, "kubermatic-nodes-timeout", defaultTimeout, "nodes creation timeout")
	flag.BoolVar(&opts.deleteClusterAfterTests, "kubermatic-delete-cluster", true, "delete test cluster at the exit")
	flag.BoolVar(&debug, "debug", true, "Enable debug logs")
	flag.StringVar(&pubKeyPath, "node-ssh-pub-key", pubkeyPath, "path to a public key which gets deployed onto every node")
	flag.StringVar(&opts.workerName, "worker-name", "", "name of the worker, if set the 'worker-name' label will be set on all clusters")
	flag.BoolVar(&opts.runKubermaticControllerManager, "run-kubermatic-controller-manager", true, "should the runner run the controller-manager")
	flag.StringVar(&sversions, "versions", "v1.10.11,v1.11.6,v1.12.4,v1.13.1", "a comma-separated list of versions to test")
	flag.StringVar(&opts.excludeSelectorRaw, "exclude-distributions", "", "a comma-separated list of distributions that will get excluded from the tests")

	flag.StringVar(&opts.secrets.AWS.AccessKeyID, "aws-access-key-id", "", "AWS: AccessKeyID")
	flag.StringVar(&opts.secrets.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS: SecretAccessKey")
	flag.StringVar(&opts.secrets.Digitalocean.Token, "digitalocean-token", "", "Digitalocean: API Token")
	flag.StringVar(&opts.secrets.Hetzner.Token, "hetzner-token", "", "Hetzner: API Token")
	flag.StringVar(&opts.secrets.OpenStack.Domain, "openstack-domain", "", "OpenStack: Domain")
	flag.StringVar(&opts.secrets.OpenStack.Tenant, "openstack-tenant", "", "OpenStack: Tenant")
	flag.StringVar(&opts.secrets.OpenStack.Username, "openstack-username", "", "OpenStack: Username")
	flag.StringVar(&opts.secrets.OpenStack.Password, "openstack-password", "", "OpenStack: Password")
	flag.StringVar(&opts.secrets.VSphere.Username, "vsphere-username", "", "vSphere: Username")
	flag.StringVar(&opts.secrets.VSphere.Password, "vsphere-password", "", "vSphere: Password")
	flag.StringVar(&opts.secrets.Azure.ClientID, "azure-client-id", "", "Azure: ClientID")
	flag.StringVar(&opts.secrets.Azure.ClientSecret, "azure-client-secret", "", "Azure: ClientSecret")
	flag.StringVar(&opts.secrets.Azure.TenantID, "azure-tenant-id", "", "Azure: TenantID")
	flag.StringVar(&opts.secrets.Azure.SubscriptionID, "azure-subscription-id", "", "Azure: SubscriptionID")

	flag.Parse()

	if debug {
		mainLog.SetLevel(logrus.DebugLevel)
	}

	if opts.excludeSelectorRaw != "" {
		excludedDistributions := strings.Split(opts.excludeSelectorRaw, ",")
		if opts.excludeSelector.Distributions == nil {
			opts.excludeSelector.Distributions = map[providerconfig.OperatingSystem]bool{}
		}
		for _, excludedDistribution := range excludedDistributions {
			switch excludedDistribution {
			case "ubuntu":
				opts.excludeSelector.Distributions[providerconfig.OperatingSystemUbuntu] = true
			case "centos":
				opts.excludeSelector.Distributions[providerconfig.OperatingSystemCentOS] = true
			case "coreos":
				opts.excludeSelector.Distributions[providerconfig.OperatingSystemCoreos] = true
			default:
				mainLog.Fatalf("Unknown distribution '%s' in '-exclude-distributions' param", excludedDistribution)
			}
		}
	}

	for _, s := range strings.Split(sversions, ",") {
		opts.versions = append(opts.versions, semver.NewSemverOrDie(s))
	}

	fields := logrus.Fields{}
	if opts.workerName != "" {
		fields["worker-name"] = opts.workerName
	}
	log := mainLog.WithFields(fields)
	opts.log = log

	for _, s := range strings.Split(providers, ",") {
		opts.providers.Insert(strings.ToLower(strings.TrimSpace(s)))
	}

	if pubKeyPath != "" {
		keyData, err := ioutil.ReadFile(pubKeyPath)
		if err != nil {
			log.Fatalf("failed to load ssh key: %v", err)
		}
		opts.publicKeys = append(opts.publicKeys, keyData)
	}

	homeDir, e2eTestPubKeyBytes, err := setupHomeDir(log)
	if err != nil {
		log.Fatalf("failed to setup temporary home dir: %v", err)
	}
	opts.publicKeys = append(opts.publicKeys, e2eTestPubKeyBytes)
	opts.homeDir = homeDir
	log = logrus.WithFields(logrus.Fields{"home": homeDir})

	stopCh := kubermaticsignals.SetupSignalHandler()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	if opts.runKubermaticControllerManager {
		controllerManagerEnviron := os.Environ()
		if opts.workerName != "" {
			controllerManagerEnviron = append(controllerManagerEnviron, fmt.Sprintf("KUBERMATIC_WORKERNAME=%s", opts.workerName))
		}
		out, err := exec.Command("go", "env", "GOPATH").CombinedOutput()
		if err != nil {
			log.Fatalf("failed to execute command `go env GOPATH`: out=%s, err=%v", string(out), err)
		}
		gopath := strings.Replace(string(out), "\n", "", -1)
		// We deliberately do not use `CommandContext` here because we expect this to be executed inside a container
		// and we want the controller to run at least as long as the conformance tester and _not_ to be killed
		// because the context for the latter got canceled, because otherwise we don't have cleanup
		command := exec.Command(path.Join(gopath, "src/github.com/kubermatic/kubermatic/api/hack/run-controller.sh"))
		command.Env = controllerManagerEnviron
		go func() {
			if out, err := command.CombinedOutput(); err != nil {
				log.Fatalf("failed to run controller-manager: Output:\n---%s\n---\nerr=%v", string(out), err)
			}
		}()
	}

	go func() {
		select {
		case <-stopCh:
			rootCancel()
			log.Info("user requested to stop the application")
		case <-rootCtx.Done():
			log.Info("context has been closed")
		}
	}()

	dcs, err := provider.LoadDatacentersMeta(opts.dcFile)
	if err != nil {
		log.Fatalf("failed to load datacenter yaml %q: %v", opts.dcFile, err)
	}
	opts.dcs = dcs

	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}

	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)
	opts.kubermaticClient = kubermaticClient
	kubeClient := kubernetes.NewForConfigOrDie(config)
	opts.kubeClient = kubeClient

	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, informer.DefaultInformerResyncPeriod)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informer.DefaultInformerResyncPeriod)

	opts.clusterLister = kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister()

	clusterClientProvider := clusterclient.New(kubeInformerFactory.Core().V1().Secrets().Lister())
	opts.clusterClientProvider = clusterClientProvider

	kubermaticInformerFactory.Start(rootCtx.Done())
	kubeInformerFactory.Start(rootCtx.Done())
	kubermaticInformerFactory.WaitForCacheSync(rootCtx.Done())
	kubeInformerFactory.WaitForCacheSync(rootCtx.Done())

	if opts.cleanupOnStart {
		if err := cleanupClusters(opts, log, kubermaticClient, clusterClientProvider); err != nil {
			log.Fatalf("failed to cleanup old clusters: %v", err)
		}
	}

	log.Info("Starting E2E tests...")
	runner := newRunner(getScenarios(opts, log), &opts)

	start := time.Now()
	if err := runner.Run(); err != nil {
		log.Fatal(err)
	}
	log.Infof("Whole suite took: %.2f seconds", time.Since(start).Seconds())
}

func cleanupClusters(opts Opts, log *logrus.Entry, kubermaticClient kubermaticclientset.Interface, clusterClientProvider *clusterclient.Provider) error {
	if opts.namePrefix == "" {
		log.Fatalf("cleanup-on-start was specified but name-prefix is empty")
	}
	clusterList, err := kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, cluster := range clusterList.Items {
		if strings.HasPrefix(cluster.Name, opts.namePrefix) {
			wg.Add(1)
			go func(cluster v1.Cluster) {
				clusterDeleteLog := logrus.WithFields(logrus.Fields{"cluster": cluster.Name})
				defer wg.Done()
				if err := tryToDeleteClusterWithRetries(clusterDeleteLog, &cluster, clusterClientProvider, kubermaticClient); err != nil {
					clusterDeleteLog.Errorf("failed to delete cluster: %v", err)
				}
			}(cluster)
		}
	}
	wg.Wait()
	log.Info("Cleaned up all old clusters")
	return nil
}

func getScenarios(opts Opts, log *logrus.Entry) []testScenario {
	var scenarios []testScenario
	if opts.providers.Has("aws") {
		log.Info("Adding AWS scenarios")
		scenarios = append(scenarios, getAWSScenarios(opts.excludeSelector, opts.versions)...)
	}
	if opts.providers.Has("digitalocean") {
		log.Info("Adding Digitalocean scenarios")
		scenarios = append(scenarios, getDigitaloceanScenarios(opts.versions)...)
	}
	if opts.providers.Has("hetzner") {
		log.Info("Adding Hetzner scenarios")
		scenarios = append(scenarios, getHetznerScenarios(opts.versions)...)
	}
	if opts.providers.Has("openstack") {
		log.Info("Adding OpenStack scenarios")
		scenarios = append(scenarios, getOpenStackScenarios(opts.versions)...)
	}
	if opts.providers.Has("vsphere") {
		log.Info("Adding vSphere scenarios")
		scenarios = append(scenarios, getVSphereScenarios(opts.versions)...)
	}
	if opts.providers.Has("azure") {
		log.Info("Adding Azure scenarios")
		scenarios = append(scenarios, getAzureScenarios(opts.versions)...)
	}
	// Shuffle scenarios - avoids timeouts caused by quota issues
	scenarios = shuffle(scenarios)
	return scenarios
}

func setupHomeDir(log *logrus.Entry) (string, []byte, error) {
	// Setup temporary home dir (Because the e2e tests have some filenames hardcoded - which might conflict with the user files)
	// We'll set the env-var $HOME to this directory when executing the tests
	homeDir, err := ioutil.TempDir("/tmp", "e2e-home-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to setup temporary home dir: %v", err)
	}
	log.Infof("Setting up temporary home directory with ssh keys at %s...", homeDir)

	if err := os.MkdirAll(path.Join(homeDir, ".ssh"), os.ModePerm); err != nil {
		return "", nil, err
	}

	// Setup temporary home dir with filepath.Join(os.Getenv("HOME"), ".ssh")
	// Make sure to create relevant ssh keys (because they are hardcoded in the e2e tests...). They must not be password protected
	log.Debug("Generating ssh keys...")
	// Private Key generation
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return "", nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return "", nil, err
	}

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	privatePEM := pem.EncodeToMemory(&privBlock)
	// Needs to be google_compute_engine as its hardcoded in the kubernetes e2e tests
	if err := ioutil.WriteFile(path.Join(homeDir, ".ssh", "google_compute_engine"), privatePEM, 0400); err != nil {
		return "", nil, err
	}

	publicRsaKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)
	if err := ioutil.WriteFile(path.Join(homeDir, ".ssh", "google_compute_engine.pub"), pubKeyBytes, 0400); err != nil {
		return "", nil, err
	}

	log.Infof("Finished setting up temporary home dir %s", homeDir)
	return homeDir, pubKeyBytes, nil
}

func shuffle(vals []testScenario) []testScenario {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]testScenario, len(vals))
	n := len(vals)
	for i := 0; i < n; i++ {
		randIndex := r.Intn(len(vals))
		ret[i] = vals[randIndex]
		vals = append(vals[:randIndex], vals[randIndex+1:]...)
	}
	return ret
}
