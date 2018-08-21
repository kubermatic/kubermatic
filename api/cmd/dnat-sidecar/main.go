package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/signals"

	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type controllerRunOptions struct {
	kubeconfig string
}

func main() {
	ctrl := DnatController{}
	flag.StringVar(&ctrl.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	networkFlag := flag.String("node-access-network", "", "Node-Access-Network to translate to.")
	flag.Parse()

	ctrl.stopCh = signals.SetupSignalHandler()

	nodeAccessNetwork, _, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		glog.Fatalf("node-access-network invalid or missing: %v", err)
	}

	// This implements the current node-access-network translations by
	// changing the first two octets of the node-ip-address into the
	// respective two octets of the node-access-network.
	ctrl.dnatTranslator = func(rule *DnatRule) string {
		octets := strings.Split(rule.OriginalTargetAddress, ".")

		l := len(nodeAccessNetwork)
		newAddress := fmt.Sprintf("%d.%d.%s.%s", nodeAccessNetwork[l-4], nodeAccessNetwork[l-3], octets[2], octets[3])
		return fmt.Sprintf("%s:%s", newAddress, rule.OriginalTargetPort)
	}

	config, err := clientcmd.BuildConfigFromFlags("", ctrl.runOptions.kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}
	ctrl.kubeMasterClient = kubernetes.NewForConfigOrDie(config)
	ctrl.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(ctrl.kubeMasterClient, time.Minute*1)

	ctrl.nodeInformer = ctrl.kubeMasterInformerFactory.Core().V1().Nodes()
	ctrl.nodeLister = ctrl.nodeInformer.Lister()

	ctrl.kubeMasterInformerFactory.Start(ctrl.stopCh)
	ctrl.kubeMasterInformerFactory.WaitForCacheSync(ctrl.stopCh)

	glog.V(6).Infof("Successfully started.")

	// Recreate iptables chain based on timer
	ctrl.timer = time.NewTimer(15 * time.Second)
	go func() {
		for {
			<-ctrl.timer.C
			ctrl.handle("timer", nil, nil)
		}
	}()

	// Recreate iptables chaing based on add/update/delete events
	ctrl.nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { ctrl.handle("add", nil, new) },
		UpdateFunc: func(old, new interface{}) { ctrl.handle("update", old, new) },
		DeleteFunc: func(old interface{}) { ctrl.handle("delete", old, nil) },
	})

	<-ctrl.stopCh
	glog.Infof("Shutting down..")
}
