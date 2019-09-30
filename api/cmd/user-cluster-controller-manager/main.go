package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	"github.com/heptiolabs/healthcheck"
	"github.com/oklog/run"
	"go.uber.org/zap"

	containerlinux "github.com/kubermatic/kubermatic/api/pkg/controller/container-linux"
	"github.com/kubermatic/kubermatic/api/pkg/controller/ipam"
	"github.com/kubermatic/kubermatic/api/pkg/controller/nodecsrapprover"
	rbacusercluster "github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"
	openshiftmasternodelabeler "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/openshift-master-node-labeler"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster"
	machinecontrolerresources "github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/machine-controller"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

type controllerRunOptions struct {
	metricsListenAddr     string
	healthListenAddr      string
	openshift             bool
	version               string
	networks              networkFlags
	namespace             string
	caPath                string
	clusterURL            string
	openvpnServerPort     int
	openvpnCACertFilePath string
	openvpnCAKeyFilePath  string
	overwriteRegistry     string
	cloudProviderName     string
	log                   kubermaticlog.Options
}

func main() {
	runOp := controllerRunOptions{}
	flag.StringVar(&runOp.metricsListenAddr, "metrics-listen-address", "127.0.0.1:8085", "The address on which the internal HTTP /metrics server is running on")
	flag.StringVar(&runOp.healthListenAddr, "health-listen-address", "127.0.0.1:8086", "The address on which the internal HTTP /ready & /live server is running on")
	flag.BoolVar(&runOp.openshift, "openshift", false, "Whether the managed cluster is an openshift cluster")
	flag.StringVar(&runOp.version, "version", "", "The version of the cluster")
	flag.Var(&runOp.networks, "ipam-controller-network", "The networks from which the ipam controller should allocate IPs for machines (e.g.: .--ipam-controller-network=10.0.0.0/16,10.0.0.1,8.8.8.8 --ipam-controller-network=192.168.5.0/24,192.168.5.1,1.1.1.1,8.8.4.4)")
	flag.StringVar(&runOp.namespace, "namespace", "", "Namespace in which the cluster is running in")
	flag.StringVar(&runOp.caPath, "ca-cert", "ca.crt", "Path to the CA cert file")
	flag.StringVar(&runOp.clusterURL, "cluster-url", "", "Cluster URL")
	flag.IntVar(&runOp.openvpnServerPort, "openvpn-server-port", 0, "OpenVPN server port")
	flag.StringVar(&runOp.openvpnCACertFilePath, "openvpn-ca-cert-file", "", "Path to the OpenVPN CA cert file")
	flag.StringVar(&runOp.openvpnCAKeyFilePath, "openvpn-ca-key-file", "", "Path to the OpenVPN CA key file")
	flag.StringVar(&runOp.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.BoolVar(&runOp.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&runOp.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.StringVar(&runOp.cloudProviderName, "cloud-provider-name", "", "Name of the cloudprovider")

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
	if runOp.caPath == "" {
		log.Fatal("-ca-cert must be set")
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

	caBytes, err := ioutil.ReadFile(runOp.caPath)
	if err != nil {
		log.Fatalw("Failed to read CA cert", zap.Error(err))
	}
	certs, err := certutil.ParseCertsPEM(caBytes)
	if err != nil {
		log.Fatalw("Failed to parse certs", zap.Error(err))
	}
	if len(certs) != 1 {
		log.Fatalw("Did not find exactly one certificate in the given CA", "certificates-count", len(certs))
	}

	openVPNCACertBytes, err := ioutil.ReadFile(runOp.openvpnCACertFilePath)
	if err != nil {
		log.Fatalw("Failed to read openvpn-ca-cert-file", zap.Error(err))
	}
	openVPNCACerts, err := certutil.ParseCertsPEM(openVPNCACertBytes)
	if err != nil {
		log.Fatalw("Failed to parse openVPN CA file", zap.Error(err))
	}
	if certsLen := len(openVPNCACerts); certsLen != 1 {
		log.Fatalw("Did not find exactly one certificate in the openVPN CA file", "certificates-count", certsLen)
	}
	openVPNCAKeyBytes, err := ioutil.ReadFile(runOp.openvpnCAKeyFilePath)
	if err != nil {
		log.Fatalw("Failed to read openvpn-ca-key-file", zap.Error(err))
	}
	openVPNCAKey, err := certutil.ParsePrivateKeyPEM(openVPNCAKeyBytes)
	if err != nil {
		log.Fatalw("Failed to parse openVPN CA key file", zap.Error(err))
	}
	openVPNECSDAKey, isECDSAKey := openVPNCAKey.(*ecdsa.PrivateKey)
	if !isECDSAKey {
		log.Fatal("The openVPN private key is not an ECDSA key")
	}
	openVPNCACert := &resources.ECDSAKeyPair{Cert: openVPNCACerts[0], Key: openVPNECSDAKey}

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
		runOp.openshift,
		runOp.version,
		runOp.namespace,
		runOp.cloudProviderName,
		certs[0],
		clusterURL,
		runOp.openvpnServerPort,
		healthHandler.AddReadinessCheck,
		openVPNCACert,
		log); err != nil {
		log.Fatalw("Failed to register user cluster controller", zap.Error(err))
	}

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

	if runOp.openshift {
		if err := nodecsrapprover.Add(mgr, 4, cfg, log); err != nil {
			log.Fatalw("Failed to add nodecsrapprover controller", zap.Error(err))
		}
		if err := openshiftmasternodelabeler.Add(context.Background(), kubermaticlog.Logger, mgr); err != nil {
			log.Fatalw("Failed to add openshiftmasternodelabeler controller", zap.Error(err))
		}
		log.Info("Registered nodecsrapprover controller")
	}

	if err := containerlinux.Add(mgr, runOp.overwriteRegistry); err != nil {
		log.Fatalw("Failed to register the ContainerLinux controller", zap.Error(err))
	}

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
