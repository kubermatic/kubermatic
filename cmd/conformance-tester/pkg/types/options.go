/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package types

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermativsemver "k8c.io/kubermatic/sdk/v2/semver"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	pubKeyPath             string
	kubevirtKubeconfigFile string
	ovdcNetworks           string
)

// Options represent combination of flags and ENV options.
type Options struct {
	Client     string
	NamePrefix string

	// these flags control the test scenario matrix;
	// the Enabled... and Exclude... sets are combined
	// together to result in the final ... set

	Providers            sets.Set[string]
	Releases             sets.Set[string]
	Versions             []*kubermativsemver.Semver
	EnableDistributions  sets.Set[string]
	ExcludeDistributions sets.Set[string]
	Distributions        sets.Set[string]
	EnableTests          sets.Set[string]
	ExcludeTests         sets.Set[string]
	Tests                sets.Set[string]

	// The tester can export the result status for all executed scenarios
	// into a JSON file and then re-read that to retry failed runs.
	ResultsFile          string
	RetryFailedScenarios bool
	ScenarioFile         string

	// additional settings identical for all scenarios

	DualStackEnabled    bool
	KonnectivityEnabled bool
	ScenarioOptions     sets.Set[string]
	TestClusterUpdate   bool

	// additional settings

	ControlPlaneReadyWaitTimeout time.Duration
	NodeReadyTimeout             time.Duration
	CustomTestTimeout            time.Duration
	UserClusterPollInterval      time.Duration
	DeleteClusterAfterTests      bool
	WaitForClusterDeletion       bool
	NodeCount                    int
	PublicKeys                   [][]byte
	ReportsRoot                  string
	LogDirectory                 string
	SeedClusterClient            ctrlruntimeclient.Client
	SeedGeneratedClient          kubernetes.Interface
	ClusterClientProvider        *clusterclient.Provider
	RepoRoot                     string
	Seed                         *kubermaticv1.Seed
	SeedRestConfig               *rest.Config
	ClusterParallelCount         int
	HomeDir                      string
	ExistingClusterLabel         string
	KubermaticNamespace          string
	KubermaticSeedName           string
	KubermaticProject            string
	KubermaticConfiguration      *kubermaticv1.KubermaticConfiguration
	PushgatewayEndpoint          string

	Secrets Secrets
}

// Secrets contains all the provider credentials.
type Secrets struct {
	Anexia struct {
		KKPDatacenter string
		Token         string
		TemplateID    string
		VlanID        string
	}
	AWS struct {
		KKPDatacenter   string
		AccessKeyID     string
		SecretAccessKey string
	}
	Azure struct {
		KKPDatacenter  string
		ClientID       string
		ClientSecret   string
		TenantID       string
		SubscriptionID string
	}
	Digitalocean struct {
		KKPDatacenter string
		Token         string
	}
	Hetzner struct {
		KKPDatacenter string
		Token         string
	}
	OpenStack struct {
		KKPDatacenter string
		Domain        string
		Project       string
		ProjectID     string
		Username      string
		Password      string
	}
	VSphere struct {
		KKPDatacenter string
		Username      string
		Password      string
	}
	GCP struct {
		KKPDatacenter string
		// ServiceAccount is the plaintext Service account (as JSON) without any (base64) encoding.
		ServiceAccount string
		Network        string
		Subnetwork     string
	}
	Kubevirt struct {
		KKPDatacenter string
		// Kubeconfig is the plaintext kubeconfig without any (base64) encoding.
		Kubeconfig string
	}
	Alibaba struct {
		KKPDatacenter   string
		AccessKeyID     string
		AccessKeySecret string
	}
	Nutanix struct {
		KKPDatacenter string
		Username      string
		Password      string
		CSIUsername   string
		CSIPassword   string
		CSIEndpoint   string
		CSIPort       int32
		ProxyURL      string
		ClusterName   string
		ProjectName   string
		SubnetName    string
	}
	VMwareCloudDirector struct {
		KKPDatacenter string
		Username      string
		Password      string
		Organization  string
		VDC           string
		OVDCNetworks  []string
	}
	RHEL struct {
		SubscriptionUser     string
		SubscriptionPassword string
		OfflineToken         string
	}
}

