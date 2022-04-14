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
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"
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
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	// set the logger used by controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog))

	// say hello
	versions := kubermatic.NewDefaultVersions()
	cli.Hello(log, "User Cluster Webhook", options.log.Debug, &versions)

	// /////////////////////////////////////////
	// get kubeconfig

	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get kubeconfig", zap.Error(err))
	}

	// /////////////////////////////////////////
	// create manager

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatalw("Failed to create the manager", zap.Error(err))
	}

	// apply the CLI flags for configuring the  webhook server to the manager
	if err := options.webhook.Configure(mgr.GetWebhookServer()); err != nil {
		log.Fatalw("Failed to configure webhook server", zap.Error(err))
	}

	// add APIs we use
	addAPIs(mgr.GetScheme(), log)

	// /////////////////////////////////////////
	// add pprof runnable, which will start a websever if configured

	if err := mgr.Add(&options.pprof); err != nil {
		log.Fatalw("Failed to add the pprof handler", zap.Error(err))
	}

	log.Info("Starting the webhook...")
	if err := mgr.Start(ctrlruntime.SetupSignalHandler()); err != nil {
		log.Fatalw("The webhook has failed", zap.Error(err))
	}
}

func addAPIs(dst *runtime.Scheme, log *zap.SugaredLogger) {
	if err := kubermaticv1.AddToScheme(dst); err != nil {
		log.Fatalw("failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(dst); err != nil {
		log.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
}
