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

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	srv := Server{}
	ctrlOpts := envoymanager.Options{}
	flag.StringVar(&srv.ListenAddress, "listen-address", ":8001", "Address to serve on")
	flag.StringVar(&ctrlOpts.EnvoyNodeName, "envoy-node-name", "kube", "Name of the envoy nodes to apply the config to via xds.")
	flag.IntVar(&ctrlOpts.EnvoyAdminPort, "envoy-admin-port", 9001, "Envoys admin port")
	flag.IntVar(&ctrlOpts.EnvoyStatsPort, "envoy-stats-port", 8002, "Limited port which should be opened on envoy to expose metrics and the health check. Endpoints are: /healthz & /stats")
	flag.StringVar(&ctrlOpts.Namespace, "namespace", "", "The namespace we should use for pods and services. Leave empty for all namespaces.")
	flag.StringVar(&ctrlOpts.ExposeAnnotationKey, "expose-annotation-key", envoymanager.DefaultExposeAnnotationKey, "The annotation key used to determine if a service should be exposed")
	flag.Parse()

	// setup signal handler
	ctx, cancel := context.WithCancel(context.Background())
	stopCh := signals.SetupSignalHandler()
	go func() {
		<-stopCh
		cancel()
	}()

	// init logging
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	cli.Hello(log, "Envoy-Manager", logOpts.Debug)

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	mgr, err := manager.New(config, manager.Options{Namespace: ctrlOpts.Namespace})
	if err != nil {
		log.Fatalw("failed to build controller-runtime manager", zap.Error(err))
	}

	r, snapshotCache, err := envoymanager.NewReconciler(ctx, log.With("component", "envoycache"), mgr.GetClient(), ctrlOpts)
	if err != nil {
		log.Fatalw("failed to build reconciler", zap.Error(err))
	}
	if err := r.SetupWithManager(mgr); err != nil {
		log.Fatalw("failed to register reconciler with controller-runtime manager", zap.Error(err))
	}

	srv.Cache = snapshotCache
	srv.Log = log.With("component", "envoyconfigserver")
	if err := mgr.Add(&srv); err != nil {
		log.Fatalw("failed to register envoy config server with controller-runtime manager", zap.Error(err))
	}

	if err := mgr.Start(stopCh); err != nil {
		log.Errorw("manager ended with error", zap.Error(err))
	}
}