func NewDefaultOptions() *Options {
	return &Options{
		Client:                       "kube",
		ScenarioOptions:              sets.New[string](),
		Providers:                    sets.New[string](),
		Releases:                     sets.New(version.GetLatestMinorVersions(defaulting.DefaultKubernetesVersioning.Versions)...),
		EnableDistributions:          sets.New[string](),
		ExcludeDistributions:         sets.New[string](),
		Distributions:                sets.New[string](),
		EnableTests:                  sets.New[string](),
		ExcludeTests:                 sets.New[string](),
		Tests:                        sets.New[string](),
		KubermaticNamespace:          "kubermatic",
		KubermaticSeedName:           "kubermatic",
		ControlPlaneReadyWaitTimeout: 10 * time.Minute,
		NodeReadyTimeout:             20 * time.Minute,
		CustomTestTimeout:            10 * time.Minute,
		UserClusterPollInterval:      5 * time.Second,
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// user.Current does not work in Alpine
	pubKeyPath = path.Join(os.Getenv("HOME"), ".ssh/id_rsa.pub")

	fs.StringVar(&o.Client, "client", o.Client, "controls how to interact with KKP; can be either `api` or `kube`")
	fs.StringVar(&o.ExistingClusterLabel, "existing-cluster-label", "", "label to use to select an existing cluster for testing. If provided, no cluster will be created. Sample: my=cluster")
	fs.StringVar(&o.NamePrefix, "name-prefix", "", "prefix used for all cluster names")
	fs.Var(flagopts.SetFlag(o.Providers), "providers", "Comma-separated list of providers to test")
	fs.Var(flagopts.SetFlag(o.Releases), "releases", "Comma-separated list of Kubernetes releases (e.g. '1.24') to test")
	fs.Var(flagopts.SetFlag(o.EnableDistributions), "distributions", "Comma-separated list of distributions to test (cannot be used in conjunction with -exclude-distributions)")
	fs.Var(flagopts.SetFlag(o.ExcludeDistributions), "exclude-distributions", "Comma-separated list of distributions that will get excluded from the tests (cannot be used in conjunction with -distributions)")
	fs.Var(flagopts.SetFlag(o.EnableTests), "tests", "Comma-separated list of enabled tests (cannot be used in conjunction with -exclude-tests)")
	fs.Var(flagopts.SetFlag(o.ExcludeTests), "exclude-tests", "Run all the tests except the ones in this comma-separated list (cannot be used in conjunction with -tests)")
	fs.Var(flagopts.SetFlag(o.ScenarioOptions), "scenario-options", "Comma-separated list of additional options to be passed to scenarios, e.g. to configure specific features to be tested.")
	fs.StringVar(&o.RepoRoot, "repo-root", "/opt/kube-test/", "Root path for the different kubernetes repositories")
	fs.StringVar(&o.KubermaticProject, "kubermatic-project", "", "Kubermatic project to use; leave empty to create a new one")
	fs.StringVar(&o.KubermaticSeedName, "kubermatic-seed-cluster", o.KubermaticSeedName, "Seed cluster name to create test cluster in")
	fs.StringVar(&o.KubermaticNamespace, "kubermatic-namespace", o.KubermaticNamespace, "Namespace where Kubermatic is installed to")
	fs.IntVar(&o.NodeCount, "kubermatic-nodes", 3, "number of worker nodes")
	fs.IntVar(&o.ClusterParallelCount, "kubermatic-parallel-clusters", 1, "number of clusters to test in parallel")
	fs.StringVar(&o.ReportsRoot, "reports-root", "/opt/reports", "Root for reports")
	fs.StringVar(&o.LogDirectory, "log-directory", "", "Root directory to place container logs into")
	fs.DurationVar(&o.ControlPlaneReadyWaitTimeout, "kubermatic-cluster-timeout", o.ControlPlaneReadyWaitTimeout, "cluster creation timeout")
	fs.DurationVar(&o.NodeReadyTimeout, "node-ready-timeout", o.NodeReadyTimeout, "base time to wait for machines to join the cluster")
	fs.DurationVar(&o.CustomTestTimeout, "custom-test-timeout", o.CustomTestTimeout, "timeout for Kubermatic-specific PVC/LB tests")
	fs.DurationVar(&o.UserClusterPollInterval, "user-cluster-poll-interval", o.UserClusterPollInterval, "poll interval when checking user-cluster conditions")
	fs.BoolVar(&o.DeleteClusterAfterTests, "kubermatic-delete-cluster", true, "delete test cluster when tests where successful")
	fs.BoolVar(&o.WaitForClusterDeletion, "wait-for-cluster-deletion", true, "wait for the cluster deletion to have finished")
	fs.StringVar(&pubKeyPath, "node-ssh-pub-key", pubKeyPath, "path to a public key which gets deployed onto every node")
	fs.BoolVar(&o.DualStackEnabled, "enable-dualstack", false, "When set, enables dualstack (IPv4+IPv6 networking) in the user cluster")
	fs.BoolVar(&o.KonnectivityEnabled, "enable-konnectivity", true, "When set, enables Konnectivity (proxy service for control plane communication) in the user cluster. When set to false, OpenVPN is used")
	fs.BoolVar(&o.TestClusterUpdate, "update-cluster", false, "When set, will first run the selected tests, then update the cluster and nodes to their next minor release and then run the same tests again")
	fs.StringVar(&o.PushgatewayEndpoint, "pushgateway-endpoint", "", "host:port of a Prometheus Pushgateway to send runtime metrics to")
	fs.StringVar(&o.ResultsFile, "results-file", "", "path to a JSON file where the test result will be written to / read from (when also using --retry)")
	fs.BoolVar(&o.RetryFailedScenarios, "retry", false, "when using --results-file, will filter the given scenarios to only run those that previously failed")
	fs.StringVar(&o.ScenarioFile, "scenarios-file", "", "Path to a YAML file defining test scenarios.")

	o.Secrets.AddFlags(fs)
}

func (s *Secrets) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Anexia.Token, "anexia-token", "", "Anexia: API Token")
	fs.StringVar(&s.Anexia.TemplateID, "anexia-template-id", "", "Anexia: Template ID")
	fs.StringVar(&s.Anexia.VlanID, "anexia-vlan-id", "", "Anexia: VLAN ID")
	fs.StringVar(&s.Anexia.KKPDatacenter, "anexia-kkp-datacenter", "", "Anexia: KKP datacenter to use")
	fs.StringVar(&s.AWS.AccessKeyID, "aws-access-key-id", "", "AWS: AccessKeyID")
	fs.StringVar(&s.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS: SecretAccessKey")
	fs.StringVar(&s.AWS.KKPDatacenter, "aws-kkp-datacenter", "", "AWS: KKP datacenter to use")
	fs.StringVar(&s.Digitalocean.Token, "digitalocean-token", "", "Digitalocean: API Token")
	fs.StringVar(&s.Digitalocean.KKPDatacenter, "digitalocean-kkp-datacenter", "", "Digitalocean: KKP datacenter to use")
	fs.StringVar(&s.Hetzner.Token, "hetzner-token", "", "Hetzner: API Token")
	fs.StringVar(&s.Hetzner.KKPDatacenter, "hetzner-kkp-datacenter", "", "Hetzner: KKP datacenter to use")
	fs.StringVar(&s.OpenStack.Domain, "openstack-domain", "", "OpenStack: Domain")
	fs.StringVar(&s.OpenStack.Project, "openstack-project", "", "OpenStack: Project")
	fs.StringVar(&s.OpenStack.ProjectID, "openstack-project-id", "", "OpenStack: Project ID")
	fs.StringVar(&s.OpenStack.Username, "openstack-username", "", "OpenStack: Username")
	fs.StringVar(&s.OpenStack.Password, "openstack-password", "", "OpenStack: Password")
	fs.StringVar(&s.OpenStack.KKPDatacenter, "openstack-kkp-datacenter", "", "OpenStack: KKP datacenter to use")
	fs.StringVar(&s.VSphere.Username, "vsphere-username", "", "vSphere: Username")
	fs.StringVar(&s.VSphere.Password, "vsphere-password", "", "vSphere: Password")
	fs.StringVar(&s.VSphere.KKPDatacenter, "vsphere-kkp-datacenter", "", "vSphere: KKP datacenter to use")
	fs.StringVar(&s.Azure.ClientID, "azure-client-id", "", "Azure: ClientID")
	fs.StringVar(&s.Azure.ClientSecret, "azure-client-secret", "", "Azure: ClientSecret")
	fs.StringVar(&s.Azure.TenantID, "azure-tenant-id", "", "Azure: TenantID")
	fs.StringVar(&s.Azure.SubscriptionID, "azure-subscription-id", "", "Azure: SubscriptionID")
	fs.StringVar(&s.Azure.KKPDatacenter, "azure-kkp-datacenter", "", "Azure: KKP datacenter to use")
	fs.StringVar(&s.GCP.ServiceAccount, "gcp-service-account", "", "GCP: Service Account")
	fs.StringVar(&s.GCP.Network, "gcp-network", "", "GCP: Network")
	fs.StringVar(&s.GCP.Subnetwork, "gcp-subnetwork", "", "GCP: Subnetwork")
	fs.StringVar(&s.GCP.KKPDatacenter, "gcp-kkp-datacenter", "", "GCP: KKP datacenter to use")
	fs.StringVar(&kubevirtKubeconfigFile, "kubevirt-kubeconfig", "", "Kubevirt: Cluster Kubeconfig filename")
	fs.StringVar(&s.Kubevirt.KKPDatacenter, "kubevirt-kkp-datacenter", "", "Kubevirt: KKP datacenter to use")
	fs.StringVar(&s.Alibaba.AccessKeyID, "alibaba-access-key-id", "", "Alibaba: AccessKeyID")
	fs.StringVar(&s.Alibaba.AccessKeySecret, "alibaba-access-key-secret", "", "Alibaba: AccessKeySecret")
	fs.StringVar(&s.Alibaba.KKPDatacenter, "alibaba-kkp-datacenter", "", "Alibaba: KKP datacenter to use")
	fs.StringVar(&s.Nutanix.Username, "nutanix-username", "", "Nutanix: Username")
	fs.StringVar(&s.Nutanix.Password, "nutanix-password", "", "Nutanix: Password")
	fs.StringVar(&s.Nutanix.CSIUsername, "nutanix-csi-username", "", "Nutanix CSI Prism Element: Username")
	fs.StringVar(&s.Nutanix.CSIPassword, "nutanix-csi-password", "", "Nutanix CSI Prism Element: Password")
	fs.StringVar(&s.Nutanix.CSIEndpoint, "nutanix-csi-endpoint", "", "Nutanix CSI Prism Element: Endpoint")
	fs.StringVar(&s.Nutanix.ProxyURL, "nutanix-proxy-url", "", "Nutanix: HTTP Proxy URL to access endpoint")
	fs.StringVar(&s.Nutanix.ClusterName, "nutanix-cluster-name", "", "Nutanix: Cluster Name")
	fs.StringVar(&s.Nutanix.ProjectName, "nutanix-project-name", "", "Nutanix: Project Name")
	fs.StringVar(&s.Nutanix.SubnetName, "nutanix-subnet-name", "", "Nutanix: Subnet Name")
	fs.StringVar(&s.Nutanix.KKPDatacenter, "nutanix-kkp-datacenter", "", "Nutanix: KKP datacenter to use")
	fs.StringVar(&s.VMwareCloudDirector.Username, "vmware-cloud-director-username", "", "VMware Cloud Director: Username")
	fs.StringVar(&s.VMwareCloudDirector.Password, "vmware-cloud-director-password", "", "VMware Cloud Director: Password")
	fs.StringVar(&s.VMwareCloudDirector.Organization, "vmware-cloud-director-organization", "", "VMware Cloud Director: Organization")
	fs.StringVar(&s.VMwareCloudDirector.VDC, "vmware-cloud-director-vdc", "", "VMware Cloud Director: Organizational VDC")
	fs.StringVar(&ovdcNetworks, "vmware-cloud-director-ovdc-networks", "", "VMware Cloud Director: Organizational VDC networks; comma separated list of network names")
	fs.StringVar(&s.VMwareCloudDirector.KKPDatacenter, "vmware-cloud-director-kkp-datacenter", "", "VMware Cloud Director: KKP datacenter to use")
	fs.StringVar(&s.RHEL.SubscriptionUser, "rhel-subscription-user", "", "RedHat Enterprise subscription user")
	fs.StringVar(&s.RHEL.SubscriptionPassword, "rhel-subscription-password", "", "RedHat Enterprise subscription password")
	fs.StringVar(&s.RHEL.OfflineToken, "rhel-offline-token", "", "RedHat Enterprise offlien token")
}

