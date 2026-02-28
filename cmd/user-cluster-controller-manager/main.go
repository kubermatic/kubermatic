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
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/zapr"
	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	applicationinstallationcontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/application-installation-controller"
	ccmcsimigrator "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/ccm-csi-migrator"
	clusterrolelabeler "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/cluster-role-labeler"
	constraintsyncer "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/constraint-syncer"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/flatcar"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/ipam"
	kvvmieviction "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/kubevirt-vmi-eviction"
	nodelabeler "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler"
	nodeversioncontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-version-controller"
	ownerbindingcreator "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/owner-binding-creator"
	rbacusercluster "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/rbac"
	usercluster "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources"
	envoyagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/envoy-agent"
	machinecontrollerresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine-controller"
	roleclonercontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/role-cloner-controller"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

type controllerRunOptions struct {
	metricsListenAddr                 string
	healthListenAddr                  string
	version                           string
	networks                          networkFlags
	namespace                         string
	clusterURL                        string
	clusterName                       string
	kasSecurePort                     int
	tunnelingAgentIP                  flagopts.IPValue
	overwriteRegistry                 string
	cloudProviderName                 string
	nodelabels                        string
	seedKubeconfig                    string
	ownerEmail                        string
	updateWindowStart                 string
	updateWindowLength                string
	dnsClusterIP                      string
	nodeLocalDNSCache                 bool
	opaIntegration                    bool
	opaEnableMutation                 bool
	opaWebhookTimeout                 int
	useSSHKeyAgent                    bool
	networkPolicies                   bool
	caBundleFile                      string
	mlaGatewayURL                     string
	userClusterLogging                bool
	userClusterMonitoring             bool
	monitoringAgentScrapeConfigPrefix string
	ccmMigration                      bool
	ccmMigrationCompleted             bool
	nutanixCSIEnabled                 bool
	isKonnectivityEnabled             bool
	konnectivityServerHost            string
	konnectivityServerPort            int
	konnectivityKeepaliveTime         string
	applicationCache                  string
	kubeVirtVMIEvictionController     bool
	kubeVirtInfraKubeconfig           string
	kubeVirtInfraNamespace            string

	clusterBackup clusterBackupOptions

	kyvernoEnabled bool
}

type clusterBackupOptions struct {
	backupStorageLocation string
	credentialSecret      string
}

