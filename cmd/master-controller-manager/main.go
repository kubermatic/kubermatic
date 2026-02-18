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
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/zapr"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/collectors"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/metrics"
	metricserver "k8c.io/kubermatic/v2/pkg/metrics/server"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	controllerName = "kkp-master-controller-manager"
)

type controllerRunOptions struct {
	internalAddr             string
	enableLeaderElection     bool
	leaderElectionNamespace  string
	featureGates             features.FeatureGate
	configFile               string
	httprouteWatchNamespaces string

	workerName string
	namespace  string
}

type controllerContext struct {
	ctx                      context.Context
	mgr                      manager.Manager
	log                      *zap.SugaredLogger
	workerCount              int
	workerName               string
	workerNameLabelSelector  labels.Selector
	workerNamePredicate      predicate.Predicate
	seedsGetter              provider.SeedsGetter
	seedKubeconfigGetter     provider.SeedKubeconfigGetter
	labelSelectorFunc        func(*metav1.ListOptions)
	namespace                string
	versions                 kubermatic.Versions
	overwriteRegistry        string
	featureGates             features.FeatureGate
	httprouteWatchNamespaces []string

	configGetter provider.KubermaticConfigurationGetter
}

func main() {
	ctrlCtx := &controllerContext{
		versions: kubermatic.GetVersions(),
	}
	runOpts := controllerRunOptions{featureGates: features.FeatureGate{}}
	klog.InitFlags(nil)
	pprofOpts := &flagopts.PProf{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	flag.StringVar(&runOpts.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&runOpts.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served.")
	flag.StringVar(&runOpts.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources.")
	flag.BoolVar(&runOpts.enableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&ctrlCtx.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&runOpts.leaderElectionNamespace, "leader-election-namespace", "", "Leader election namespace. In-cluster discovery will be attempted in such case.")
	flag.Var(&runOpts.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&runOpts.configFile, "kubermatic-configuration-file", "", "(for development only) path to a KubermaticConfiguration YAML file")
	flag.StringVar(&runOpts.httprouteWatchNamespaces, "httproute-watch-namespaces", "mla,monitoring", "Comma-separated list of namespaces to watch HTTPRoutes for Gateway listener sync")
	addFlags(flag.CommandLine)
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	kubermaticlog.Logger = log
	reconciling.Configure(log)
	ctrlCtx.log = log
	ctrlCtx.workerName = runOpts.workerName
	ctrlCtx.namespace = runOpts.namespace
	ctrlCtx.featureGates = runOpts.featureGates
	ctrlCtx.httprouteWatchNamespaces = strings.Split(runOpts.httprouteWatchNamespaces, ",")

	// Set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	cli.Hello(log, "Master Controller-Manager", &ctrlCtx.versions)

	// TODO remove label selector when everything is migrated to controller-runtime
	selector, err := workerlabel.LabelSelector(runOpts.workerName)
	if err != nil {
		log.Fatalw("failed to create the label selector for the given worker", "workerName", runOpts.workerName, zap.Error(err))
	}
	ctrlCtx.workerNameLabelSelector = selector

	ctrlCtx.workerNamePredicate = workerlabel.Predicate(runOpts.workerName)

	// for development purposes, a local configuration file
	// can be used to provide the KubermaticConfiguration
	var config *kubermaticv1.KubermaticConfiguration
	if runOpts.configFile != "" {
		if config, err = loadKubermaticConfiguration(runOpts.configFile); err != nil {
			log.Fatalw("invalid KubermaticConfiguration", zap.Error(err))
		}
	}

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_master_controller_manager", prometheus.DefaultRegisterer)

	// prepare a context to use throughout the controller manager
	ctx := signals.SetupSignalHandler()
	ctrlCtx.ctx = ctx

	ctrlCtx.labelSelectorFunc = func(listOpts *metav1.ListOptions) {
		listOpts.LabelSelector = selector.String()
	}

	electionName := controllerName + "-leader-election"
	if runOpts.workerName != "" {
		electionName += "-" + runOpts.workerName
	}
	mgr, err := manager.New(ctrlruntime.GetConfigOrDie(), manager.Options{
		BaseContext: func() context.Context {
			return ctx
		},
		LeaderElection:          runOpts.enableLeaderElection,
		LeaderElectionNamespace: runOpts.leaderElectionNamespace,
		LeaderElectionID:        electionName,
		Metrics:                 metricsserver.Options{BindAddress: "0"},
		PprofBindAddress:        pprofOpts.ListenAddress,
	})
	if err != nil {
		log.Fatalw("failed to create Controller Manager instance", zap.Error(err))
	}
	ctrlCtx.mgr = mgr

	if config != nil {
		ctrlCtx.configGetter, err = kubernetesprovider.StaticKubermaticConfigurationGetterFactory(config)
	} else {
		ctrlCtx.configGetter, err = kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(mgr.GetClient(), runOpts.namespace)
	}
	if err != nil {
		log.Fatalw("Unable to create the configuration getter", zap.Error(err))
	}

	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}

	// these two getters rely on the ctrlruntime manager being started; they
	// are only used inside controllers
	ctrlCtx.seedsGetter, err = seedsGetterFactory(ctx, mgr.GetClient(), ctrlCtx.namespace)
	if err != nil {
		log.Fatalw("failed to construct seedsGetter", zap.Error(err))
	}
	ctrlCtx.seedKubeconfigGetter, err = seedKubeconfigGetterFactory(ctx, mgr.GetClient(), runOpts)
	if err != nil {
		log.Fatalw("failed to construct seedKubeconfigGetter", zap.Error(err))
	}

	log.Debug("Starting external clusters collector")
	collectors.MustRegisterExternalClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())

	log.Debug("Starting projects collector")
	collectors.MustRegisterProjectCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())

	log.Debug("Starting seeds collector")
	collectors.MustRegisterSeedCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())

	if err := createAllControllers(ctrlCtx); err != nil {
		log.Fatalw("could not create all controllers", zap.Error(err))
	}

	if err := mgr.Add(metricserver.New(runOpts.internalAddr)); err != nil {
		log.Fatalw("failed to add metrics server", zap.Error(err))
	}

	log.Info("starting the master-controller-manager...")
	if err := mgr.Start(ctx); err != nil {
		log.Fatalw("problem running manager", zap.Error(err))
	}
}

func loadKubermaticConfiguration(filename string) (*kubermaticv1.KubermaticConfiguration, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer f.Close()

	config := &kubermaticv1.KubermaticConfiguration{}
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse file as YAML: %w", err)
	}

	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to process: %w", err)
	}

	return defaulted, nil
}
