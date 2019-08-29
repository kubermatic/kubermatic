package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/heptiolabs/healthcheck"
	"github.com/oklog/run"

	"github.com/kubermatic/kubermatic/api/pkg/controller/container-linux"
	"github.com/kubermatic/kubermatic/api/pkg/controller/ipam"
	"github.com/kubermatic/kubermatic/api/pkg/controller/nodecsrapprover"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"
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
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
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
	flag.Parse()

	if runOp.namespace == "" {
		glog.Fatal("-namespace must be set")
	}
	if runOp.caPath == "" {
		glog.Fatal("-ca-cert must be set")
	}
	if runOp.clusterURL == "" {
		glog.Fatal("-cluster-url must be set")
	}
	clusterURL, err := url.Parse(runOp.clusterURL)
	if err != nil {
		glog.Fatal(err)
	}
	if runOp.openvpnServerPort == 0 {
		glog.Fatal("-openvpn-server-port must be set")
	}

	caBytes, err := ioutil.ReadFile(runOp.caPath)
	if err != nil {
		glog.Fatal(err)
	}
	certs, err := certutil.ParseCertsPEM(caBytes)
	if err != nil {
		glog.Fatal(err)
	}
	if len(certs) != 1 {
		glog.Fatalf("did not find exactly one but %d certificates in the given CA", len(certs))
	}

	openVPNCACertBytes, err := ioutil.ReadFile(runOp.openvpnCACertFilePath)
	if err != nil {
		glog.Fatalf("failed to read openvpn-ca-cert-file: %v", err)
	}
	openVPNCACerts, err := certutil.ParseCertsPEM(openVPNCACertBytes)
	if err != nil {
		glog.Fatalf("failed to parse openVPN CA file: %v", err)
	}
	if certsLen := len(openVPNCACerts); certsLen != 1 {
		glog.Fatalf("did not find exactly one but %v certificates in the openVPN CA file", certsLen)
	}
	openVPNCAKeyBytes, err := ioutil.ReadFile(runOp.openvpnCAKeyFilePath)
	if err != nil {
		glog.Fatalf("failed to read openvpn-ca-key-file: %v", err)
	}
	openVPNCAKey, err := certutil.ParsePrivateKeyPEM(openVPNCAKeyBytes)
	if err != nil {
		glog.Fatalf("failed to parse openVPN CA key file: %v", err)
	}
	openVPNECSDAKey, isECDSAKey := openVPNCAKey.(*ecdsa.PrivateKey)
	if !isECDSAKey {
		glog.Fatal("the openVPN private key is not an ECDSA key")
	}
	openVPNCACert := &resources.ECDSAKeyPair{Cert: openVPNCACerts[0], Key: openVPNECSDAKey}

	var g run.Group

	healthHandler := healthcheck.NewHandler()

	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()

	log.SetLogger(log.ZapLogger(false))

	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:          true,
		LeaderElectionNamespace: metav1.NamespaceSystem,
		LeaderElectionID:        "user-cluster-controller-leader-lock",
		MetricsBindAddress:      runOp.metricsListenAddr,
	})
	if err != nil {
		glog.Fatal(err)
	}

	glog.Info("registering components")
	if err := apiextensionv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}
	if err := apiregistrationv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	// Setup all Controllers
	glog.Info("registering controllers")
	if err := usercluster.Add(mgr,
		runOp.openshift,
		runOp.version,
		runOp.namespace,
		certs[0],
		clusterURL,
		runOp.openvpnServerPort,
		healthHandler.AddReadinessCheck,
		openVPNCACert); err != nil {
		glog.Fatalf("failed to register user cluster controller: %v", err)
	}

	if len(runOp.networks) > 0 {
		if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			glog.Fatalf("failed to add clusterv1alpha1 scheme: %v", err)
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
				glog.Fatalf("failed to initially create the Machine CR: %v", err)
			}
		}
		if err := ipam.Add(mgr, runOp.networks); err != nil {
			glog.Fatalf("failed to add IPAM controller to mgr: %v", err)
		}
		glog.Infof("Added IPAM controller to mgr")
	}

	if err := rbacusercluster.Add(mgr, healthHandler.AddReadinessCheck); err != nil {
		glog.Fatalf("failed to add user RBAC controller to mgr: %v", err)
	}

	if runOp.openshift {
		if err := nodecsrapprover.Add(mgr, 4, cfg); err != nil {
			glog.Fatalf("failed to add nodecsrapprover controller: %v", err)
		}
		if err := openshiftmasternodelabeler.Add(context.Background(), kubermaticlog.Logger, mgr); err != nil {
			glog.Fatalf("failed to add openshiftmasternodelabeler contorller: %v", err)
		}
		glog.Infof("Registered nodecsrapprover controller")
	}

	if err := containerlinux.Add(mgr, runOp.overwriteRegistry); err != nil {
		glog.Fatalf("failed to register the ContainerLinux controller: %v", err)
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
			glog.Infof("stopping user cluster controller manager, err = %v", err)
		})
	}

	// This group starts the readiness & liveness http server
	{
		h := &http.Server{Addr: runOp.healthListenAddr, Handler: healthHandler}
		g.Add(func() error {
			return h.ListenAndServe()
		}, func(err error) {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := h.Shutdown(shutdownCtx); err != nil {
				glog.Errorf("Healthcheck handler terminated with an error: %v", err)
			}
		})
	}

	if err := g.Run(); err != nil {
		glog.Fatal(err)
	}

}
