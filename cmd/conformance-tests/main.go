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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermativsemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	apiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/dex"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Opts represent combination of flags and ENV options
type Opts struct {
	namePrefix                   string
	providers                    sets.String
	controlPlaneReadyWaitTimeout time.Duration
	nodeReadyTimeout             time.Duration
	customTestTimeout            time.Duration
	userClusterPollInterval      time.Duration
	deleteClusterAfterTests      bool
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
	versions                     []*kubermativsemver.Semver
	distributions                map[providerconfig.OperatingSystem]struct{}
	existingClusterLabel         string
	printGinkoLogs               bool
	printContainerLogs           bool
	onlyTestCreation             bool
	pspEnabled                   bool
	createOIDCToken              bool
	dexHelmValuesFile            string
	kubermaticNamespace          string
	kubermaticEndpoint           string
	kubermaticSeedName           string
	kubermaticProjectID          string
	kubermaticOIDCToken          string
	kubermaticClient             *apiclient.KubermaticKubernetesPlatformAPI
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
	kubermaticClient        *apiclient.KubermaticKubernetesPlatformAPI
	kubermaticAuthenticator runtime.ClientAuthInfoWriter
}

var (
	providers              string
	pubKeyPath             string
	sversions              string
	sdistributions         string
	sexcludeDistributions  string
	kubevirtKubeconfigFile string
)

