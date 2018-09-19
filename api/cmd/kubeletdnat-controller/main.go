package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"github.com/golang/glog"
	"github.com/oklog/run"

	"github.com/kubermatic/kubermatic/api/pkg/controller/kubeletdnat"

	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfigFlag := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	networkFlag := flag.String("node-access-network", "", "The network in CIDR notation to translate to.")
	chainNameFlag := flag.String("chain-name", "node-access-dnat", "Name of the chain in nat table.")
	vpnInterfaceFlag := flag.String("vpn-interface", "tun0", "Name of the vpn interface.")
	flag.Parse()

	nodeAccessNetwork, _, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		glog.Fatalf("node-access-network invalid or missing: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigFlag)
	if err != nil {
		glog.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}

	kubeInformerFactory := coreinformers.NewSharedInformerFactory(client, informer.DefaultInformerResyncPeriod)

	ctrl := kubeletdnat.NewController(
		client,
		kubeInformerFactory.Core().V1().Nodes(),
		*chainNameFlag,
		nodeAccessNetwork,
		*vpnInterfaceFlag)

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	ctx, cancel := context.WithCancel(context.Background())
	var gr run.Group
	{
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		gr.Add(func() error {
			<-sig
			return nil
		}, func(err error) {
			cancel()
			close(sig)
		})
	}
	{
		gr.Add(func() error {
			ctrl.Run(ctx.Done())
			return nil
		}, func(err error) {
			cancel()
		})
	}

	if err := gr.Run(); err != nil {
		glog.Fatal(err)
	}
}
