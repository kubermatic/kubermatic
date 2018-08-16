package main

import (
	"flag"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	//kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	utilnode "k8s.io/kubernetes/pkg/util/node"
)

const (
	NodeTranslationChainName = "node-translation"
)

type DnatRule struct {
	OriginalTargetAddress string
	OriginalTargetPort    string
	GetTarget             func(*DnatRule) string
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig.")
	networkFlag := flag.String("node-access-network", "", "Node-Access-Network to translate to.")
	flag.Parse()

	nodeAccessNetwork, _, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		glog.Fatalf("node-access-network invalid or missing: %v", err)
	}

	translator := func(rule *DnatRule) string {
		octets := strings.Split(rule.OriginalTargetAddress, ".")

		l := len(nodeAccessNetwork)
		newAddress := fmt.Sprintf("%d.%d.%s.%s", nodeAccessNetwork[l-4], nodeAccessNetwork[l-3], octets[2], octets[3])
		return fmt.Sprintf("%s:%s", newAddress, rule.OriginalTargetPort)
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}
	// kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)

	client := kubernetes.NewForConfigOrDie(config)
	nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})

	stopChannel := make(chan struct{})
	glog.Infof("Successfully started.")

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

		rule := DnatRule{host, strconv.FormatInt(int64(port), 10), translator}
		exists, err := rule.Exists()
		if err != nil {
			glog.Fatalf("failed to check for rule existence: %v", err)
			continue
		}
		glog.V(6).Infof("node %s: address: %s:%d rule: %s exists: %v\n", node.Name, host, port, rule.String(), exists)
		if !exists {
			glog.V(3).Infof("inserting rule: %s", rule.String())
			rule.Insert()
		}
		glog.Flush()
	}

	close(stopChannel)

	<-stopChannel
	glog.Infof("Shutting down..")
}

func execIptables(cmdcode []string) (int, error) {
	cmd := exec.Command("iptables", cmdcode...)
	_, err := cmd.CombinedOutput()
	if err == nil {
		return 0, nil
	}
	if xErr, ok := err.(*exec.ExitError); ok {
		wstat := xErr.Sys().(syscall.WaitStatus)
		if wstat.Exited() {
			return wstat.ExitStatus(), nil
		}
	}
	return -1, err
}

func (rule *DnatRule) GetMatchArgs() []string {
	return []string{
		"-p", "tcp",
		"-d", rule.OriginalTargetAddress,
		"--dport", rule.OriginalTargetPort,
	}
}
func (rule *DnatRule) GetTargetArgs() []string {
	return []string{
		"-j", "DNAT",
		"--to", rule.GetTarget(rule),
	}
}
func (rule *DnatRule) Exists() (bool, error) {
	args := []string{"-t", "nat", "-C", NodeTranslationChainName}
	args = append(args, rule.GetMatchArgs()...)
	args = append(args, rule.GetTargetArgs()...)
	rc, err := execIptables(args)
	return rc == 0, err
}
func (rule *DnatRule) Insert() error {
	args := []string{"-t", "nat", "-I", NodeTranslationChainName}
	args = append(args, rule.GetMatchArgs()...)
	args = append(args, rule.GetTargetArgs()...)
	rc, err := execIptables(args)
	if err != nil {
		return err
	}
	if rc != 0 {
		return fmt.Errorf("iptables returned non-zero for: %v", args)
	}
	return nil
}
func (rule *DnatRule) String() string {
	repr := []string{}
	repr = append(repr, rule.GetMatchArgs()...)
	repr = append(repr, rule.GetTargetArgs()...)

	return fmt.Sprintf("(%s) -> (%s)",
		strings.Join(rule.GetMatchArgs(), " "),
		strings.Join(rule.GetTargetArgs(), " "))
}
