/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	"path"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	cmdutil "github.com/kubermatic/kubermatic/cmd/util"
	clusterclient "github.com/kubermatic/kubermatic/pkg/cluster/client"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/semver"
	kubermaticsignals "github.com/kubermatic/kubermatic/pkg/signals"
	apitest "github.com/kubermatic/kubermatic/pkg/test/e2e/api"
	apiclient "github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/project"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/dex"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//TODO: Move Kubernetes versions into this as well
type excludeSelector struct {
	// The value in this map is never used, we use the keys only to have a simple set mechanism
	Distributions map[providerconfig.OperatingSystem]bool
}

// Opts represent combination of flags and ENV options
type Opts struct {
	namePrefix                   string
	providers                    sets.String
	controlPlaneReadyWaitTimeout time.Duration
	deleteClusterAfterTests      bool
	kubeconfigPath               string
	nodeCount                    int
	publicKeys                   [][]byte
	reportsRoot                  string
	seedClusterClient            ctrlruntimeclient.Client
	seedGeneratedClient          kubernetes.Interface
	clusterClientProvider        *clusterclient.Provider
	repoRoot                     string
	seed                         *kubermaticv1.Seed
	seedRestConfig               *rest.Config
	clusterParallelCount         int
	workerName                   string
	homeDir                      string
	versions                     []*semver.Semver
	excludeSelector              excludeSelector
	excludeSelectorRaw           string
	existingClusterLabel         string
	openshift                    bool
	openshiftPullSecret          string
	printGinkoLogs               bool
	onlyTestCreation             bool
	pspEnabled                   bool
	createOIDCToken              bool
	dexHelmValuesFile            string
	kubermatcProjectID           string
	kubermaticClient             *apiclient.Kubermatic
	kubermaticAuthenticator      runtime.ClientAuthInfoWriter
	scenarioOptions              string
	pushgatewayEndpoint          string

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
	Packet struct {
		APIKey    string
		ProjectID string
	}
	GCP struct {
		ServiceAccount string
		Network        string
		Subnetwork     string
		Zone           string
	}
	Kubevirt struct {
		Kubeconfig string
	}
	Alibaba struct {
		AccessKeyID     string
		AccessKeySecret string
	}
	kubermaticClient        *apiclient.Kubermatic
	kubermaticAuthenticator runtime.ClientAuthInfoWriter
}

const (
	defaultUserClusterPollInterval = 10 * time.Second
	defaultAPIRetries              = 100

	controlPlaneReadyPollPeriod = 5 * time.Second
)

var defaultTimeout = 10 * time.Minute

var (
	providers  string
	pubKeyPath string
	sversions  string
)

