package main

import (
	"flag"
	"fmt"

	//kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilnode "k8s.io/kubernetes/pkg/util/node"
	utilexec "k8s.io/utils/exec"
)

const (
	NodeTranslationChain = "node-translation"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}
	// kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)

	client := kubernetes.NewForConfigOrDie(config)
	nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})

	stopChannel := make(chan struct{})
	glog.Infof("Successfully started.")

	iptrun := utiliptables.New(utilexec.New(), utildbus.New(), utiliptables.ProtocolIpv4)
	ver, err := iptrun.GetVersion()
	if err != nil {
		glog.Fatalf("failed to get iptables version")
	}

	glog.V(2).Infof("iptables version: %s", ver)

	chain := utiliptables.Chain(NodeTranslationChain)
	existed, err := iptrun.EnsureChain(utiliptables.TableNAT, chain)
	if !existed {
		iptrun.EnsureRule(utiliptables.Prepend, utiliptables.TableNAT, utiliptables.ChainOutput, "-j", string(chain))
	}

	// TODO make sure this matches the actually used values
	// (maybe a const which is then also used to create the apiserver deployment)
	preferredAddressTypes := []corev1.NodeAddressType{corev1.NodeExternalIP}
	for _, node := range nodeList.Items {
		host, err := utilnode.GetPreferredNodeAddress(&node, preferredAddressTypes)
		if err != nil {
			glog.Errorf("failed to get preferred node address for node %s", node.Name)
			continue
		}

		port := int(node.Status.DaemonEndpoints.KubeletEndpoint.Port)
		if port <= 0 {
			port = provider.DefaultKubeletPort
		}

		fmt.Printf("node %s: %s:%d\n", node.Name, host, port)
		glog.Flush()
	}

	close(stopChannel)

	<-stopChannel
	glog.Infof("Shutting down..")
}
