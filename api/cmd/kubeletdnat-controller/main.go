package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/kubeletdnat"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	kubeconfigFlag := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	master := flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig")
	networkFlag := flag.String("node-access-network", "", "The network in CIDR notation to translate to.")
	chainNameFlag := flag.String("chain-name", "node-access-dnat", "Name of the chain in nat table.")
	vpnInterfaceFlag := flag.String("vpn-interface", "tun0", "Name of the vpn interface.")

	var logOptions kubermaticlog.Options
	logOptions.Debug = *flag.Bool("log-debug", false, "Enables debug logging")
	logOptions.Format = *flag.String("log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())

	flag.Parse()

	if err := logOptions.Validate(); err != nil {
		fmt.Printf("error occurred while validating zap logger options: %v\n", err)
		os.Exit(1)
	}

	rawLog := kubermaticlog.New(logOptions.Debug, kubermaticlog.Format(logOptions.Format))
	log := rawLog.Sugar()

	_, network, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		log.Fatalf("node-access-network invalid or missing: %v", err)
	}
	nodeAccessNetwork := network.IP

	config, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfigFlag)
	if err != nil {
		log.Fatal(err)
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
		log.Fatalf("Failed waiting for the API server to be alive")
	}

	mgr, err := manager.New(config, manager.Options{})
	if err != nil {
		log.Fatalf("failed to create mgr: %v", err)
	}

	if err := kubeletdnat.Add(mgr, *chainNameFlag, nodeAccessNetwork, log, *vpnInterfaceFlag); err != nil {
		log.Fatalf("failed to add the kubelet dnat controller: %v", err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal(err)
	}
}