func main() {
	opts := Opts{
		providers:  sets.NewString(),
		publicKeys: [][]byte{},
		versions:   []*semver.Semver{},
	}

	defaultTimeoutMinutes := 10

	rawLog := kubermaticlog.New(true, kubermaticlog.FormatJSON)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	cmdutil.Hello(log, "Conformance Tests", true)

	// user.Current does not work in Alpine
	pubkeyPath := path.Join(os.Getenv("HOME"), ".ssh/id_rsa.pub")

	flag.StringVar(&opts.kubeconfigPath, "kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&opts.existingClusterLabel, "existing-cluster-label", "", "label to use to select an existing cluster for testing. If provided, no cluster will be created. Sample: my=cluster")
	flag.StringVar(&providers, "providers", "aws,digitalocean,openstack,hetzner,vsphere,azure,packet,gcp", "comma separated list of providers to test")
	flag.StringVar(&opts.namePrefix, "name-prefix", "", "prefix used for all cluster names")
	flag.StringVar(&opts.repoRoot, "repo-root", "/opt/kube-test/", "Root path for the different kubernetes repositories")
	flag.IntVar(&opts.nodeCount, "kubermatic-nodes", 3, "number of worker nodes")
	flag.IntVar(&opts.clusterParallelCount, "kubermatic-parallel-clusters", 5, "number of clusters to test in parallel")
	_ = flag.String("datacenters", "", "No-Op flag kept for compatibility reasons")
	flag.StringVar(&opts.reportsRoot, "reports-root", "/opt/reports", "Root for reports")
	_ = flag.Bool("cleanup-on-start", false, "No-Op kept for compatibility reasons")
	flag.DurationVar(&opts.controlPlaneReadyWaitTimeout, "kubermatic-cluster-timeout", defaultTimeout, "cluster creation timeout")
	flag.BoolVar(&opts.deleteClusterAfterTests, "kubermatic-delete-cluster", true, "delete test cluster when tests where successful")
	flag.StringVar(&pubKeyPath, "node-ssh-pub-key", pubkeyPath, "path to a public key which gets deployed onto every node")
	flag.StringVar(&opts.workerName, "worker-name", "", "name of the worker, if set the 'worker-name' label will be set on all clusters")
	flag.StringVar(&sversions, "versions", "v1.15.12,v1.16.13,,v1.17.9,v1.18.6", "a comma-separated list of versions to test")
	flag.StringVar(&opts.excludeSelectorRaw, "exclude-distributions", "", "a comma-separated list of distributions that will get excluded from the tests")
	_ = flag.Bool("run-kubermatic-controller-manager", false, "Unused, but kept for compatibility reasons")
	flag.IntVar(&defaultTimeoutMinutes, "default-timeout-minutes", 10, "The default timeout in minutes")
	flag.BoolVar(&opts.openshift, "openshift", false, "Whether to create an openshift cluster")
	flag.BoolVar(&opts.printGinkoLogs, "print-ginkgo-logs", false, "Whether to print ginkgo logs when ginkgo encountered failures")
	flag.BoolVar(&opts.onlyTestCreation, "only-test-creation", false, "Only test if nodes become ready. Does not perform any extended checks like conformance tests")
	_ = flag.Bool("debug", false, "No-Op flag kept for compatibility reasons")
	flag.BoolVar(&opts.createOIDCToken, "create-oidc-token", false, "Whether to create a OIDC token. If false, environment vars for projectID and OIDC token must be set.")
	// This won't be used directly for backwards compatibility in upgrade-tests;
	// instead the KUBERMATIC_DEX_VALUES_FILE env variable will be used.
	flag.StringVar(&opts.dexHelmValuesFile, "dex-helm-values-file", "", "Helm values.yaml of the OAuth (Dex) chart to read and configure a matching client for. Only needed if -create-oidc-token is enabled.")
	flag.StringVar(&opts.scenarioOptions, "scenario-options", "", "Additional options to be passed to scenarios, e.g. to configure specific features to be tested.")
	flag.StringVar(&opts.pushgatewayEndpoint, "pushgateway-endpoint", "", "host:port of a Prometheus Pushgateway to send runtime metrics to")

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
	flag.StringVar(&opts.secrets.Packet.APIKey, "packet-api-key", "", "Packet: APIKey")
	flag.StringVar(&opts.secrets.Packet.ProjectID, "packet-project-id", "", "Packet: ProjectID")
	flag.StringVar(&opts.secrets.GCP.ServiceAccount, "gcp-service-account", "", "GCP: Service Account")
	flag.StringVar(&opts.secrets.GCP.Zone, "gcp-zone", "europe-west3-c", "GCP: Zone")
	flag.StringVar(&opts.secrets.GCP.Network, "gcp-network", "", "GCP: Network")
	flag.StringVar(&opts.secrets.GCP.Subnetwork, "gcp-subnetwork", "", "GCP: Subnetwork")
	flag.StringVar(&opts.secrets.Kubevirt.Kubeconfig, "kubevirt-kubeconfig", "", "Kubevirt: Cluster Kubeconfig")
	flag.StringVar(&opts.secrets.Alibaba.AccessKeyID, "alibaba-access-key-id", "", "Alibaba: AccessKeyID")
	flag.StringVar(&opts.secrets.Alibaba.AccessKeySecret, "alibaba-access-key-secret", "", "Alibaba: AccessKeySecret")
	flag.Parse()

	defaultTimeout = time.Duration(defaultTimeoutMinutes) * time.Minute

	if opts.workerName != "" {
		log = log.With("worker-name", opts.workerName)
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
			case "sles":
				opts.excludeSelector.Distributions[providerconfig.OperatingSystemSLES] = true
			case "rhel":
				opts.excludeSelector.Distributions[providerconfig.OperatingSystemRHEL] = true
			case "flatcar":
				opts.excludeSelector.Distributions[providerconfig.OperatingSystemFlatcar] = true
			default:
				log.Fatalf("Unknown distribution '%s' in '-exclude-distributions' param", excludedDistribution)
			}
		}
	}

	for _, s := range strings.Split(sversions, ",") {
		opts.versions = append(opts.versions, semver.NewSemverOrDie(s))
	}

	kubermaticAPIServerAddress := os.Getenv("KUBERMATIC_HOST")
	if kubermaticAPIServerAddress == "" {
		log.Fatal("Kubermatic apiserver address must be set via KUBERMATIC_HOST env var")
	}
	kubermaticAPIServerScheme := os.Getenv("KUBERMATIC_SCHEME")
	if kubermaticAPIServerScheme == "" {
		log.Fatal("Kubermatic apiserver protocol must be set via KUBERMATIC_SCHEME env var")
	}
	opts.kubermaticClient = apiclient.New(httptransport.New(kubermaticAPIServerAddress, "", []string{kubermaticAPIServerScheme}), nil)
	opts.secrets.kubermaticClient = opts.kubermaticClient
	// May be empty if creating an OIDC token
	opts.kubermatcProjectID = strings.TrimSpace(os.Getenv("KUBERMATIC_PROJECT_ID"))

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	initMetrics(opts.pushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))
	defer updateMetrics(log)

	if !opts.createOIDCToken {
		if opts.kubermatcProjectID == "" {
			log.Fatal("Kubermatic project id must be set via KUBERMATIC_PROJECT_ID env var")
		}
		kubermaticServiceaAccountToken := os.Getenv("KUBERMATIC_SERVICEACCOUNT_TOKEN")
		if kubermaticServiceaAccountToken == "" {
			log.Fatal("A Kubermatic serviceAccountToken must be set via KUBERMATIC_SERVICEACCOUNT_TOKEN env var")
		}
		opts.kubermaticAuthenticator = httptransport.BearerToken(kubermaticServiceaAccountToken)
	} else {
		// fallback to the KUBERMATIC_DEX_HELM_VALUES_FILE env var if the CLI flag
		// was not specified due to backwards compatibility
		if opts.dexHelmValuesFile == "" {
			opts.dexHelmValuesFile = os.Getenv("KUBERMATIC_DEX_VALUES_FILE")
		}

		dexClient, err := dex.NewClientFromHelmValues(opts.dexHelmValuesFile, "kubermatic", log)
		if err != nil {
			log.Fatalw("Failed to create OIDC client", zap.Error(err))
		}

		var login, password, token string

		if err := measureTime(
			kubermaticLoginDurationMetric.WithLabelValues(),
			log,
			func() error {
				login, password = apitest.OIDCCredentials()
				token, err = dexClient.Login(rootCtx, login, password)
				return err
			},
		); err != nil {
			log.Fatalw("Failed to get master token", zap.Error(err))
		}

		log.Info("Successfully retrieved master token")

		opts.kubermaticAuthenticator = httptransport.BearerToken(token)

		if opts.kubermatcProjectID == "" {
			projectID, err := createProject(opts.kubermaticClient, opts.kubermaticAuthenticator, log)
			if err != nil {
				log.Fatalw("Failed to create project", zap.Error(err))
			}
			opts.kubermatcProjectID = projectID
		}
	}
	opts.secrets.kubermaticAuthenticator = opts.kubermaticAuthenticator

	if opts.openshift {
		opts.openshiftPullSecret = os.Getenv("OPENSHIFT_IMAGE_PULL_SECRET")
		if opts.openshiftPullSecret == "" {
			log.Fatal("Testing openshift requires the `OPENSHIFT_IMAGE_PULL_SECRET` env var to be set")
		}
	}

	if val := os.Getenv("KUBERMATIC_PSP_ENABLED"); val == "true" {
		opts.pspEnabled = true
		log.Info("Enabling PSPs")
	}

	// We use environment variables instead of flags for compatibility reasons, because during upgrade tests we
	// run two versions of the conformance tester with the same set of flags, which breaks if the older version
	// doesn't have all flags
	seedName := os.Getenv("SEED_NAME")
	if seedName == "" {
		log.Fatal("The name of the seed dc must be configured via the SEED_NAME env var")
	}

	if opts.existingClusterLabel != "" && opts.clusterParallelCount != 1 {
		log.Fatal("-cluster-parallel-count must be 1 when testing an existing cluster")
	}

	for _, s := range strings.Split(providers, ",") {
		opts.providers.Insert(strings.ToLower(strings.TrimSpace(s)))
	}

	if pubKeyPath != "" {
		keyData, err := ioutil.ReadFile(pubKeyPath)
		if err != nil {
			log.Fatalw("Failed to load ssh key", zap.Error(err))
		}
		opts.publicKeys = append(opts.publicKeys, keyData)
	}

	homeDir, e2eTestPubKeyBytes, err := setupHomeDir(log)
	if err != nil {
		log.Fatalw("Failed to setup temporary home dir", zap.Error(err))
	}
	opts.publicKeys = append(opts.publicKeys, e2eTestPubKeyBytes)
	opts.homeDir = homeDir

	stopCh := kubermaticsignals.SetupSignalHandler()

	go func() {
		select {
		case <-stopCh:
			rootCancel()
			log.Info("User requested to stop the application")
		case <-rootCtx.Done():
			log.Info("Context has been closed")
		}
	}()

	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}
	opts.seedRestConfig = config

	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalw("Failed to add clusterv1alpha1 to scheme", zap.Error(err))
	}

	if err := metricsv1beta1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalw("Failed to add metrics v1beta1 to scheme", zap.Error(err))
	}

	seedClusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		log.Fatal(err)
	}
	opts.seedClusterClient = seedClusterClient

	seedGeneratedClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	opts.seedGeneratedClient = seedGeneratedClient

	namespaceName := os.Getenv("NAMESPACE")
	if namespaceName == "" {
		log.Warn("Environment variable `NAMESPACE` was unset, defaulting to `kubermatic`")
		namespaceName = "kubermatic"
	}
	seedGetter, err := provider.SeedGetterFactory(context.Background(), seedClusterClient, seedName, namespaceName)
	if err != nil {
		log.Fatalw("Failed to consturct seedGetter", zap.Error(err))
	}
	opts.seed, err = seedGetter()
	if err != nil {
		log.Fatalw("Failed to get seed", zap.Error(err))
	}

	clusterClientProvider, err := clusterclient.NewExternal(seedClusterClient)
	if err != nil {
		log.Fatalw("Failed to get clusterClientProvider", zap.Error(err))
	}
	opts.clusterClientProvider = clusterClientProvider

	log.Info("Starting E2E tests...")
	runner := newRunner(getScenarios(opts, log), &opts, log)

	start := time.Now()
	if err := runner.Run(); err != nil {
		log.Fatal(err)
	}
	log.Infof("Whole suite took: %.2f seconds", time.Since(start).Seconds())
}

