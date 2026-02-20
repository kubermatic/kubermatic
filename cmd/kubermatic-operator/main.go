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
	"flag"
	"strings"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	catalogv1alpha1 "k8c.io/application-catalog-manager/pkg/apis/applicationcatalog/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	masterctrl "k8c.io/kubermatic/v2/pkg/controller/operator/master"
	seedctrl "k8c.io/kubermatic/v2/pkg/controller/operator/seed"
	seedinit "k8c.io/kubermatic/v2/pkg/controller/operator/seed-init"
	seedcontrollerlifecycle "k8c.io/kubermatic/v2/pkg/controller/shared/seed-controller-lifecycle"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapischeme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"
)

type controllerRunOptions struct {
	namespace                string
	internalAddr             string
	workerCount              int
	workerName               string
	enableLeaderElection     bool
	enableGatewayAPI         bool
	httprouteWatchNamespaces string
}

func main() {
	ctx := signals.SetupSignalHandler()

	klog.InitFlags(nil)

	pprofOpts := &flagopts.PProf{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	opt := &controllerRunOptions{}
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace the operator runs in, uses to determine where to look for KubermaticConfigurations.")
	flag.IntVar(&opt.workerCount, "worker-count", 4, "Number of workers which process reconcilings in parallel.")
	flag.StringVar(&opt.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.StringVar(&opt.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.BoolVar(&opt.enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(
		&opt.enableGatewayAPI,
		"enable-gateway-api",
		false,
		"Allow watching Gateway API resources (Gateway and HTTPRoute). Requires Gateway API CRDs to exist",
	)
	flag.StringVar(
		&opt.httprouteWatchNamespaces,
		"httproute-watch-namespaces",
		"monitoring,mla",
		"Comma-separated list of namespaces to watch HTTPRoutes for Gateway listener sync. Only used when --enable-gateway-api is set.",
	)
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format).Named(opt.workerName)
	log := rawLog.Sugar()

	// update global logger instance
	kubermaticlog.Logger = log
	reconciling.Configure(log)

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	if len(opt.namespace) == 0 {
		log.Fatal("-namespace is a mandatory flag")
	}

	httprouteWatchNamespaces := sets.New[string]()
	if opt.enableGatewayAPI {
		for ns := range strings.SplitSeq(opt.httprouteWatchNamespaces, ",") {
			ns = strings.TrimSpace(ns)
			if ns != "" {
				httprouteWatchNamespaces.Insert(ns)
			}
		}

		if httprouteWatchNamespaces.Len() == 0 {
			log.Fatal("-httproute-watch-namespaces must contain at least one namespace")
		}
	}

	versions := kubermatic.GetVersions()
	helloLog := log.With("kubermatic-tag", versions.KubermaticContainerTag, "ui-tag", versions.UIContainerTag, "edition")

	cli.Hello(helloLog, "Kubermatic Operator", &versions)

	mgr, err := manager.New(ctrlruntime.GetConfigOrDie(), manager.Options{
		BaseContext: func() context.Context {
			return ctx
		},
		Metrics:          metricsserver.Options{BindAddress: opt.internalAddr},
		LeaderElection:   opt.enableLeaderElection,
		LeaderElectionID: "operator.kubermatic.k8c.io",
		PprofBindAddress: pprofOpts.ListenAddress,
	})
	if err != nil {
		log.Fatalw("Failed to create Controller Manager instance", zap.Error(err))
	}

	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}

	if err := catalogv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", catalogv1alpha1.SchemeGroupVersion), zap.Error(err))
	}

	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionsv1.SchemeGroupVersion), zap.Error(err))
	}

	if err := ciliumv2.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", ciliumv2.SchemeGroupVersion), zap.Error(err))
	}

	if opt.enableGatewayAPI {
		if err := gwapischeme.AddToScheme(mgr.GetScheme()); err != nil {
			log.Fatalw("Failed to register scheme", zap.Stringer("api", gatewayv1.SchemeGroupVersion), zap.Error(err))
		}
	}

	configGetter, err := kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(mgr.GetClient(), opt.namespace)
	if err != nil {
		log.Fatalw("Failed to construct configGetter", zap.Error(err))
	}

	seedsGetter, err := seedsGetterFactory(ctx, mgr.GetClient(), opt)
	if err != nil {
		log.Fatalw("Failed to construct seedsGetter", zap.Error(err))
	}

	seedKubeconfigGetter, err := kubernetesprovider.SeedKubeconfigGetterFactory(ctx, mgr.GetClient())
	if err != nil {
		log.Fatalw("Failed to construct seedKubeconfigGetter", zap.Error(err))
	}

	seedClientGetter := kubernetesprovider.SeedClientGetterFactory(seedKubeconfigGetter)

	err = masterctrl.Add(
		mgr,
		log,
		opt.namespace,
		opt.workerCount,
		opt.workerName,
		opt.enableGatewayAPI,
		sets.List(httprouteWatchNamespaces),
	)
	if err != nil {
		log.Fatalw("Failed to add operator-master controller", zap.Error(err))
	}

	if err := seedinit.Add(ctx, log, opt.namespace, mgr, seedClientGetter, opt.workerCount, opt.workerName); err != nil {
		log.Fatalw("Failed to add seed-init controller", zap.Error(err))
	}

	seedOperatorControllerFactory := func(ctx context.Context, mgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return seedctrl.ControllerName, seedctrl.Add(
			log,
			opt.namespace,
			mgr,
			seedManagerMap,
			configGetter,
			seedsGetter,
			opt.workerCount,
			opt.workerName,
		)
	}

	if err := seedcontrollerlifecycle.Add(ctx, log, mgr, opt.namespace, seedsGetter, seedKubeconfigGetter, seedOperatorControllerFactory); err != nil {
		log.Fatalw("Failed to create seed-lifecycle controller", zap.Error(err))
	}

	if err := mgr.Start(ctx); err != nil {
		log.Fatalw("Cannot start manager", zap.Error(err))
	}
}
