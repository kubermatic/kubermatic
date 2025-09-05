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
	"net"
	"os"
	"slices"
	"strings"

	"github.com/go-logr/zapr"
	constrainttemplatesv1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	"github.com/prometheus/client_golang/prometheus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"

	kubelbv1alpha1 "k8c.io/kubelb/api/ee/kubelb.k8c.io/v1alpha1"
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/collectors"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/metrics"
	metricserver "k8c.io/kubermatic/v2/pkg/metrics/server"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	controllerName = "kkp-seed-controller-manager"
)

func main() {
	klog.InitFlags(nil)
	pprofOpts := &flagopts.PProf{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	options, err := newControllerRunOptions()
	if err != nil {
		fmt.Printf("Failed to create controller run options: %v\n", err)
		os.Exit(1)
	}

	if err := options.validate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	if options.workerName != "" {
		log = log.With("worker-name", options.workerName)
	}

	// Set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))
	reconciling.Configure(log)

	// make sure the logging flags actually affect the global (deprecated) logger instance
	kubermaticlog.Logger = log

	versions := kubermatic.GetVersions()
	cli.Hello(log, "Seed Controller-Manager", &versions)

	electionName := controllerName + "-leader-election"
	if options.workerName != "" {
		electionName += "-" + options.workerName
	}

	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get kubeconfig", zap.Error(err))
	}

	// Create a manager, disable metrics as we have our own handler that exposes
	// the metrics of both the ctrlruntime registry and the default registry
	rootCtx := signals.SetupSignalHandler()

	mgr, err := manager.New(cfg, manager.Options{
		BaseContext: func() context.Context {
			return rootCtx
		},
		Metrics:                 metricsserver.Options{BindAddress: "0"},
		LeaderElection:          options.enableLeaderElection,
		LeaderElectionNamespace: options.leaderElectionNamespace,
		LeaderElectionID:        electionName,
		// inject a custom broadcaster because during cluster deletion we emit more than
		// usual events and the default configuration would consider this spam.
		EventBroadcaster: record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{
			BurstSize: 20,
			QPS:       5,
		}),
		PprofBindAddress: pprofOpts.ListenAddress,
	})
	if err != nil {
		log.Fatalw("Failed to create the manager", zap.Error(err))
	}
	// Add all custom type schemes to our scheme. Otherwise we won't get a informer
	if err := autoscalingv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", autoscalingv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := constrainttemplatesv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", constrainttemplatesv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := osmv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", osmv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := appskubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", appskubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionsv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := velerov1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", velerov1.SchemeGroupVersion), zap.Error(err))
	}
	if err := kubelbv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", kubelbv1alpha1.SchemeGroupVersion), zap.Error(err))
	}

	// Check if the CRD for the VerticalPodAutoscaler is registered by allocating an informer
	if err := mgr.GetAPIReader().List(rootCtx, &autoscalingv1.VerticalPodAutoscalerList{}); err != nil {
		if meta.IsNoMatchError(err) {
			log.Fatal(`
The VerticalPodAutoscaler is not installed in this seed cluster.
Please install the VerticalPodAutoscaler according to the documentation: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#installation`)
		}
	}

	// Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_controller_manager", prometheus.DefaultRegisterer)

	// Default to empty JSON object
	// TODO: Do not create secret and image pull secret if empty
	dockerPullConfigJSON := []byte("{}")
	if options.dockerPullConfigJSONFile != "" {
		dockerPullConfigJSON, err = os.ReadFile(options.dockerPullConfigJSONFile)
		if err != nil {
			log.Fatalw(
				"Failed to read docker pull config file",
				zap.String("file", options.dockerPullConfigJSONFile),
				zap.Error(err),
			)
		}
	}

	seedGetter, err := seedGetterFactory(rootCtx, mgr.GetClient(), options)
	if err != nil {
		log.Fatalw("Unable to create the seed getter", zap.Error(err))
	}

	var configGetter provider.KubermaticConfigurationGetter
	if options.kubermaticConfiguration != nil {
		configGetter, err = kubernetesprovider.StaticKubermaticConfigurationGetterFactory(options.kubermaticConfiguration)
	} else {
		configGetter, err = kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(mgr.GetClient(), options.namespace)
	}
	if err != nil {
		log.Fatalw("Unable to create the configuration getter", zap.Error(err))
	}

	projectsGetter, err := kubernetesprovider.ProjectsGetterFactory(rootCtx, mgr.GetClient())
	if err != nil {
		log.Fatalw("Unable to create the seed getter", zap.Error(err))
	}

	var clientProvider *client.Provider
	if isInternalConfig(cfg) {
		clientProvider, err = client.NewInternal(mgr.GetClient())
	} else {
		clientProvider, err = client.NewExternal(mgr.GetClient())
	}
	if err != nil {
		log.Fatalw("Failed to get clientProvider", zap.Error(err))
	}

	ctrlCtx := &controllerContext{
		ctx:                  rootCtx,
		runOptions:           options,
		mgr:                  mgr,
		clientProvider:       clientProvider,
		seedGetter:           seedGetter,
		projectsGetter:       projectsGetter,
		configGetter:         configGetter,
		dockerPullConfigJSON: dockerPullConfigJSON,
		log:                  log,
		versions:             versions,
	}

	if err := createAllControllers(ctrlCtx); err != nil {
		log.Fatalw("Could not create all controllers", zap.Error(err))
	}

	disabledCollectors := strings.Split(options.disabledCollectors, ",")

	if !slices.Contains(disabledCollectors, string(kubermaticv1.ClusterBackupCollector)) {
		// Use the API reader as the cache-backed reader will only contain data when we are leader
		// and return errors otherwise.
		// Ideally, the cache wouldn't require the leader lease:
		// https://github.com/kubernetes-sigs/controller-runtime/issues/677
		log.Debug("Starting cluster backup collector")
		collectors.MustRegisterClusterBackupCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader(), log, options.caBundle, seedGetter)
	}
	if !slices.Contains(disabledCollectors, string(kubermaticv1.ClusterCollector)) {
		log.Debug("Starting clusters collector")
		collectors.MustRegisterClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())
	}
	if !slices.Contains(disabledCollectors, string(kubermaticv1.AddonCollector)) {
		log.Debug("Starting addons collector")
		collectors.MustRegisterAddonCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())
	}
	if !slices.Contains(disabledCollectors, string(kubermaticv1.ProjectCollector)) {
		// The canonical source of projects is the master cluster, but since they are replicated onto
		// seeds, we start the project collctor on seed clusters as well, just for convenience for the admin.
		log.Debug("Starting projects collector")
		collectors.MustRegisterProjectCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())
	}

	if err := mgr.Add(metricserver.New(options.internalAddr)); err != nil {
		log.Fatalw("failed to add metrics server", zap.Error(err))
	}

	log.Info("Starting the seed-controller-manager")
	if err := mgr.Start(rootCtx); err != nil {
		log.Fatalw("problem running manager", zap.Error(err))
	}
}

// isInternalConfig returns `true` if the Host contained in the given config
// matches the one used when the controller runs in cluster, `false` otherwise.
func isInternalConfig(cfg *rest.Config) bool {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	return cfg.Host == "https://"+net.JoinHostPort(host, port)
}
