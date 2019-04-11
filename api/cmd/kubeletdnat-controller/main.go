package main

import (
	"flag"
	"net"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/kubeletdnat"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	kubeconfigFlag := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	master := flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig")
	networkFlag := flag.String("node-access-network", "", "The network in CIDR notation to translate to.")
	chainNameFlag := flag.String("chain-name", "node-access-dnat", "Name of the chain in nat table.")
	vpnInterfaceFlag := flag.String("vpn-interface", "tun0", "Name of the vpn interface.")
	flag.Parse()

	nodeAccessNetwork, _, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		glog.Fatalf("node-access-network invalid or missing: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfigFlag)
	if err != nil {
		glog.Fatal(err)
	}

	// Wait until the API server is actually up
	// This is a smallish hack to avoid dying instantly when running as sidecar to the kube API server
	// The API server takes a few seconds to start which makes this sidecar die immediately
	err = wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
		_, err := client.New(config, client.Options{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		glog.Fatalf("Failed waiting for the API server to be alive")
	}

	mgr, err := manager.New(config, manager.Options{})
	if err != nil {
		glog.Fatalf("failed to create mgr: %v", err)
	}

	if err := kubeletdnat.Add(mgr, *chainNameFlag, nodeAccessNetwork, *vpnInterfaceFlag); err != nil {
		glog.Fatalf("failed to add the kubelet dnat controller: %v", err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatal(err)
	}
}