func (o *Options) ParseFlags(log *zap.SugaredLogger) error {
	if o.ExistingClusterLabel != "" && o.ClusterParallelCount != 1 {
		return errors.New("-cluster-parallel-count must be 1 when testing an existing cluster")
	}

	if !sets.New("api", "kube").Has(o.Client) {
		return fmt.Errorf("invalid -client option %q", o.Client)
	}

	o.Providers = AllProviders.Intersection(o.Providers)
	if o.Providers.Len() == 0 {
		return errors.New("no cloud provider was enabled")
	}

	var err error
	o.Tests, err = o.effectiveTests()
	if err != nil {
		return err
	}

	if o.Tests.Len() == 0 {
		log.Warn("All tests have been disabled, will only test cluster creation and whether nodes come up successfully.")
	}

	for _, release := range sets.List(o.Releases) {
		version := test.LatestKubernetesVersionForRelease(release, nil)
		if version == nil {
			return fmt.Errorf("no version found for release %q", release)
		}

		o.Versions = append(o.Versions, version)
	}

	// periodics do not specify a version at all and just rely on us auto-determining
	// the most recent stable (stable = latest-1) supported Kubernetes version
	if len(o.Versions) == 0 {
		o.Versions = append(o.Versions, test.LatestStableKubernetesVersion(nil))
		log.Infow("No -releases specified, defaulting to latest stable Kubernetes version", "version", o.Versions[0])
	}

	o.Distributions, err = o.effectiveDistributions()
	if err != nil {
		return err
	}

	if pubKeyPath != "" {
		keyData, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load SSH key: %w", err)
		}

		o.PublicKeys = [][]byte{keyData}
	}

	if o.LogDirectory != "" {
		if _, err := os.Stat(o.LogDirectory); err != nil {
			if err := os.MkdirAll(o.LogDirectory, 0755); err != nil {
				return fmt.Errorf("log-directory %q is not a valid directory and could not be created", o.LogDirectory)
			}
		}
	}

	if err := o.Secrets.ParseFlags(); err != nil {
		return err
	}

	return nil
}