//nolint:gocritic,exitAfterDefer
func main() {
	var err error

	opts := Opts{
		providers:                    sets.NewString(),
		publicKeys:                   [][]byte{},
		versions:                     []*kubermativsemver.Semver{},
		kubermaticNamespace:          "kubermatic",
		kubermaticSeedName:           "kubermatic",
		controlPlaneReadyWaitTimeout: 10 * time.Minute,
		nodeReadyTimeout:             20 * time.Minute,
		customTestTimeout:            10 * time.Minute,
		userClusterPollInterval:      5 * time.Second,
	}

	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	// user.Current does not work in Alpine
	pubkeyPath := path.Join(os.Getenv("HOME"), ".ssh/id_rsa.pub")

	supportedVersions := getLatestMinorVersions(common.DefaultKubernetesVersioning.Versions)

	flag.StringVar(&opts.existingClusterLabel, "existing-cluster-label", "", "label to use to select an existing cluster for testing. If provided, no cluster will be created. Sample: my=cluster")
	flag.StringVar(&providers, "providers", "aws,digitalocean,openstack,hetzner,vsphere,azure,packet,gcp", "comma separated list of providers to test")
	flag.StringVar(&opts.namePrefix, "name-prefix", "", "prefix used for all cluster names")
	flag.StringVar(&opts.repoRoot, "repo-root", "/opt/kube-test/", "Root path for the different kubernetes repositories")
	flag.StringVar(&opts.kubermaticEndpoint, "kubermatic-endpoint", "http://localhost:8080", "scheme://host[:port] of the Kubermatic API endpoint to talk to")
	flag.StringVar(&opts.kubermaticProjectID, "kubermatic-project-id", "", "Kubermatic project to use; leave empty to create a new one")
	flag.StringVar(&opts.kubermaticOIDCToken, "kubermatic-oidc-token", "", "Token to authenticate against the Kubermatic API")
	flag.StringVar(&opts.kubermaticSeedName, "kubermatic-seed-cluster", opts.kubermaticSeedName, "Seed cluster name to create test cluster in")
	flag.StringVar(&opts.kubermaticNamespace, "kubermatic-namespace", opts.kubermaticNamespace, "Namespace where Kubermatic is installed to")
	flag.BoolVar(&opts.createOIDCToken, "create-oidc-token", false, "Whether to create a OIDC token. If false, -kubermatic-project-id and -kubermatic-oidc-token must be set")
	flag.IntVar(&opts.nodeCount, "kubermatic-nodes", 3, "number of worker nodes")
	flag.IntVar(&opts.clusterParallelCount, "kubermatic-parallel-clusters", 1, "number of clusters to test in parallel")
	flag.StringVar(&opts.reportsRoot, "reports-root", "/opt/reports", "Root for reports")
	flag.DurationVar(&opts.controlPlaneReadyWaitTimeout, "kubermatic-cluster-timeout", opts.controlPlaneReadyWaitTimeout, "cluster creation timeout")
	flag.DurationVar(&opts.nodeReadyTimeout, "node-ready-timeout", opts.nodeReadyTimeout, "base time to wait for machines to join the cluster")
	flag.DurationVar(&opts.customTestTimeout, "custom-test-timeout", opts.customTestTimeout, "timeout for Kubermatic-specific PVC/LB tests")
	flag.DurationVar(&opts.userClusterPollInterval, "user-cluster-poll-interval", opts.userClusterPollInterval, "poll interval when checking user-cluster conditions")
	flag.BoolVar(&opts.deleteClusterAfterTests, "kubermatic-delete-cluster", true, "delete test cluster when tests where successful")
	flag.StringVar(&pubKeyPath, "node-ssh-pub-key", pubkeyPath, "path to a public key which gets deployed onto every node")
	flag.StringVar(&opts.workerName, "worker-name", "", "name of the worker, if set the 'worker-name' label will be set on all clusters")
	flag.StringVar(&sversions, "versions", strings.Join(supportedVersions, ","), "a comma-separated list of versions to test")
	flag.StringVar(&sdistributions, "distributions", "", "a comma-separated list of distributions to test (cannot be used in conjunction with -exclude-distributions)")
	flag.StringVar(&sexcludeDistributions, "exclude-distributions", "", "a comma-separated list of distributions that will get excluded from the tests (cannot be used in conjunction with -distributions)")
	flag.BoolVar(&opts.printGinkoLogs, "print-ginkgo-logs", false, "Whether to print ginkgo logs when ginkgo encountered failures")
	flag.BoolVar(&opts.printContainerLogs, "print-container-logs", false, "Whether to print the logs of all controlplane containers after the test (even in case of success)")
	flag.BoolVar(&opts.onlyTestCreation, "only-test-creation", false, "Only test if nodes become ready. Does not perform any extended checks like conformance tests")
	flag.BoolVar(&opts.pspEnabled, "enable-psp", false, "When set, enables the Pod Security Policy plugin in the user cluster")
	flag.StringVar(&opts.dexHelmValuesFile, "dex-helm-values-file", "", "Helm values.yaml of the OAuth (Dex) chart to read and configure a matching client for. Only needed if -create-oidc-token is enabled.")
	flag.StringVar(&opts.scenarioOptions, "scenario-options", "", "Additional options to be passed to scenarios, e.g. to configure specific features to be tested.")
	flag.StringVar(&opts.pushgatewayEndpoint, "pushgateway-endpoint", "", "host:port of a Prometheus Pushgateway to send runtime metrics to")

	// cloud provider credentials
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
	flag.StringVar(&kubevirtKubeconfigFile, "kubevirt-kubeconfig", "", "Kubevirt: Cluster Kubeconfig filename")
	flag.StringVar(&opts.secrets.Alibaba.AccessKeyID, "alibaba-access-key-id", "", "Alibaba: AccessKeyID")
	flag.StringVar(&opts.secrets.Alibaba.AccessKeySecret, "alibaba-access-key-secret", "", "Alibaba: AccessKeySecret")

	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	if opts.workerName != "" {
		rawLog = rawLog.Named(opts.workerName)
	}

	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	cli.Hello(log, "Conformance Tests", true, nil)
	log.Infow("Kubermatic API Endpoint", "endpoint", opts.kubermaticEndpoint)

	if opts.existingClusterLabel != "" && opts.clusterParallelCount != 1 {
		log.Fatal("-cluster-parallel-count must be 1 when testing an existing cluster")
	}

	if kubevirtKubeconfigFile != "" {
		content, err := ioutil.ReadFile(kubevirtKubeconfigFile)
		if err != nil {
			log.Fatalw("Failed to read kubevirt kubeconfig file", zap.Error(err))
		}

		opts.secrets.Kubevirt.Kubeconfig = string(content)
	}

	opts.distributions, err = getEffectiveDistributions(sdistributions, sexcludeDistributions)
	if err != nil {
		log.Fatalw("Failed to determine distribution list", zap.Error(err))
	}

	if len(opts.distributions) == 0 {
		log.Fatal("No distribution to use in tests remained after evaluating -distributions and -exclude-distributions")
	}

	var osNames []string
	for dist := range opts.distributions {
		osNames = append(osNames, string(dist))
	}
	sort.Strings(osNames)
	log.Infow("Enabled operating system", "distributions", osNames)

	for _, s := range strings.Split(sversions, ",") {
		opts.versions = append(opts.versions, kubermativsemver.NewSemverOrDie(s))
	}
	log.Infow("Enabled versions", "versions", opts.versions)

	kubermaticClient, err := utils.NewKubermaticClient(opts.kubermaticEndpoint)
	if err != nil {
		log.Fatalf("Failed to create Kubermatic API client: %v", err)
	}

	opts.kubermaticClient = kubermaticClient
	opts.secrets.kubermaticClient = opts.kubermaticClient

	rootCtx := signals.SetupSignalHandler()

	// collect runtime metrics if there is a pushgateway URL configured
	// and these variables are set
	initMetrics(opts.pushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))
	defer updateMetrics(log)

	if !opts.createOIDCToken {
		if opts.kubermaticOIDCToken == "" {
			log.Fatal("An existing OIDC token must be set via the -kubermatic-oidc-token flag")
		}

		opts.kubermaticAuthenticator = httptransport.BearerToken(opts.kubermaticOIDCToken)
	} else {
		dexClient, err := dex.NewClientFromHelmValues(opts.dexHelmValuesFile, "kubermatic", log)
		if err != nil {
			log.Fatalw("Failed to create OIDC client", zap.Error(err))
		}

		// OIDC credentials are passed in as environment variables instead of
		// CLI flags because having the password as a flag might be a security
		// issue and then it also makes sense to handle the login name in the
		// same way.
		// Also, the API E2E tests use environment variables to get the values
		// into the `go test` runs.
		login, password, err := utils.OIDCCredentials()
		if err != nil {
			log.Fatalf("Invalid OIDC credentials: %v", err)
		}

		log.Infow("Creating login token", "login", login, "provider", dexClient.ProviderURI, "client", dexClient.ClientID)

		var token string

		if err := measureTime(
			kubermaticLoginDurationMetric.WithLabelValues(),
			log,
			func() error {
				token, err = dexClient.Login(rootCtx, login, password)
				return err
			},
		); err != nil {
			log.Fatalw("Failed to get master token", zap.Error(err))
		}

		log.Info("Successfully retrieved master token")

		opts.kubermaticAuthenticator = httptransport.BearerToken(token)
	}
	opts.secrets.kubermaticAuthenticator = opts.kubermaticAuthenticator

	if opts.kubermaticProjectID == "" {
		projectID, err := createProject(rootCtx, opts.kubermaticClient, opts.kubermaticAuthenticator, log)
		if err != nil {
			log.Fatalw("Failed to create project", zap.Error(err))
		}

		opts.kubermaticProjectID = projectID
	}

	log.Infow("Using project", "project", opts.kubermaticProjectID)

	for _, s := range strings.Split(providers, ",") {
		opts.providers.Insert(strings.ToLower(strings.TrimSpace(s)))
	}
	log.Infow("Enabled cloud providers", "providers", opts.providers.List())

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

	if err := createSSHKeys(rootCtx, opts.kubermaticClient, opts.kubermaticAuthenticator, &opts, log); err != nil {
		log.Fatalw("Failed to create SSH keys", zap.Error(err))
	}

	_, _, config, err := utils.GetClients()
	if err != nil {
		log.Fatalw("Failed to get client config", zap.Error(err))
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

	seedGetter, err := provider.SeedGetterFactory(rootCtx, seedClusterClient, opts.kubermaticSeedName, opts.kubermaticNamespace)
	if err != nil {
		log.Fatalw("Failed to construct seedGetter", zap.Error(err))
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
	if err := runner.Run(rootCtx); err != nil {
		log.Fatal(err)
	}
	log.Infof("Whole suite took: %.2f seconds", time.Since(start).Seconds())
}

func getScenarios(opts Opts, log *zap.SugaredLogger) []testScenario {
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

	hasDistribution := func(distribution providerconfig.OperatingSystem) bool {
		_, ok := opts.distributions[distribution]
		return ok
	}

	var filteredScenarios []testScenario
	for _, scenario := range scenarios {
		osspec := scenario.OS()
		if osspec.Ubuntu != nil && hasDistribution(providerconfig.OperatingSystemUbuntu) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Flatcar != nil && hasDistribution(providerconfig.OperatingSystemFlatcar) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Centos != nil && hasDistribution(providerconfig.OperatingSystemCentOS) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Sles != nil && hasDistribution(providerconfig.OperatingSystemSLES) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Rhel != nil && hasDistribution(providerconfig.OperatingSystemRHEL) {
			filteredScenarios = append(filteredScenarios, scenario)
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

func createProject(ctx context.Context, client *apiclient.KubermaticKubernetesPlatformAPI, bearerToken runtime.ClientAuthInfoWriter, log *zap.SugaredLogger) (string, error) {
	log.Info("Creating Kubermatic project...")

	params := &project.CreateProjectParams{
		Context: ctx,
		Body:    project.CreateProjectBody{Name: "kubermatic-conformance-tester"},
	}
	utils.SetupParams(nil, params, 3*time.Second, 1*time.Minute, http.StatusConflict)

	result, err := client.Project.CreateProject(params, bearerToken)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %v", err)
	}
	projectID := result.Payload.ID

	// we have to wait a moment for the RBAC stuff to be reconciled, and to try to avoid
	// logging a misleading error in the following loop, we just wait a few seconds
	time.Sleep(3 * time.Second)

	getProjectParams := &project.GetProjectParams{Context: ctx, ProjectID: projectID}
	utils.SetupParams(nil, getProjectParams, 2*time.Second, 1*time.Minute, http.StatusConflict)

	if err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		response, err := client.Project.GetProject(getProjectParams, bearerToken)
		if err != nil {
			log.Errorw("Failed to get project", zap.Error(err))
			return false, nil
		}
		if response.Payload.Status != kubermaticv1.ProjectActive {
			log.Warnw("Project not active yet", "project-status", response.Payload.Status)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to wait for project to be ready: %v", err)
	}

	return projectID, nil
}

func createSSHKeys(ctx context.Context, client *apiclient.KubermaticKubernetesPlatformAPI, bearerToken runtime.ClientAuthInfoWriter, opts *Opts, log *zap.SugaredLogger) error {
	for i, key := range opts.publicKeys {
		log.Infow("Creating UserSSHKey", "pubkey", string(key))

		body := &project.CreateSSHKeyParams{
			Context:   ctx,
			ProjectID: opts.kubermaticProjectID,
			Key: &models.SSHKey{
				Name: fmt.Sprintf("SSH Key No. %d", i+1),
				Spec: &models.SSHKeySpec{
					PublicKey: string(key),
				},
			},
		}
		utils.SetupParams(nil, body, 3*time.Second, 1*time.Minute, http.StatusConflict)

		if _, err := client.Project.CreateSSHKey(body, bearerToken); err != nil {
			return fmt.Errorf("failed to create SSH key: %v", err)
		}
	}

	return nil
}

func getLatestMinorVersions(versions []*semver.Version) []string {
	minorMap := map[uint64]*semver.Version{}

	for i, version := range versions {
		minor := version.Minor()

		if existing := minorMap[minor]; existing == nil || existing.LessThan(version) {
			minorMap[minor] = versions[i]
		}
	}

	list := []string{}
	for _, v := range minorMap {
		list = append(list, "v"+v.String())
	}
	sort.Strings(list)

	return list
}

func getEffectiveDistributions(distributions string, excludeDistributions string) (map[providerconfig.OperatingSystem]struct{}, error) {
	all := providerconfig.AllOperatingSystems

	if distributions == "" && excludeDistributions == "" {
		return nil, fmt.Errorf("either -distributions or -exclude-distributions must be given (each is a comma-separated list of %v)", all)
	}

	if distributions != "" && excludeDistributions != "" {
		return nil, errors.New("-distributions and -exclude-distributions must not be given at the same time")
	}

	chosen := map[providerconfig.OperatingSystem]struct{}{}

	if distributions != "" {
		for _, distribution := range strings.Split(distributions, ",") {
			exists := false
			for _, dist := range all {
				if string(dist) == distribution {
					exists = true
					break
				}
			}

			if !exists {
				return nil, fmt.Errorf("unknown distribution %q specified", distribution)
			}

			chosen[providerconfig.OperatingSystem(distribution)] = struct{}{}
		}
	} else {
		excluded := strings.Split(excludeDistributions, ",")

		for _, dist := range all {
			exclude := false
			for _, ex := range excluded {
				if string(dist) == ex {
					exclude = true
					break
				}
			}

			if !exclude {
				chosen[dist] = struct{}{}
			}
		}
	}

	return chosen, nil
}
