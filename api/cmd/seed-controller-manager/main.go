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
	"os"

	"github.com/go-logr/zapr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	cmdutil "github.com/kubermatic/kubermatic/api/cmd/util"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/collectors"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	seedmigrations "github.com/kubermatic/kubermatic/api/pkg/crd/migrations/seed"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	metricserver "github.com/kubermatic/kubermatic/api/pkg/metrics/server"
	"github.com/kubermatic/kubermatic/api/pkg/pprof"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/clientcmd"
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

	config, err := clientcmd.BuildConfigFromFlags(options.masterURL, options.kubeconfig)
	if err != nil {
		log.Fatalw("Failed to create a kubernetes config", zap.Error(err))
	}

	// Set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	// Create a manager, disable metrics as we have our own handler that exposes
	// the metrics of both the ctrltuntime registry and the default registry
	mgr, err := manager.New(config, manager.Options{MetricsBindAddress: "0"})
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

	dockerPullConfigJSON, err := ioutil.ReadFile(options.dockerPullConfigJSONFile)
	if err != nil {
		log.Fatalw(
			"Failed to read docker pull config file",
			zap.String("file", options.dockerPullConfigJSONFile),
			zap.Error(err),
		)
	}

	rootCtx, rootCancel := context.WithCancel(context.Background())
	seedGetter, err := seedGetterFactory(rootCtx, mgr.GetClient(), options)
	if err != nil {
		log.Fatalw("Unable to create the seed factory", zap.Error(err))
	}

	var clientProvider *client.Provider
	if options.kubeconfig != "" {
		clientProvider, err = client.NewExternal(mgr.GetClient())
	} else {
		clientProvider, err = client.NewInternal(mgr.GetClient())
	}
	if err != nil {
		log.Fatalw("Failed to get clientProvider", zap.Error(err))
	}

	if options.seedValidationHook.CertFile != "" && options.seedValidationHook.KeyFile != "" {
		restMapperCache := restmapper.New()
		seedValidationWebhookServer, err := options.seedValidationHook.Server(
			rootCtx,
			log,
			options.namespace,
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
			false)
		if err != nil {
			log.Fatalw("Failed to get seedValidationWebhookServer", zap.Error(err))
		}
		if err := mgr.Add(seedValidationWebhookServer); err != nil {
			log.Fatalw("Failed to add seedValidationWebhookServer to mgr", zap.Error(err))
		}
	}

	ctrlCtx := &controllerContext{
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

	var g run.Group
	// This group is forever waiting in a goroutine for signals to stop
	{
		signalChan := signals.SetupSignalHandler()
		g.Add(func() error {
			select {
			case <-signalChan:
				log.Info("Received a signal to stop")
				return nil
			case <-rootCtx.Done():
				return nil
			}
		}, func(err error) {
			rootCancel()
		})
	}

	// This group is running the metrics server, which needs to run all the time
	// because Prometheus scrapes all controller-manager pods, not just the leader.
	{
		g.Add(func() error {
			server := metricserver.New(options.internalAddr)
			if err := server.Start(rootCtx.Done()); err != nil {
				return fmt.Errorf("failed to start the metrics server: %v", err)
			}
			return nil
		}, func(err error) {
			rootCancel()
		})
	}

	// This group is running the actual controller logic
	{
		leaderCtx, stopLeaderElection := context.WithCancel(rootCtx)
		defer stopLeaderElection()

		g.Add(func() error {
			electionName := controllerName
			if options.workerName != "" {
				electionName += "-" + options.workerName
			}

			return leaderelection.RunAsLeader(leaderCtx, log, config, mgr.GetEventRecorderFor(controllerName), electionName, func(ctx context.Context) error {
				log.Info("Executing migrations...")
				if err := seedmigrations.RunAll(leaderCtx, ctrlCtx.mgr.GetConfig(), options.workerName); err != nil {
					return fmt.Errorf("failed to run migrations: %v", err)
				}
				log.Info("Migrations executed successfully")

				log.Info("Starting the controller-manager...")
				if err := mgr.Start(ctx.Done()); err != nil {
					return fmt.Errorf("the controller-manager stopped with an error: %v", err)
				}

				return nil
			})
		}, func(err error) {
			stopLeaderElection()
		})
	}

	if err := g.Run(); err != nil {
		// Set the error as field so we have a consistent way of logging errors
		log.Fatalw("Shutting down with error", zap.Error(err))
	}
}
