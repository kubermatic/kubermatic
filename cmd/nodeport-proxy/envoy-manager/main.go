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
	"flag"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
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
	flag.IntVar(&ctrlOpts.EnvoySNIListenerPort, "envoy-sni-port", 0, "Port used for SNI entry point.")
	flag.IntVar(&ctrlOpts.EnvoyTunnelingListenerPort, "envoy-tunneling-port", 0, "Port used for HTTP/2 CONNECT termination.")
	flag.DurationVar(&ctrlOpts.SNIListenerIdleTimeout, "sni-listener-idle-timeout", 0, "Idle timeout for SNI listener downstream TCP connections. Set to 0 to keep Envoy default behavior.")
	flag.DurationVar(&ctrlOpts.TunnelingConnectionIdleTimeout, "tunneling-connection-idle-timeout", 0, "Idle timeout for tunneling listener downstream connections. Set to 0 to keep Envoy default behavior.")
	flag.DurationVar(&ctrlOpts.TunnelingConnectionStreamTimeout, "tunneling-stream-idle-timeout", 0, "Idle timeout for tunneling listener CONNECT streams. Set to 0 to keep Envoy default behavior.")
	flag.DurationVar(&ctrlOpts.DownstreamTCPKeepaliveTime, "downstream-tcp-keepalive-time", 0, "Idle time before sending TCP keepalive probes on downstream listener sockets. Set to 0 to leave unset; keepalive is configured only when at least one downstream keepalive option is set.")
	flag.DurationVar(&ctrlOpts.DownstreamTCPKeepaliveInterval, "downstream-tcp-keepalive-interval", 0, "Interval between TCP keepalive probes on downstream listener sockets. Set to 0 to leave unset; keepalive is configured only when at least one downstream keepalive option is set.")
	flag.IntVar(&ctrlOpts.DownstreamTCPKeepaliveProbes, "downstream-tcp-keepalive-probes", 0, "Maximum unanswered TCP keepalive probes on downstream listener sockets before considering a connection dead. Set to 0 to leave unset; keepalive is configured only when at least one downstream keepalive option is set.")
	flag.DurationVar(&ctrlOpts.UpstreamTCPKeepaliveTime, "upstream-tcp-keepalive-time", 0, "Idle time before sending TCP keepalive probes on upstream cluster sockets. Set to 0 to leave unset; keepalive is configured only when at least one upstream keepalive option is set.")
	flag.DurationVar(&ctrlOpts.UpstreamTCPKeepaliveProbeInterval, "upstream-tcp-keepalive-interval", 0, "Interval between TCP keepalive probes on upstream cluster sockets. Set to 0 to leave unset; keepalive is configured only when at least one upstream keepalive option is set.")
	flag.IntVar(&ctrlOpts.UpstreamTCPKeepaliveProbeAttempts, "upstream-tcp-keepalive-probes", 0, "Maximum unanswered TCP keepalive probes on upstream cluster sockets before considering a connection dead. Set to 0 to leave unset; keepalive is configured only when at least one upstream keepalive option is set.")
	flag.StringVar(&ctrlOpts.Namespace, "namespace", "", "The namespace we should use for pods and services. Leave empty for all namespaces.")
	flag.StringVar(&ctrlOpts.ExposeAnnotationKey, "expose-annotation-key", nodeportproxy.DefaultExposeAnnotationKey, "The annotation key used to determine if a service should be exposed")
	flag.Parse()

	// setup signal handler
	ctx := signals.SetupSignalHandler()

	// init logging
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	cli.Hello(log, "Envoy-Manager", nil)

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	cacheOpts := cache.Options{
		DefaultNamespaces: map[string]cache.Config{},
	}

	if ctrlOpts.Namespace != "" {
		cacheOpts.DefaultNamespaces[ctrlOpts.Namespace] = cache.Config{}
	}

	mgr, err := manager.New(config, manager.Options{
		Cache: cacheOpts,
	})
	if err != nil {
		log.Fatalw("failed to build controller-runtime manager", zap.Error(err))
	}

	r, snapshotCache, err := envoymanager.NewReconciler(ctx, log.With("component", "envoycache"), mgr.GetClient(), ctrlOpts)
	if err != nil {
		log.Fatalw("failed to build reconciler", zap.Error(err))
	}
	if err := r.SetupWithManager(ctx, mgr); err != nil {
		log.Fatalw("failed to register reconciler with controller-runtime manager", zap.Error(err))
	}

	srv.Cache = snapshotCache
	srv.Log = log.With("component", "envoyconfigserver")
	if err := mgr.Add(&srv); err != nil {
		log.Fatalw("failed to register envoy config server with controller-runtime manager", zap.Error(err))
	}

	if err := mgr.Start(ctx); err != nil {
		log.Errorw("manager ended with error", zap.Error(err))
	}
}
