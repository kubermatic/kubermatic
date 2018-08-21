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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

type controllerRunOptions struct {
	kubeconfig string
}

var (
	kubeconfig string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	networkFlag := flag.String("node-access-network", "", "Node-Access-Network to translate to.")
	flag.Parse()

	nodeAccessNetwork, _, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		glog.Fatalf("node-access-network invalid or missing: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}

	kubeInformerFactory := coreinformers.NewSharedInformerFactory(client, time.Minute*5)

	ctrl := DnatController{
		client:     client,
		nodeLister: kubeInformerFactory.Core().V1().Nodes().Lister(),
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "nodes"),

		// This implements the current node-access-network translations by
		// changing the first two octets of the node-ip-address into the
		// respective two octets of the node-access-network.
		dnatTranslator: func(rule *DnatRule) string {
			octets := strings.Split(rule.OriginalTargetAddress, ".")

			l := len(nodeAccessNetwork)
			newAddress := fmt.Sprintf("%d.%d.%s.%s",
				nodeAccessNetwork[l-4], nodeAccessNetwork[l-3],
				octets[2], octets[3])
			return fmt.Sprintf("%s:%s", newAddress, rule.OriginalTargetPort)
		},
	}

	// Recreate iptables chaing based on add/update/delete events
	kubeInformerFactory.Core().V1().Nodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { ctrl.enqueue(obj.(*corev1.Node)) },
		UpdateFunc: func(_, newObj interface{}) { ctrl.enqueue(newObj.(*corev1.Node)) },
		DeleteFunc: func(obj interface{}) {
			n, ok := obj.(*corev1.Node)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				n, ok = tombstone.Obj.(*corev1.Node)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Service %#v", obj))
					return
				}
			}
			ctrl.enqueue(n)
		},
	})

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
