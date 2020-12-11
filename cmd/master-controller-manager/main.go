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

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/features"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/metrics"
	metricserver "k8c.io/kubermatic/v2/pkg/metrics/server"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/validation"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	ctrlruntimezaplog "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	controllerName = "kubermatic-master-controller-manager"
)

type controllerRunOptions struct {
	internalAddr            string
	seedvalidationHook      validation.WebhookOpts
	enableLeaderElection    bool
	leaderElectionNamespace string
	featureGates            features.FeatureGate

	workerName string
	namespace  string
}

type controllerContext struct {
	ctx                     context.Context
	mgr                     manager.Manager
	log                     *zap.SugaredLogger
	workerCount             int
	workerName              string
	workerNameLabelSelector labels.Selector
	workerNamePredicate     predicate.Predicate
	seedsGetter             provider.SeedsGetter
	seedKubeconfigGetter    provider.SeedKubeconfigGetter
	labelSelectorFunc       func(*metav1.ListOptions)
	namespace               string
}

func main() {
	ctrlCtx := &controllerContext{}
	runOpts := controllerRunOptions{featureGates: features.FeatureGate{}}
	klog.InitFlags(nil)
	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	runOpts.seedvalidationHook.AddFlags(flag.CommandLine, true)
	flag.StringVar(&runOpts.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&runOpts.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served.")
	flag.StringVar(&runOpts.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources.")
	flag.BoolVar(&runOpts.enableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&runOpts.leaderElectionNamespace, "leader-election-namespace", "", "Leader election namespace. In-cluster discovery will be attempted in such case.")
	flag.Var(&runOpts.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	addFlags(flag.CommandLine)
	flag.Parse()

	ctrlruntimelog.SetLogger(ctrlruntimezaplog.Logger(false))
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = log
	ctrlCtx.log = log
	ctrlCtx.workerName = runOpts.workerName
	ctrlCtx.namespace = runOpts.namespace

	cli.Hello(log, "Master Controller-Manager", logOpts.Debug, nil)

	// TODO remove label selector when everything is migrated to controller-runtime
	selector, err := workerlabel.LabelSelector(runOpts.workerName)
	if err != nil {
		log.Fatalw("failed to create the label selector for the given worker", "workerName", runOpts.workerName, zap.Error(err))
	}
	ctrlCtx.workerNameLabelSelector = selector

	ctrlCtx.workerNamePredicate = workerlabel.Predicates(runOpts.workerName)

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_master_controller_manager", prometheus.DefaultRegisterer)

	// prepare a context to use throughout the controller manager
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	ctrlCtx.ctx = ctx

	ctrlCtx.labelSelectorFunc = func(listOpts *metav1.ListOptions) {
		listOpts.LabelSelector = selector.String()
	}

	electionName := controllerName + "-leader-election"
	if runOpts.workerName != "" {
		electionName += "-" + runOpts.workerName
	}
	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		LeaderElection:          runOpts.enableLeaderElection,
		LeaderElectionNamespace: runOpts.leaderElectionNamespace,
		LeaderElectionID:        electionName,
		MetricsBindAddress:      "0",
	})
	if err != nil {
		log.Fatalw("failed to create Controller Manager instance", zap.Error(err))
	}
	ctrlCtx.mgr = mgr

	if err := mgr.Add(pprofOpts); err != nil {
		log.Fatalw("failed to add pprof endpoint", zap.Error(err))
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

	if runOpts.seedvalidationHook.CertFile != "" || runOpts.seedvalidationHook.KeyFile != "" {
		if err := runOpts.seedvalidationHook.Configure(mgr.GetWebhookServer()); err != nil {
			log.Fatalw("failed to configure admission webhook server", zap.Error(err))
		}

		// Register Seed validation handler
		h, err := seedValidationHandler(ctx, mgr.GetClient(), runOpts)
		if err != nil {
			log.Fatalw("failed to build Seed validation handler", zap.Error(err))
		}
		h.SetupWebhookWithManager(mgr)

	} else {
		log.Info("the validatingAdmissionWebhook server can not be started because seed-admissionwebhook-cert-file and seed-admissionwebhook-key-file are empty")
	}

	if err := createAllControllers(ctrlCtx); err != nil {
		log.Fatalw("could not create all controllers", zap.Error(err))
	}

	if err := mgr.Add(metricserver.New(runOpts.internalAddr)); err != nil {
		log.Fatalw("failed to add metrics server", zap.Error(err))
	}

	log.Info("starting the master-controller-manager...")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalw("problem running manager", zap.Error(err))
	}

}