func main() {
	runOp := controllerRunOptions{}
	klog.InitFlags(nil)
	pprofOpts := &flagopts.PProf{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	addFlags(flag.CommandLine, &runOp)

	flag.StringVar(&runOp.metricsListenAddr, "metrics-listen-address", "127.0.0.1:8085", "The address on which the internal HTTP /metrics server is running on")
	flag.StringVar(&runOp.healthListenAddr, "health-listen-address", "127.0.0.1:8086", "The address on which the internal HTTP /ready & /live server is running on")
	flag.StringVar(&runOp.version, "version", "", "The version of the cluster")
	flag.Var(&runOp.networks, "ipam-controller-network", "The networks from which the ipam controller should allocate IPs for machines (e.g.: .--ipam-controller-network=10.0.0.0/16,10.0.0.1,8.8.8.8 --ipam-controller-network=192.168.5.0/24,192.168.5.1,1.1.1.1,8.8.4.4)")
	flag.StringVar(&runOp.namespace, "namespace", "", "Namespace in which the cluster is running in")
	flag.StringVar(&runOp.clusterURL, "cluster-url", "", "Cluster URL")
	flag.StringVar(&runOp.clusterName, "cluster-name", "", "Cluster name")
	flag.StringVar(&runOp.dnsClusterIP, "dns-cluster-ip", "", "KubeDNS service IP for the cluster")
	flag.BoolVar(&runOp.nodeLocalDNSCache, "node-local-dns-cache", false, "Enable NodeLocal DNS Cache in user cluster")
	flag.IntVar(&runOp.kasSecurePort, "kas-secure-port", 6443, "Secure KAS port")
	flag.Var(&runOp.tunnelingAgentIP, "tunneling-agent-ip", "If specified the tunneling agent will bind to this IP address, otherwise it will not be deployed.")
	flag.StringVar(&runOp.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&runOp.cloudProviderName, "cloud-provider-name", "", "Name of the cloudprovider")
	flag.StringVar(&runOp.nodelabels, "node-labels", "", "A json-encoded map of node labels. If set, those labels will be enforced on all nodes.")
	flag.StringVar(&runOp.seedKubeconfig, "seed-kubeconfig", "", "Path to the seed kubeconfig. In-Cluster config will be used if unset")
	flag.StringVar(&runOp.ownerEmail, "owner-email", "", "An email address of the user who created the cluster. Used as default subject for the admin cluster role binding")
	flag.StringVar(&runOp.updateWindowStart, "update-window-start", "", "The start time of the update window, e.g. 02:00")
	flag.StringVar(&runOp.updateWindowLength, "update-window-length", "", "The length of the update window, e.g. 1h")
	flag.BoolVar(&runOp.opaIntegration, "opa-integration", false, "Enable OPA integration in user cluster")
	flag.BoolVar(&runOp.opaEnableMutation, "enable-mutation", false, "Enable OPA experimental mutation in user cluster")
	flag.IntVar(&runOp.opaWebhookTimeout, "opa-webhook-timeout", 1, "Timeout for OPA Integration validating webhook, in seconds")
	flag.BoolVar(&runOp.useSSHKeyAgent, "enable-ssh-key-agent", false, "Enable UserSSHKeyAgent integration in user cluster")
	flag.BoolVar(&runOp.networkPolicies, "enable-network-policies", false, "Enable deployment of network policies to kube-system namespace in user cluster")
	flag.StringVar(&runOp.caBundleFile, "ca-bundle", "", "The path to the cluster's CA bundle (PEM-encoded).")
	flag.StringVar(&runOp.mlaGatewayURL, "mla-gateway-url", "", "The URL of MLA (Monitoring, Logging, and Alerting) gateway endpoint.")
	flag.BoolVar(&runOp.userClusterLogging, "user-cluster-logging", false, "Enable logging in user cluster.")
	flag.BoolVar(&runOp.userClusterMonitoring, "user-cluster-monitoring", false, "Enable monitoring in user cluster.")
	flag.StringVar(&runOp.monitoringAgentScrapeConfigPrefix, "monitoring-agent-scrape-config-prefix", "monitoring-scraping", fmt.Sprintf("The name prefix of ConfigMaps in namespace %s, which will be used to add customized scrape configs for user cluster monitoring Agent.", resources.UserClusterMLANamespace))
	flag.BoolVar(&runOp.ccmMigration, "ccm-migration", false, "Enable ccm migration in user cluster.")
	flag.BoolVar(&runOp.ccmMigrationCompleted, "ccm-migration-completed", false, "cluster has been successfully migrated.")
	flag.BoolVar(&runOp.nutanixCSIEnabled, "nutanix-csi-enabled", false, "enable Nutanix CSI")
	flag.BoolVar(&runOp.isKonnectivityEnabled, "konnectivity-enabled", false, "Enable Konnectivity.")
	flag.StringVar(&runOp.konnectivityServerHost, "konnectivity-server-host", "", "Konnectivity Server host.")
	flag.IntVar(&runOp.konnectivityServerPort, "konnectivity-server-port", 6443, "Konnectivity Server port.")
	flag.StringVar(&runOp.konnectivityKeepaliveTime, "konnectivity-keepalive-time", "1m", "Konnectivity keepalive time.")
	flag.StringVar(&runOp.applicationCache, "application-cache", "", "Path to Application cache directory.")
	flag.BoolVar(&runOp.kubeVirtVMIEvictionController, "kv-vmi-eviction-controller", false, "Start the KubeVirt VMI eviction controller")
	flag.StringVar(&runOp.kubeVirtInfraKubeconfig, "kv-infra-kubeconfig", "", "Path to the KubeVirt infra kubeconfig.")
	flag.StringVar(&runOp.kubeVirtInfraNamespace, "kv-infra-namespace", "", "Kubevirt infra namespace where workload will be deployed")
	flag.BoolVar(&runOp.kyvernoEnabled, "kyverno-enabled", false, "Enable Kyverno in user cluster.")
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	// make sure the logging flags actually affect the global (deprecated) logger instance
	kubermaticlog.Logger = log
	reconciling.Configure(log)

	versions := kubermatic.GetVersions()
	cli.Hello(log, "User-Cluster Controller-Manager", &versions)

	kubeconfigFlag := flag.Lookup("kubeconfig")
	if kubeconfigFlag == nil { // Should not be possible.
		log.Fatal("can not get kubeconfig flag")
	}

	if runOp.namespace == "" {
		log.Fatal("-namespace must be set")
	}
	if runOp.clusterURL == "" {
		log.Fatal("-cluster-url must be set")
	}
	if runOp.clusterName == "" {
		log.Fatal("-cluster-name must be set")
	}
	if runOp.dnsClusterIP == "" {
		log.Fatal("-dns-cluster-ip must be set")
	}
	if runOp.kasSecurePort == int(envoyagent.StatsPort) {
		log.Fatalf("-kas-secure-port \"%d\" is reserved and must not be used", runOp.kasSecurePort)
	}
	clusterURL, err := url.Parse(runOp.clusterURL)
	if err != nil {
		log.Fatalw("Failed parsing clusterURL", zap.Error(err))
	}
	if runOp.isKonnectivityEnabled && runOp.konnectivityServerHost == "" {
		log.Fatal("-konnectivity-server-host must be set when Konnectivity is enabled")
	}
	if len(runOp.caBundleFile) == 0 {
		log.Fatal("-ca-bundle must be set")
	}
	if runOp.userClusterLogging || runOp.userClusterMonitoring {
		if runOp.mlaGatewayURL == "" {
			log.Fatal("-mla-gateway-url must be set when enabling user cluster logging or monitoring")
		}
	}

	if runOp.applicationCache == "" {
		log.Fatal("application-cache must be set")
	}

	nodeLabels := map[string]string{}
	if runOp.nodelabels != "" {
		if err := json.Unmarshal([]byte(runOp.nodelabels), &nodeLabels); err != nil {
			log.Fatalw("Failed to unmarshal value of --node-labels arg", zap.Error(err))
		}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalw("Failed getting user cluster controller config", zap.Error(err))
	}

	caBundle, err := certificates.NewCABundleFromFile(runOp.caBundleFile)
	if err != nil {
		log.Fatalw("Failed loading CA bundle", zap.Error(err))
	}

	rootCtx := signals.SetupSignalHandler()

	mgr, err := manager.New(cfg, manager.Options{
		BaseContext: func() context.Context {
			return rootCtx
		},
		LeaderElection:          true,
		LeaderElectionNamespace: metav1.NamespaceSystem,
		LeaderElectionID:        "user-cluster-controller-leader-lock",
		Metrics:                 metricsserver.Options{BindAddress: runOp.metricsListenAddr},
		HealthProbeBindAddress:  runOp.healthListenAddr,
		PprofBindAddress:        pprofOpts.ListenAddress,
	})
	if err != nil {
		log.Fatalw("Failed creating user cluster manager", zap.Error(err))
	}

	var seedConfig *rest.Config
	if runOp.seedKubeconfig != "" {
		seedConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: runOp.seedKubeconfig},
			&clientcmd.ConfigOverrides{}).ClientConfig()
	} else {
		seedConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalw("Failed to get seed kubeconfig", zap.Error(err))
	}

	seedMgr, err := setupSeedManager(seedConfig, runOp)
	if err != nil {
		log.Fatalw("Failed to construct seed mgr", zap.Error(err))
	}
	if err := mgr.Add(seedMgr); err != nil {
		log.Fatalw("Failed to add seed mgr to main mgr", zap.Error(err))
	}

	log.Info("registering components")
	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionsv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := apiregistrationv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiregistrationv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := osmv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", osmv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := velerov1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", velerov1.SchemeGroupVersion), zap.Error(err))
	}
	if err := kyvernov1.Install(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", kyvernov1.SchemeGroupVersion), zap.Error(err))
	}

	isPausedChecker := userclustercontrollermanager.NewClusterPausedChecker(seedMgr.GetClient(), runOp.clusterName)

	// Setup all Controllers
	log.Info("registering controllers")
	if err := usercluster.Add(mgr,
		seedMgr,
		runOp.version,
		runOp.namespace,
		runOp.cloudProviderName,
		clusterURL,
		isPausedChecker,
		runOp.overwriteRegistry,
		uint32(runOp.kasSecurePort),
		runOp.tunnelingAgentIP.IP,
		mgr.AddReadyzCheck,
		runOp.dnsClusterIP,
		runOp.nodeLocalDNSCache,
		runOp.opaIntegration,
		runOp.opaEnableMutation,
		versions,
		runOp.useSSHKeyAgent,
		runOp.networkPolicies,
		runOp.opaWebhookTimeout,
		caBundle,
		usercluster.UserClusterMLA{
			Logging:                           runOp.userClusterLogging,
			Monitoring:                        runOp.userClusterMonitoring,
			MLAGatewayURL:                     runOp.mlaGatewayURL,
			MonitoringAgentScrapeConfigPrefix: runOp.monitoringAgentScrapeConfigPrefix,
		},
		runOp.clusterName,
		runOp.nutanixCSIEnabled,
		runOp.isKonnectivityEnabled,
		runOp.konnectivityServerHost,
		runOp.konnectivityServerPort,
		runOp.konnectivityKeepaliveTime,
		runOp.ccmMigration,
		runOp.ccmMigrationCompleted,
		runOp.kyvernoEnabled,
		log,
	); err != nil {
		log.Fatalw("Failed to register user cluster controller", zap.Error(err))
	}
	log.Info("Registered usercluster controller")

	if len(runOp.networks) > 0 {
		// We need to add the machine CRDs once here, because otherwise the IPAM
		// controller keeps the manager from starting as it can not establish a
		// watch for machine CRs, keeping us from creating them
		creators := []reconciling.NamedCustomResourceDefinitionReconcilerFactory{
			machinecontrollerresources.MachineCRDReconciler(),
		}
		if err := reconciling.ReconcileCustomResourceDefinitions(rootCtx, creators, "", mgr.GetClient()); err != nil {
			// The mgr.Client is uninitianlized here and hence always returns a 404, regardless of the object existing or not
			if !strings.Contains(err.Error(), `customresourcedefinitions.apiextensions.k8s.io "machines.cluster.k8s.io" already exists`) {
				log.Fatalw("Failed to initially create the Machine CR", zap.Error(err))
			}
		}

		// IPAM is critical for a cluster, so the isPauseChecker is not used in this controller
		if err := ipam.Add(mgr, runOp.networks, log); err != nil {
			log.Fatalw("Failed to add IPAM controller to mgr", zap.Error(err))
		}
		log.Infof("Added IPAM controller to mgr")
	}

	if err := rbacusercluster.Add(mgr, log, mgr.AddReadyzCheck, isPausedChecker); err != nil {
		log.Fatalw("Failed to add user RBAC controller to mgr", zap.Error(err))
	}
	log.Info("Registered user RBAC controller")

	updateWindow := kubermaticv1.UpdateWindow{
		Start:  runOp.updateWindowStart,
		Length: runOp.updateWindowLength,
	}
	if err := flatcar.Add(mgr, runOp.overwriteRegistry, updateWindow, isPausedChecker); err != nil {
		log.Fatalw("Failed to register the Flatcar controller", zap.Error(err))
	}
	log.Info("Registered Flatcar controller")

	// node labels can be critical to a cluster functioning, so we do not stop applying
	// labels once a cluster is paused, hence no isPausedChecker here
	if err := nodelabeler.Add(rootCtx, log, mgr, nodeLabels); err != nil {
		log.Fatalw("Failed to register nodelabel controller", zap.Error(err))
	}
	log.Info("Registered nodelabel controller")

	if err := nodeversioncontroller.Add(rootCtx, log, seedMgr, mgr, runOp.clusterName); err != nil {
		log.Fatalw("Failed to register node-version controller", zap.Error(err))
	}
	log.Info("Registered node-version controller")

	if err := clusterrolelabeler.Add(rootCtx, log, mgr, isPausedChecker); err != nil {
		log.Fatalw("Failed to register clusterrolelabeler controller", zap.Error(err))
	}
	log.Info("Registered clusterrolelabeler controller")

	if err := roleclonercontroller.Add(log, mgr, isPausedChecker); err != nil {
		log.Fatalw("Failed to register role-cloner controller", zap.Error(err))
	}
	log.Info("Registered role-cloner controller")

	if runOp.ownerEmail != "" {
		if err := ownerbindingcreator.Add(log, mgr, runOp.ownerEmail, isPausedChecker); err != nil {
			log.Fatalw("Failed to register owner-binding-creator controller", zap.Error(err))
		}
		log.Info("Registered owner-binding-creator controller")
	} else {
		log.Info("No -owner-email given, skipping owner-binding-creator controller")
	}

	if runOp.ccmMigration {
		if err := ccmcsimigrator.Add(log, seedMgr, mgr, versions, runOp.clusterName, isPausedChecker); err != nil {
			log.Fatalw("failed to register ccm-csi-migrator controller", zap.Error(err))
		}
		log.Info("registered ccm-csi-migrator controller")
	}

	if runOp.opaIntegration {
		if err := constraintsyncer.Add(rootCtx, log, seedMgr, mgr, runOp.namespace, isPausedChecker); err != nil {
			log.Fatalw("Failed to register constraintsyncer controller", zap.Error(err))
		}
		log.Info("Registered constraintsyncer controller")
	}

	if err := applicationinstallationcontroller.Add(rootCtx, log, seedMgr, mgr, isPausedChecker, runOp.namespace, runOp.overwriteRegistry, &applications.ApplicationManager{ApplicationCache: runOp.applicationCache, Kubeconfig: kubeconfigFlag.Value.String(), SecretNamespace: runOp.namespace, ClusterName: runOp.clusterName}); err != nil {
		log.Fatalw("Failed to add user Application Installation controller to mgr", zap.Error(err))
	}
	log.Info("Registered Application Installation controller")

	if err := setupControllers(log, seedMgr, mgr, runOp.clusterName, versions, runOp.overwriteRegistry, caBundle, isPausedChecker, runOp.namespace, runOp.kyvernoEnabled); err != nil {
		log.Fatalw("Failed to add controllers to mgr", zap.Error(err))
	}

	// KubeVirt infra
	if runOp.kubeVirtVMIEvictionController {
		var kubevirtInfraConfig *rest.Config
		kubevirtInfraConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: runOp.kubeVirtInfraKubeconfig},
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			log.Fatalw("Failed to get KubeVirt infra kubeconfig", zap.Error(err))
		}
		// VM and VMIs are created in a namespace having the same name as the cluster namespace name.
		kvInfraNamespace := runOp.namespace
		// If kv-infra-namespace flag is set, that means Kubevirt is running in the namespaced mode, so all workloads will be deployed in the specified namespace.
		if runOp.kubeVirtInfraNamespace != "" {
			kvInfraNamespace = runOp.kubeVirtInfraNamespace
		}
		kubevirtInfraMgr, err := manager.New(kubevirtInfraConfig, manager.Options{
			LeaderElection: false,
			Metrics:        metricsserver.Options{BindAddress: "0"},
			Cache: cache.Options{
				DefaultNamespaces: map[string]cache.Config{
					kvInfraNamespace: {},
				},
			},
		})
		if err != nil {
			log.Fatalw("Failed to construct kubevirt infra mgr", zap.Error(err))
		}
		if err := mgr.Add(kubevirtInfraMgr); err != nil {
			log.Fatalw("Failed to add kubevirt infra mgr to main mgr", zap.Error(err))
		}

		if err := kvvmieviction.Add(rootCtx, log, mgr, kubevirtInfraMgr, isPausedChecker, runOp.clusterName); err != nil {
			log.Fatalw("Failed to register kubevirt-vmi-eviction controller", zap.Error(err))
		}
		log.Info("Registered kubevirt-vmi-eviction controller")
	}

	if err := mgr.Start(rootCtx); err != nil {
		log.Fatalw("Failed running manager", zap.Error(err))
	}
}
