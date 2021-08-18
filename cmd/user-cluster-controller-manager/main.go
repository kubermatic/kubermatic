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
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	ccmcsimigrator "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/ccm-csi-migrator"
	clusterrolelabeler "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/cluster-role-labeler"
	constraintsyncer "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/constraint-syncer"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/flatcar"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/ipam"
	nodelabeler "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler"
	ownerbindingcreator "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/owner-binding-creator"
	rbacusercluster "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/rbac"
	usercluster "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources"
	machinecontrolerresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine-controller"
	rolecloner "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/role-cloner"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type controllerRunOptions struct {
	metricsListenAddr            string
	healthListenAddr             string
	version                      string
	networks                     networkFlags
	namespace                    string
	clusterURL                   string
	clusterName                  string
	openvpnServerPort            int
	kasSecurePort                int
	tunnelingAgentIP             flagopts.IPValue
	overwriteRegistry            string
	cloudProviderName            string
	nodelabels                   string
	seedKubeconfig               string
	ownerEmail                   string
	updateWindowStart            string
	updateWindowLength           string
	dnsClusterIP                 string
	nodeLocalDNSCache            bool
	opaIntegration               bool
	opaWebhookTimeout            int
	useSSHKeyAgent               bool
	caBundleFile                 string
	mlaGatewayURL                string
	userClusterLogging           bool
	userClusterMonitoring        bool
	prometheusScrapeConfigPrefix string
	ccmMigration                 bool
}