func getScenarios(opts Opts, log *zap.SugaredLogger) []testScenario {
	if opts.openshift {
		// Openshift is only supported on CentOS
		opts.excludeSelector.Distributions[providerconfig.OperatingSystemUbuntu] = true
		opts.excludeSelector.Distributions[providerconfig.OperatingSystemCoreos] = true
		opts.excludeSelector.Distributions[providerconfig.OperatingSystemCentOS] = false
		opts.excludeSelector.Distributions[providerconfig.OperatingSystemSLES] = true
		opts.excludeSelector.Distributions[providerconfig.OperatingSystemRHEL] = true
		opts.excludeSelector.Distributions[providerconfig.OperatingSystemFlatcar] = true
	}

	scenarioOptions := strings.Split(opts.scenarioOptions, ",")

	var scenarios []testScenario
	if opts.providers.Has("aws") {
		log.Info("Adding AWS scenarios")
		scenarios = append(scenarios, getAWSScenarios(opts.versions)...)
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
		scenarios = append(scenarios, getVSphereScenarios(scenarioOptions, opts.versions)...)
	}
	if opts.providers.Has("azure") {
		log.Info("Adding Azure scenarios")
		scenarios = append(scenarios, getAzureScenarios(opts.versions)...)
	}
	if opts.providers.Has("packet") {
		log.Info("Adding Packet scenarios")
		scenarios = append(scenarios, getPacketScenarios(opts.versions)...)
	}
	if opts.providers.Has("gcp") {
		log.Info("Adding GCP scenarios")
		scenarios = append(scenarios, getGCPScenarios(opts.versions)...)
	}
	if opts.providers.Has("kubevirt") {
		log.Info("Adding Kubevirt scenarios")
		scenarios = append(scenarios, getKubevirtScenarios(opts.versions, log)...)
	}
	if opts.providers.Has("alibaba") {
		log.Info("Adding Alibaba scenarios")
		scenarios = append(scenarios, getAlibabaScenarios(opts.versions)...)
	}

	var filteredScenarios []testScenario
	for _, scenario := range scenarios {
		osspec := scenario.OS()
		if osspec.Ubuntu != nil {
			if !opts.excludeSelector.Distributions[providerconfig.OperatingSystemUbuntu] {
				filteredScenarios = append(filteredScenarios, scenario)
			}
		}
		if osspec.ContainerLinux != nil {
			if !opts.excludeSelector.Distributions[providerconfig.OperatingSystemCoreos] {
				filteredScenarios = append(filteredScenarios, scenario)
			}
		}
		if osspec.Flatcar != nil {
			if !opts.excludeSelector.Distributions[providerconfig.OperatingSystemFlatcar] {
				filteredScenarios = append(filteredScenarios, scenario)
			}
		}
		if osspec.Centos != nil {
			if !opts.excludeSelector.Distributions[providerconfig.OperatingSystemCentOS] {
				filteredScenarios = append(filteredScenarios, scenario)
			}
		}
		if osspec.Sles != nil {
			if !opts.excludeSelector.Distributions[providerconfig.OperatingSystemSLES] {
				filteredScenarios = append(filteredScenarios, scenario)
			}
		}
		if osspec.Rhel != nil {
			if !opts.excludeSelector.Distributions[providerconfig.OperatingSystemRHEL] {
				filteredScenarios = append(filteredScenarios, scenario)
			}
		}
	}

	// Shuffle scenarios - avoids timeouts caused by quota issues
	return shuffle(filteredScenarios)
}

