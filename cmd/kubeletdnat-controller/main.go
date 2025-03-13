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
	"net"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubeletdnatcontroller "k8c.io/kubermatic/v2/pkg/controller/kubeletdnat-controller"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	pprofOpts := &flagopts.PProf{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	klog.InitFlags(nil)
	kubeconfigFlag := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	master := flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig")
	networkFlag := flag.String("node-access-network", "", "The network in CIDR notation to translate to.")
	chainNameFlag := flag.String("chain-name", "node-access-dnat", "Name of the chain in nat table.")
	vpnInterfaceFlag := flag.String("vpn-interface", "tun0", "Name of the vpn interface.")
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	reconciling.Configure(log)

	cli.Hello(log, "Kubelet DNAT-Controller", nil)

	_, network, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		log.Fatalw("node-access-network invalid or missing", zap.Error(err))
	}
	nodeAccessNetwork := network.IP

	config, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfigFlag)
	if err != nil {
		log.Fatalw("Failed to build configs from flags", zap.Error(err))
	}

	ctx := signals.SetupSignalHandler()

	// Wait until the API server is actually up & the corev1 api groups is available.
	// This is a smallish hack to avoid dying instantly when running as sidecar to the kube API server
	// The API server takes a few seconds to start which makes this sidecar die immediately
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
		if err != nil {
			return false, nil
		}

		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList); err != nil {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		log.Fatalw("Failed waiting for the API server to be alive", zap.Error(err))
	}

	// 8080 is already in use by the insecure port of the apiserver
	mgr, err := manager.New(config, manager.Options{
		BaseContext: func() context.Context {
			return ctx
		},
		Metrics: metricsserver.Options{
			BindAddress: ":8090",
		},
		PprofBindAddress: pprofOpts.ListenAddress,
	})
	if err != nil {
		log.Fatalw("Failed to create manager", zap.Error(err))
	}

	if err := kubeletdnatcontroller.Add(mgr, *chainNameFlag, nodeAccessNetwork, log, *vpnInterfaceFlag); err != nil {
		log.Fatalw("Failed to add the kubelet dnat controller", zap.Error(err))
	}

	if err := mgr.Start(ctx); err != nil {
		log.Fatalw("Failed to start kubeletdnat controller", zap.Error(err))
	}
}
