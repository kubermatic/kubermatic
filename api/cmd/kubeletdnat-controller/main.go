package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	networkFlag := flag.String("node-access-network", "", "Node-Access-Network to translate to.")
	chainNameFlag := flag.String("chain-name", "node-access-dnat", "Name of the chain in nat table.")
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

	kubeInformerFactory := coreinformers.NewSharedInformerFactory(client, time.Minute*5)

	ctrl := kubeletdnat.NewController(
		client,
		kubeInformerFactory.Core().V1().Nodes(),

		// This implements the current node-access-network translations by
		// changing the first two octets of the node-ip-address into the
		// respective two octets of the node-access-network.
		func(rule *kubeletdnat.DnatRule) string {
			octets := strings.Split(rule.OriginalTargetAddress, ".")

			l := len(nodeAccessNetwork)
			newAddress := fmt.Sprintf("%d.%d.%s.%s",
				nodeAccessNetwork[l-4], nodeAccessNetwork[l-3],
				octets[2], octets[3])
			return fmt.Sprintf("%s:%s", newAddress, rule.OriginalTargetPort)
		},

		*chainNameFlag)

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
	glog.V(6).Infof("Factory started.")

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
