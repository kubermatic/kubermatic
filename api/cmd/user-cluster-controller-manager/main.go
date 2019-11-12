package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	"github.com/heptiolabs/healthcheck"
	"github.com/oklog/run"
	"go.uber.org/zap"

	containerlinux "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/container-linux"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/ipam"
	nodelabeler "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/node-labeler"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/nodecsrapprover"
	openshiftmasternodelabeler "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/openshift-master-node-labeler"
	openshiftseedsyncer "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/openshift-seed-syncer"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/rbac"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/resources"
	machinecontrolerresources "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/resources/resources/machine-controller"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

type controllerRunOptions struct {
	metricsListenAddr             string
	healthListenAddr              string
	openshift                     bool
	version                       string
	networks                      networkFlags
	namespace                     string
	clusterURL                    string
	openvpnServerPort             int
	overwriteRegistry             string
	cloudProviderName             string
	cloudCredentialSecretTemplate string
	nodelabels                    string
	seedKubeconfig                string
	log                           kubermaticlog.Options
}

func main() {
	runOp := controllerRunOptions{}
	klog.InitFlags(nil)
	flag.StringVar(&runOp.metricsListenAddr, "metrics-listen-address", "127.0.0.1:8085", "The address on which the internal HTTP /metrics server is running on")
	flag.StringVar(&runOp.healthListenAddr, "health-listen-address", "127.0.0.1:8086", "The address on which the internal HTTP /ready & /live server is running on")
	flag.BoolVar(&runOp.openshift, "openshift", false, "Whether the managed cluster is an openshift cluster")
	flag.StringVar(&runOp.version, "version", "", "The version of the cluster")
	flag.Var(&runOp.networks, "ipam-controller-network", "The networks from which the ipam controller should allocate IPs for machines (e.g.: .--ipam-controller-network=10.0.0.0/16,10.0.0.1,8.8.8.8 --ipam-controller-network=192.168.5.0/24,192.168.5.1,1.1.1.1,8.8.4.4)")
	flag.StringVar(&runOp.namespace, "namespace", "", "Namespace in which the cluster is running in")
	flag.StringVar(&runOp.clusterURL, "cluster-url", "", "Cluster URL")
	flag.IntVar(&runOp.openvpnServerPort, "openvpn-server-port", 0, "OpenVPN server port")
	flag.StringVar(&runOp.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.BoolVar(&runOp.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&runOp.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.StringVar(&runOp.cloudProviderName, "cloud-provider-name", "", "Name of the cloudprovider")
	flag.StringVar(&runOp.cloudCredentialSecretTemplate, "cloud-credential-secret-template", "", "A serialized Kubernetes secret whose Name and Data fields will be used to create a secret for the openshift cloud credentials operator.")
	flag.StringVar(&runOp.nodelabels, "node-labels", "", "A json-encoded map of node labels. If set, those labels will be enforced on all nodes.")
	flag.StringVar(&runOp.seedKubeconfig, "seed-kubeconfig", "", "Path to the seed kubeconfig. In-Cluster config will be used if unset")

	flag.Parse()

	if err := runOp.log.Validate(); err != nil {
		fmt.Printf("error occurred while validating zap logger options: %v\n", err)
		os.Exit(1)
	}

	rawLog := kubermaticlog.New(runOp.log.Debug, kubermaticlog.Format(runOp.log.Format))
	log := rawLog.Sugar()

	if runOp.namespace == "" {
		log.Fatal("-namespace must be set")
	}
	if runOp.clusterURL == "" {
		log.Fatal("-cluster-url must be set")
	}
	clusterURL, err := url.Parse(runOp.clusterURL)
	if err != nil {
		log.Fatalw("Failed parsing clusterURL", zap.Error(err))
	}
	if runOp.openvpnServerPort == 0 {
		log.Fatal("-openvpn-server-port must be set")
	}

	var cloudCredentialSecretTemplate *corev1.Secret
	if runOp.cloudCredentialSecretTemplate != "" {
		cloudCredentialSecretTemplate = &corev1.Secret{}
		if err := json.Unmarshal([]byte(runOp.cloudCredentialSecretTemplate), cloudCredentialSecretTemplate); err != nil {
			log.Fatalw("Failed to unmarshal value of --cloud-credential-secret-template flag into secret", zap.Error(err))
		}
	}

	nodeLabels := map[string]string{}
	if runOp.nodelabels != "" {
		if err := json.Unmarshal([]byte(runOp.nodelabels), &nodeLabels); err != nil {
			log.Fatalw("Failed to unmarshal value of --node-labels arg", zap.Error(err))
		}
	}

	var g run.Group

	healthHandler := healthcheck.NewHandler()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalw("Failed getting user cluster controller config", zap.Error(err))
	}
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:          true,
		LeaderElectionNamespace: metav1.NamespaceSystem,
		LeaderElectionID:        "user-cluster-controller-leader-lock",
		MetricsBindAddress:      runOp.metricsListenAddr,
	})
	if err != nil {
		log.Fatalw("Failed creating user cluster controller", zap.Error(err))
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
	if err := apiextensionv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionv1beta1.SchemeGroupVersion), zap.Error(err))
	}
	if err := apiregistrationv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiregistrationv1beta1.SchemeGroupVersion), zap.Error(err))
	}

	// Setup all Controllers
	log.Info("registering controllers")
	if err := usercluster.Add(mgr,
		seedMgr,
		runOp.openshift,
		runOp.version,
		runOp.namespace,
		runOp.cloudProviderName,
		clusterURL,
		runOp.openvpnServerPort,
		healthHandler.AddReadinessCheck,
		cloudCredentialSecretTemplate,
		log); err != nil {
		log.Fatalw("Failed to register user cluster controller", zap.Error(err))
	}
	log.Info("Registered usercluster controller")

	if len(runOp.networks) > 0 {
		if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			log.Fatalw("Failed to add clusterv1alpha1 scheme", zap.Error(err))
		}
		// We need to add the machine CRDs once here, because otherwise the IPAM
		// controller keeps the manager from starting as it can not establish a
		// watch for machine CRs, keeping us from creating them
		creators := []reconciling.NamedCustomResourceDefinitionCreatorGetter{
			machinecontrolerresources.MachineCRDCreator(),
		}
		if err := reconciling.ReconcileCustomResourceDefinitions(context.Background(), creators, "", mgr.GetClient()); err != nil {
			// The mgr.Client is uninitianlized here and hence always returns a 404, regardless of the object existing or not
			if !strings.Contains(err.Error(), `customresourcedefinitions.apiextensions.k8s.io "machines.cluster.k8s.io" already exists`) {
				log.Fatalw("Failed to initially create the Machine CR", zap.Error(err))
			}
		}
		if err := ipam.Add(mgr, runOp.networks, log); err != nil {
			log.Fatalw("Failed to add IPAM controller to mgr", zap.Error(err))
		}
		log.Infof("Added IPAM controller to mgr")
	}

	if err := rbacusercluster.Add(mgr, healthHandler.AddReadinessCheck); err != nil {
		log.Fatalw("Failed to add user RBAC controller to mgr", zap.Error(err))
	}
	log.Info("Registered user RBAC controller")

	if runOp.openshift {
		if err := nodecsrapprover.Add(mgr, 4, cfg, log); err != nil {
			log.Fatalw("Failed to add nodecsrapprover controller", zap.Error(err))
		}
		if err := openshiftmasternodelabeler.Add(context.Background(), kubermaticlog.Logger, mgr); err != nil {
			log.Fatalw("Failed to add openshiftmasternodelabeler controller", zap.Error(err))
		}
		log.Info("Registered nodecsrapprover controller")
		if err := openshiftseedsyncer.Add(log, mgr, seedMgr, runOp.clusterURL, runOp.namespace); err != nil {
			log.Fatalw("Failed to register the openshiftseedsyncer", zap.Error(err))
		}
		log.Info("Registered openshiftseedsyncer controller")
	}

	if err := containerlinux.Add(mgr, runOp.overwriteRegistry); err != nil {
		log.Fatalw("Failed to register the ContainerLinux controller", zap.Error(err))
	}
	log.Info("Registered ContainerLinux controller")

	if err := nodelabeler.Add(ctx, log, mgr, nodeLabels); err != nil {
		log.Fatalw("Failed to register nodelabel controller", zap.Error(err))
	}
	log.Info("Registered nodelabel controller")

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("user requested to stop the application")
			case <-done:
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxDone()
		})
	}

	// This group starts the controller manager
	{
		g.Add(func() error {
			// Start the Cmd
			return mgr.Start(done)
		}, func(err error) {
			log.Infow("stopping user cluster controller manager", zap.Error(err))
		})
	}

	// This group starts the readiness & liveness http server
	{
		h := &http.Server{Addr: runOp.healthListenAddr, Handler: healthHandler}
		g.Add(h.ListenAndServe, func(err error) {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := h.Shutdown(shutdownCtx); err != nil {
				log.Errorw("Healthcheck handler terminated with an error", zap.Error(err))
			}
		})
	}

	if err := g.Run(); err != nil {
		log.Fatalw("Failed running user cluster controller", zap.Error(err))
	}

}