func setupHomeDir(log *zap.SugaredLogger) (string, []byte, error) {
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

func createProject(client *apiclient.Kubermatic, bearerToken runtime.ClientAuthInfoWriter, log *zap.SugaredLogger) (string, error) {
	params := &project.CreateProjectParams{Body: project.CreateProjectBody{Name: "kubermatic-conformance-tester"}}
	params.WithTimeout(15 * time.Second)

	var projectID string
	if err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		result, err := client.Project.CreateProject(params, bearerToken)
		if err != nil {
			log.Errorw("Failed to create project", "error", fmtSwaggerError(err))
			return false, nil
		}
		projectID = result.Payload.ID
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed waiting for project to get successfully created: %v", err)
	}

	getProjectParams := &project.GetProjectParams{ProjectID: projectID}
	getProjectParams.WithTimeout(15 * time.Second)
	if err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		response, err := client.Project.GetProject(getProjectParams, bearerToken)
		if err != nil {
			log.Errorw("Failed to get project", "error", fmtSwaggerError(err))
			return false, nil
		}
		if response.Payload.Status != kubermaticv1.ProjectActive {
			log.Infow("Project not active yet", "project-status", response.Payload.Status)
			return false, nil
		}
		log.Info("Successfully got project")
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to wait for project to be ready: %v", err)
	}

	return projectID, nil
}