func main() {
	runOp := controllerRunOptions{}
	klog.InitFlags(nil)
	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	flag.StringVar(&runOp.metricsListenAddr, "metrics-listen-address", "127.0.0.1:8085", "The address on which the internal HTTP /metrics server is running on")
	flag.StringVar(&runOp.healthListenAddr, "health-listen-address", "127.0.0.1:8086", "The address on which the internal HTTP /ready & /live server is running on")
	flag.StringVar(&runOp.version, "version", "", "The version of the cluster")
	flag.Var(&runOp.networks, "ipam-controller-network", "The networks from which the ipam controller should allocate IPs for machines (e.g.: .--ipam-controller-network=10.0.0.0/16,10.0.0.1,8.8.8.8 --ipam-controller-network=192.168.5.0/24,192.168.5.1,1.1.1.1,8.8.4.4)")
	flag.StringVar(&runOp.namespace, "namespace", "", "Namespace in which the cluster is running in")
	flag.StringVar(&runOp.clusterURL, "cluster-url", "", "Cluster URL")
	flag.StringVar(&runOp.clusterName, "cluster-name", "", "Cluster name")
	flag.StringVar(&runOp.dnsClusterIP, "dns-cluster-ip", "", "KubeDNS service IP for the cluster")
	flag.BoolVar(&runOp.nodeLocalDNSCache, "node-local-dns-cache", false, "Enable NodeLocal DNS Cache in user cluster")
	flag.IntVar(&runOp.openvpnServerPort, "openvpn-server-port", 0, "OpenVPN server port")
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
	flag.IntVar(&runOp.opaWebhookTimeout, "opa-webhook-timeout", 3, "Timeout for OPA Integration validating webhook, in seconds")
	flag.BoolVar(&runOp.useSSHKeyAgent, "enable-ssh-key-agent", false, "Enable UserSSHKeyAgent integration in user cluster")
	flag.StringVar(&runOp.caBundleFile, "ca-bundle", "", "The path to the cluster's CA bundle (PEM-encoded).")
	flag.StringVar(&runOp.mlaGatewayURL, "mla-gateway-url", "", "The URL of MLA (Monitoring, Logging, and Alerting) gateway endpoint.")
	flag.BoolVar(&runOp.userClusterLogging, "user-cluster-logging", false, "Enable logging in user cluster.")
	flag.BoolVar(&runOp.userClusterMonitoring, "user-cluster-monitoring", false, "Enable monitoring in user cluster.")
	flag.StringVar(&runOp.prometheusScrapeConfigPrefix, "prometheus-scrape-config-prefix", "prometheus-scraping", fmt.Sprintf("The name prefix of ConfigMaps in namespace %s, which will be used to add customized scrape configs for user cluster Prometheus.", resources.UserClusterMLANamespace))
	flag.BoolVar(&runOp.ccmMigration, "ccm-migration", false, "Enable ccm migration in user cluster.")

	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	versions := kubermatic.NewDefaultVersions()
	cli.Hello(log, "User-Cluster Controller-Manager", logOpts.Debug, &versions)

	if runOp.ownerEmail == "" {
		log.Fatal("-owner-email must be set")
	}
	if runOp.namespace == "" {
		log.Fatal("-namespace must be set")
	}
	if runOp.clusterURL == "" {
		log.Fatal("-cluster-url must be set")
	}
	if runOp.ccmMigration && runOp.clusterName == "" {
		log.Fatal("-cluster-name must be set")
	}
	if runOp.dnsClusterIP == "" {
		log.Fatal("-dns-cluster-ip must be set")
	}
	clusterURL, err := url.Parse(runOp.clusterURL)
	if err != nil {
		log.Fatalw("Failed parsing clusterURL", zap.Error(err))
	}
	if runOp.openvpnServerPort == 0 {
		log.Fatal("-openvpn-server-port must be set")
	}
	if len(runOp.caBundleFile) == 0 {
		log.Fatal("-ca-bundle must be set")
	}
	if runOp.userClusterLogging || runOp.userClusterMonitoring {
		if runOp.mlaGatewayURL == "" {
			log.Fatal("-mla-gateway-url must be set when enabling user cluster logging or monitoring")
		}
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

	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:          true,
		LeaderElectionNamespace: metav1.NamespaceSystem,
		LeaderElectionID:        "user-cluster-controller-leader-lock",
		MetricsBindAddress:      runOp.metricsListenAddr,
		HealthProbeBindAddress:  runOp.healthListenAddr,
	})
	if err != nil {
		log.Fatalw("Failed creating user cluster controller", zap.Error(err))
	}
	if err := mgr.Add(pprofOpts); err != nil {
		log.Fatalw("Failed to add pprof handler", zap.Error(err))
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
	seedMgr, err := manager.New(seedConfig, manager.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0",
		Namespace:          runOp.namespace,
	})
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
	if err := apiregistrationv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiregistrationv1beta1.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
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
		uint32(runOp.openvpnServerPort),
		uint32(runOp.kasSecurePort),
		runOp.tunnelingAgentIP.IP,
		mgr.AddReadyzCheck,
		runOp.dnsClusterIP,
		runOp.nodeLocalDNSCache,
		runOp.opaIntegration,
		versions,
		runOp.useSSHKeyAgent,
		runOp.opaWebhookTimeout,
		caBundle,
		usercluster.UserClusterMLA{
			Logging:                      runOp.userClusterLogging,
			Monitoring:                   runOp.userClusterMonitoring,
			MLAGatewayURL:                runOp.mlaGatewayURL,
			PrometheusScrapeConfigPrefix: runOp.prometheusScrapeConfigPrefix,
		},
		log,
	); err != nil {
		log.Fatalw("Failed to register user cluster controller", zap.Error(err))
	}
	log.Info("Registered usercluster controller")

	if len(runOp.networks) > 0 {
		// We need to add the machine CRDs once here, because otherwise the IPAM
		// controller keeps the manager from starting as it can not establish a
		// watch for machine CRs, keeping us from creating them
		creators := []reconciling.NamedCustomResourceDefinitionCreatorGetter{
			machinecontrolerresources.MachineCRDCreator(),
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

	if err := rbacusercluster.Add(mgr, mgr.AddReadyzCheck, isPausedChecker); err != nil {
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

	if err := clusterrolelabeler.Add(rootCtx, log, mgr, isPausedChecker); err != nil {
		log.Fatalw("Failed to register clusterrolelabeler controller", zap.Error(err))
	}
	log.Info("Registered clusterrolelabeler controller")

	if err := rolecloner.Add(rootCtx, log, mgr, isPausedChecker); err != nil {
		log.Fatalw("Failed to register rolecloner controller", zap.Error(err))
	}
	log.Info("Registered rolecloner controller")

	if err := ownerbindingcreator.Add(rootCtx, log, mgr, runOp.ownerEmail, isPausedChecker); err != nil {
		log.Fatalw("Failed to register ownerbindingcreator controller", zap.Error(err))
	}
	log.Info("Registered ownerbindingcreator controller")

	if runOp.ccmMigration {
		if err := ccmcsimigrator.Add(rootCtx, log, seedMgr, mgr, versions, runOp.clusterName, isPausedChecker); err != nil {
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

	if err := mgr.Start(rootCtx); err != nil {
		log.Fatalw("Failed running manager", zap.Error(err))
	}
}
