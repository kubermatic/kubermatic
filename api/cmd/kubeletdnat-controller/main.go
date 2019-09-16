package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/kubeletdnat"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	kubeconfigFlag := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	master := flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig")
	networkFlag := flag.String("node-access-network", "", "The network in CIDR notation to translate to.")
	chainNameFlag := flag.String("chain-name", "node-access-dnat", "Name of the chain in nat table.")
	vpnInterfaceFlag := flag.String("vpn-interface", "tun0", "Name of the vpn interface.")

	var logOptions kubermaticlog.Options
	flag.BoolVar(&logOptions.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&logOptions.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())

	flag.Parse()

	if err := logOptions.Validate(); err != nil {
		fmt.Printf("error occurred while validating zap logger options: %v\n", err)
		os.Exit(1)
	}

	rawLog := kubermaticlog.New(logOptions.Debug, kubermaticlog.Format(logOptions.Format))
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))
	log := rawLog.Sugar()

	_, network, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		log.Fatalw("node-access-network invalid or missing", "error", zap.Error(err))
	}
	nodeAccessNetwork := network.IP

	config, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfigFlag)
	if err != nil {
		log.Fatalw("Failed to build configs from flags", "error", zap.Error(err))
	}

	// Wait until the API server is actually up & the corev1 api groups is available.
	// This is a smallish hack to avoid dying instantly when running as sidecar to the kube API server
	// The API server takes a few seconds to start which makes this sidecar die immediately
	err = wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
		client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
		if err != nil {
			return false, nil
		}

		nodeList := &corev1.NodeList{}
		if err := client.List(context.Background(), &ctrlruntimeclient.ListOptions{}, nodeList); err != nil {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		log.Fatalw("Failed waiting for the API server to be alive", "error", zap.Error(err))
	}

	mgr, err := manager.New(config, manager.Options{})
	if err != nil {
		log.Fatalw("failed to create mgr", "error", zap.Error(err))
	}

	if err := kubeletdnat.Add(mgr, *chainNameFlag, nodeAccessNetwork, *vpnInterfaceFlag, log); err != nil {
		log.Fatalw("failed to add the kubelet dnat controller", "error", zap.Error(err))
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalw("Failed to start kubeletdnat controller", "error", zap.Error(err))
	}
}
