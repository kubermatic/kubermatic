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

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	masterctrl "k8c.io/kubermatic/v2/pkg/controller/operator/master"
	seedctrl "k8c.io/kubermatic/v2/pkg/controller/operator/seed"
	seedcontrollerlifecycle "k8c.io/kubermatic/v2/pkg/controller/shared/seed-controller-lifecycle"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/klog"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type controllerRunOptions struct {
	namespace            string
	internalAddr         string
	workerCount          int
	workerName           string
	enableLeaderElection bool
}

//nolint:gocritic,exitAfterDefer
func main() {
	ctx := context.Background()

	klog.InitFlags(nil)

	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	opt := &controllerRunOptions{}
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace the operator runs in, uses to determine where to look for KubermaticConfigurations.")
	flag.IntVar(&opt.workerCount, "worker-count", 4, "Number of workers which process reconcilings in parallel.")
	flag.StringVar(&opt.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.StringVar(&opt.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.BoolVar(&opt.enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format).Named(opt.workerName)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	// update global logger instance
	kubermaticlog.Logger = log

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	if len(opt.namespace) == 0 {
		log.Fatal("-namespace is a mandatory flag")
	}

	v := kubermatic.NewDefaultVersions()
	log.With("kubermatic", v.Kubermatic, "ui", v.UI).Infof("Moin, moin, I'm the Kubermatic %s Operator and these are the versions I work with.", v.KubermaticEdition)

	mgr, err := manager.New(ctrlruntime.GetConfigOrDie(), manager.Options{
		MetricsBindAddress: opt.internalAddr,
		LeaderElection:     opt.enableLeaderElection,
		LeaderElectionID:   "operator.kubermatic.io",
	})
	if err != nil {
		log.Fatalw("Failed to create Controller Manager instance", zap.Error(err))
	}

	if err := mgr.Add(pprofOpts); err != nil {
		log.Fatalw("Failed to add pprof endpoint", zap.Error(err))
	}

	if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", operatorv1alpha1.SchemeGroupVersion), zap.Error(err))
	}

	seedsGetter, err := seedsGetterFactory(ctx, mgr.GetClient(), opt)
	if err != nil {
		log.Fatalw("Failed to construct seedsGetter", zap.Error(err))
	}

	seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(ctx, mgr.GetClient())
	if err != nil {
		log.Fatalw("Failed to construct seedKubeconfigGetter", zap.Error(err))
	}

	if err := masterctrl.Add(ctx, mgr, log, opt.namespace, opt.workerCount, opt.workerName); err != nil {
		log.Fatalw("Failed to add operator-master controller", zap.Error(err))
	}

	seedOperatorControllerFactory := func(ctx context.Context, mgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return seedctrl.ControllerName, seedctrl.Add(
			ctx,
			log,
			opt.namespace,
			mgr,
			seedManagerMap,
			seedsGetter,
			opt.workerCount,
			opt.workerName,
		)
	}

	if err := seedcontrollerlifecycle.Add(ctx, log, mgr, opt.namespace, seedsGetter, seedKubeconfigGetter, seedOperatorControllerFactory); err != nil {
		log.Fatalw("Failed to create seed-lifecycle controller", zap.Error(err))
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalw("Cannot start manager", zap.Error(err))
	}
}