func (s *Secrets) ParseFlags() error {
	if kubevirtKubeconfigFile != "" {
		content, err := os.ReadFile(kubevirtKubeconfigFile)
		if err != nil {
			return fmt.Errorf("failed to read kubevirt kubeconfig file: %w", err)
		}

		s.Kubevirt.Kubeconfig = test.SafeBase64Decoding(string(content))
	}

	if ovdcNetworks != "" {
		s.VMwareCloudDirector.OVDCNetworks = strings.Split(ovdcNetworks, ",")
	}

	if s.GCP.ServiceAccount != "" {
		s.GCP.ServiceAccount = test.SafeBase64Decoding(s.GCP.ServiceAccount)
	}

	return nil
}

func (o *Options) effectiveDistributions() (sets.Set[string], error) {
	all := sets.New[string]()
	for _, os := range providerconfig.AllOperatingSystems {
		all.Insert(string(os))
	}

	return combineSets(o.EnableDistributions, o.ExcludeDistributions, all, "distributions")
}

func (o *Options) effectiveTests() (sets.Set[string], error) {
	// Do not force all scripts to keep a list of all tests, just default to running all tests
	// when no relevant CLI flag was given.
	if o.EnableTests.Len() == 0 && o.ExcludeTests.Len() == 0 {
		return AllTests, nil
	}

	// Make it more comfortable to disable all custom tests.
	if o.EnableTests.Len() == 0 && o.ExcludeTests.Has("all") {
		return sets.New[string](), nil
	}

	return combineSets(o.EnableTests, o.ExcludeTests, AllTests, "tests")
}

func combineSets(include, exclude, all sets.Set[string], flagname string) (sets.Set[string], error) {
	if include.Len() == 0 && exclude.Len() == 0 {
		return nil, fmt.Errorf("either -%s or -exclude-%s must be given (each is a comma-separated list of %v)", flagname, flagname, sets.List(all))
	}

	if include.Len() > 0 && exclude.Len() > 0 {
		return nil, fmt.Errorf("-%s and -exclude-%s must not be given at the same time", flagname, flagname)
	}

	var chosen sets.Set[string]

	if include.Len() > 0 {
		chosen = include

		if unsupported := chosen.Difference(all); unsupported.Len() > 0 {
			return nil, fmt.Errorf("unknown %s: %v", flagname, sets.List(unsupported))
		}
	} else {
		chosen = all.Difference(exclude)
	}

	if chosen.Len() == 0 {
		return nil, fmt.Errorf("no %s to use in tests remained after evaluating -%s and -exclude-%s", flagname, flagname, flagname)
	}

	return chosen, nil
}
