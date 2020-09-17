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
	"io/ioutil"
	"net"
	"os"

	"github.com/go-logr/zapr"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	cmdutil "k8c.io/kubermatic/v2/cmd/util"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/collectors"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/metrics"
	metricserver "k8c.io/kubermatic/v2/pkg/metrics/server"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"
	seedvalidation "k8c.io/kubermatic/v2/pkg/validation/seed"
	ctrl "sigs.k8s.io/controller-runtime"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	controllerName = "seed-controller-manager"
)

func main() {
	klog.InitFlags(nil)
	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	options, err := newControllerRunOptions()
	if err != nil {
		fmt.Printf("Failed to create controller run options due to = %v\n", err)
		os.Exit(1)
	}

	if err := options.validate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar().With(
		"worker-name", options.workerName,
	)
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	cmdutil.Hello(log, "Seed Controller-Manager", logOpts.Debug)

	// Set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	electionName := controllerName + "-leader-election"
	if options.workerName != "" {
		electionName += "-" + options.workerName
	}

	cfg := ctrl.GetConfigOrDie()
	// Create a manager, disable metrics as we have our own handler that exposes
	// the metrics of both the ctrltuntime registry and the default registry
	mgr, err := manager.New(cfg, manager.Options{
		MetricsBindAddress:      "0",
		LeaderElection:          options.enableLeaderElection,
		LeaderElectionNamespace: options.leaderElectionNamespace,
		LeaderElectionID:        electionName,
	})
	if err != nil {
		log.Fatalw("Failed to create the manager", zap.Error(err))
	}
	if err := mgr.Add(pprofOpts); err != nil {
		log.Fatalw("Failed to add the pprof handler", zap.Error(err))
	}
	// Add all custom type schemes to our scheme. Otherwise we won't get a informer
	if err := autoscalingv1beta2.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", autoscalingv1beta2.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}

	// Check if the CRD for the VerticalPodAutoscaler is registered by allocating an informer
	if err := mgr.GetAPIReader().List(context.Background(), &autoscalingv1beta2.VerticalPodAutoscalerList{}); err != nil {
		if _, crdNotRegistered := err.(*meta.NoKindMatchError); crdNotRegistered {
			log.Fatal(`
The VerticalPodAutoscaler is not installed in this seed cluster.
Please install the VerticalPodAutoscaler according to the documentation: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#installation`)
		}
	}

	//Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_controller_manager", prometheus.DefaultRegisterer)

	// Default to empty JSON object
	// TODO(irozzo) Do not create secret and image pull secret if empty
	dockerPullConfigJSON := []byte("{}")
	if options.dockerPullConfigJSONFile != "" {
		dockerPullConfigJSON, err = ioutil.ReadFile(options.dockerPullConfigJSONFile)
		if err != nil {
			log.Fatalw(
				"Failed to read docker pull config file",
				zap.String("file", options.dockerPullConfigJSONFile),
				zap.Error(err),
			)
		}
	}

	rootCtx := context.Background()
	seedGetter, err := seedGetterFactory(rootCtx, mgr.GetClient(), options)
	if err != nil {
		log.Fatalw("Unable to create the seed factory", zap.Error(err))
	}

	var clientProvider *client.Provider
	if !isInternalConfig(cfg) {
		clientProvider, err = client.NewExternal(mgr.GetClient())
	} else {
		clientProvider, err = client.NewInternal(mgr.GetClient())
	}
	if err != nil {
		log.Fatalw("Failed to get clientProvider", zap.Error(err))
	}

	if options.seedValidationHook.CertFile != "" && options.seedValidationHook.KeyFile != "" {
		restMapperCache := restmapper.New()
		// Creates a new default validator
		validator, err := seedvalidation.NewDefaultSeedValidator(
			options.workerName,
			// We only have a SeedGetter and not a SeedsGetter, so construct a little
			// wrapper
			func() (map[string]*kubermaticv1.Seed, error) {
				seeds := make(map[string]*kubermaticv1.Seed)

				seed, err := seedGetter()
				if err != nil {
					// ignore 404 errors so that on new seed clusters the initial
					// seed CR creation/validation can succeed
					if kerrors.IsNotFound(err) {
						return seeds, nil
					}

					return nil, err
				}

				seeds[seed.Name] = seed
				return seeds, nil
			},
			// This controler doesn't necessarily have an explicit kubeconfig, most of the time it
			// runs with in-cluster config. Just return the config from the manager and only allow
			// our own seed
			func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
				if seed.Name != options.dc {
					return nil, fmt.Errorf("can only return kubeconfig for our own seed (%q), got request for %q", options.dc, seed.Name)
				}
				return restMapperCache.Client(mgr.GetConfig())
			},
		)
		if err != nil {
			log.Fatalw("failed to create seed validator webhook server: %v", zap.Error(err))
		}
		seedValidationWebhookServer, err := options.seedValidationHook.Server(
			rootCtx,
			log,
			options.namespace,
			validator.Validate,
		)
		if err != nil {
			log.Fatalw("Failed to get seedValidationWebhookServer", zap.Error(err))
		}
		if err := mgr.Add(seedValidationWebhookServer); err != nil {
			log.Fatalw("Failed to add seedValidationWebhookServer to mgr", zap.Error(err))
		}
	}

	ctrlCtx := &controllerContext{
		ctx:                  rootCtx,
		runOptions:           options,
		mgr:                  mgr,
		clientProvider:       clientProvider,
		seedGetter:           seedGetter,
		dockerPullConfigJSON: dockerPullConfigJSON,
		log:                  log,
	}

	if err := createAllControllers(ctrlCtx); err != nil {
		log.Fatalw("Could not create all controllers", zap.Error(err))
	}

	// Use the API reader as the cache-backed reader will only contain data when we are leader
	// and return errors otherwise.
	// Ideally, the cache wouldn't require the leader lease:
	// https://github.com/kubernetes-sigs/controller-runtime/issues/677
	log.Debug("Starting clusters collector")
	collectors.MustRegisterClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())
	log.Debug("Starting addons collector")
	collectors.MustRegisterAddonCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetAPIReader())

	if err := mgr.Add(metricserver.New(options.internalAddr)); err != nil {
		log.Fatalw("failed to add metrics server", zap.Error(err))
	}

	log.Info("starting the seed-controller-manager...")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalw("problem running manager", zap.Error(err))
	}
}

// isInternalConfig returns `true` if the Host contained in the given config
// matches the one used when the controller runs in cluster, `false` otherwise.
func isInternalConfig(cfg *rest.Config) bool {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	return cfg.Host == "https://"+net.JoinHostPort(host, port)
}
