/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	applicationinstallationvalidation "k8c.io/kubermatic/v2/pkg/webhook/application/applicationinstallation/validation"
	machinevalidation "k8c.io/kubermatic/v2/pkg/webhook/machine/validation"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	// /////////////////////////////////////////
	// setup flags

	options, err := initApplicationOptions()
	if err != nil {
		fmt.Printf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	// /////////////////////////////////////////
	// setup logging

	rawLog := kubermaticlog.New(options.log.Debug, options.log.Format)
	log := rawLog.Sugar()

	// set the logger used by controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	// say hello
	versions := kubermatic.NewDefaultVersions()
	cli.Hello(log, "User Cluster Webhook", options.log.Debug, &versions)

	// /////////////////////////////////////////
	// get kubeconfigs

	seedCfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalw("Failed to get seed kubeconfig", zap.Error(err))
	}

	userCfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get user cluster kubeconfig")
	}

	// This case unusual in a sense that we need to configure and start managers for both user and seed clusters.
	ctx := ctrlruntime.SetupSignalHandler()

	seedMgr, err := manager.New(seedCfg, manager.Options{
		BaseContext: func() context.Context {
			return ctx
		},
	})
	if err != nil {
		log.Fatalw("Failed to create the seed cluster manager", zap.Error(err))
	}

	userMgr, err := manager.New(userCfg, manager.Options{
		BaseContext: func() context.Context {
			return ctx
		},
		MetricsBindAddress: "0",
	})
	if err != nil {
		log.Fatalw("Failed to create the user cluster manager", zap.Error(err))
	}

	// Apply the CLI flags for configuring the webhook servers.
	if err := options.seedWebhook.Configure(seedMgr.GetWebhookServer()); err != nil {
		log.Fatalw("Failed to configure webhook server", zap.Error(err))
	}
	if err := options.userWebhook.Configure(userMgr.GetWebhookServer()); err != nil {
		log.Fatalw("Failed to configure webhook server", zap.Error(err))
	}

	// add APIs we use
	addAPIs(seedMgr.GetScheme(), log)
	addAPIs(userMgr.GetScheme(), log)

	// /////////////////////////////////////////
	// add pprof runnable, which will start a websever if configured

	if err := seedMgr.Add(&options.pprof); err != nil {
		log.Fatalw("Failed to add the pprof handler (seed manager)", zap.Error(err))
	}
	if err := userMgr.Add(&options.pprof); err != nil {
		log.Fatalw("Failed to add the pprof handler (user-cluster manager)", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup webhooks

	// Setup the validation admission handler for ApplicationInstallation CRDs in seed manager.
	applicationinstallationvalidation.NewAdmissionHandler(seedMgr.GetClient()).SetupWebhookWithManager(seedMgr)

	// Setup Machine Webhook in user manager.
	machineValidator, err := machinevalidation.NewValidator(seedMgr.GetClient(), userMgr.GetClient(), log, options.caBundle, options.projectID)
	if err != nil {
		log.Fatalw("Failed to setup Machine validator", zap.Error(err))
	}
	if err := builder.WebhookManagedBy(userMgr).For(&clusterv1alpha1.Machine{}).WithValidator(machineValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup Machine validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// Start managers

	go func() {
		log.Info("Starting the user cluster manager in the background...")
		if err := userMgr.Start(ctx); err != nil {
			log.Fatalw("The user manager has failed", zap.Error(err))
		}
	}()

	log.Info("Starting the seed manager...")
	if err := seedMgr.Start(ctx); err != nil {
		log.Fatalw("The seed manager has failed", zap.Error(err))
	}
}

func addAPIs(dst *runtime.Scheme, log *zap.SugaredLogger) {
	if err := kubermaticv1.AddToScheme(dst); err != nil {
		log.Fatalw("failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(dst); err != nil {
		log.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := appskubermaticv1.AddToScheme(dst); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", appskubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
}
