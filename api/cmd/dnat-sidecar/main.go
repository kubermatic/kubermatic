package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kuberinformers "k8s.io/client-go/informers"
	k8sinformersV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	k8slistersV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// NodeTranslationChainName is the name of the iptables chain holding the translation rules.
	NodeTranslationChainName = "node-translation"
)

type controllerRunOptions struct {
	kubeconfig string
}
type controllerContext struct {
	runOptions                controllerRunOptions
	stopCh                    <-chan struct{}
	kubeMasterClient          kubernetes.Interface
	kubeMasterInformerFactory kuberinformers.SharedInformerFactory
	nodeInformer              k8sinformersV1.NodeInformer
	nodeLister                k8slistersV1.NodeLister
	timer                     *time.Timer
	dnatTranslator            func(rule *DnatRule) string
	rulesHash                 []byte
}

type DnatRule struct {
	OriginalTargetAddress string
	OriginalTargetPort    string
	Translate             func(*DnatRule) string
}

func main() {
	ctrlCtx := controllerContext{}
	flag.StringVar(&ctrlCtx.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	networkFlag := flag.String("node-access-network", "", "Node-Access-Network to translate to.")
	flag.Parse()

	ctrlCtx.stopCh = signals.SetupSignalHandler()

	nodeAccessNetwork, _, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		glog.Fatalf("node-access-network invalid or missing: %v", err)
	}

	// This implements the current node-access-network translations by
	// changing the first two octets of the node-ip-address into the
	// respective two octets of the node-access-network.
	ctrlCtx.dnatTranslator = func(rule *DnatRule) string {
		octets := strings.Split(rule.OriginalTargetAddress, ".")

		l := len(nodeAccessNetwork)
		newAddress := fmt.Sprintf("%d.%d.%s.%s", nodeAccessNetwork[l-4], nodeAccessNetwork[l-3], octets[2], octets[3])
		return fmt.Sprintf("%s:%s", newAddress, rule.OriginalTargetPort)
	}

	config, err := clientcmd.BuildConfigFromFlags("", ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}
	ctrlCtx.kubeMasterClient = kubernetes.NewForConfigOrDie(config)
	ctrlCtx.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(ctrlCtx.kubeMasterClient, time.Minute*1)

	ctrlCtx.nodeInformer = ctrlCtx.kubeMasterInformerFactory.Core().V1().Nodes()
	ctrlCtx.nodeLister = ctrlCtx.nodeInformer.Lister()

	ctrlCtx.kubeMasterInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)

	glog.V(6).Infof("Successfully started.")

	// Recreate iptables chain based on timer
	ctrlCtx.timer = time.NewTimer(15 * time.Second)
	go func() {
		for {
			<-ctrlCtx.timer.C
			ctrlCtx.handle("timer", nil, nil)
		}
	}()

	// Recreate iptables chaing based on add/update/delete events
	ctrlCtx.nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { ctrlCtx.handle("add", nil, new) },
		UpdateFunc: func(old, new interface{}) { ctrlCtx.handle("update", old, new) },
		DeleteFunc: func(old interface{}) { ctrlCtx.handle("delete", old, nil) },
	})

	<-ctrlCtx.stopCh
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

func execRestore(rules []string) (int, error) {
	cmd := exec.Command("iptables-restore", []string{"--noflush", "-v", "-T", "nat"}...)
	//cmd := exec.Command("dd", []string{"of=/tmp/fake-restore"}...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return -1, err
	}
	io.WriteString(stdin, strings.Join(rules, "\n"))
	io.WriteString(stdin, "\n")
	stdin.Close()

	_, err = cmd.CombinedOutput()
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
	if rule.Translate == nil {
		return []string{}
	}
	return []string{
		"-j", "DNAT",
		"--to", rule.Translate(rule),
	}
}
func (rule *DnatRule) RestoreLine(chain string) string {
	args := []string{"-A", chain}
	args = append(args, rule.GetMatchArgs()...)
	args = append(args, rule.GetTargetArgs()...)
	return strings.Join(args, " ")
}

func (rule *DnatRule) Insert(chain string) error {
	args := []string{"-t", "nat", "-I", chain}
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

// handle is called on add/update/delete and timer events.
func (ctrl *controllerContext) handle(reason string, oldobj, newobj interface{}) {
	ctrl.timer.Reset(15 * time.Second)

	// newNode, okNewNode := newobj.(*corev1.Node)
	// oldNode, okOldNode := oldobj.(*corev1.Node)
	// newNode/oldNode is no longer used as we are now creating
	// a complete ruleset all the time and comparing for changes.

	// Check that we have a rule in OUTPUT chain which jumps to our node-translation chain
	ensureJump()

	nodes, err := ctrl.nodeLister.List(labels.Everything())
	if err != nil {
		glog.Errorf("failed to list nodes: %v", err)
		return
	}
	glog.V(6).Infof("handling %d nodes in \"%s\" call.", len(nodes), reason)

	// Create the set of rules from all listed nodes.
	allRules := []*DnatRule{}
	for _, node := range nodes {
		rule, err := getRuleFromNode(node)
		if err != nil {
			glog.Errorf("failed to get dnat rule from node %s: %v", node.Name, err)
			continue
		}
		rule.Translate = ctrl.dnatTranslator
		allRules = append(allRules, rule)
	}

	// Comparing to previous controller state (not against actual chain in kernel)
	// and apply ruleset if it differs.
	currentHash := hashRules(allRules)
	if !bytes.Equal(currentHash, ctrl.rulesHash) {
		// rules changed, need to update chain in kernel
		glog.V(6).Infof("updating iptables chain in kernel...")
		if err := applyRules(allRules); err != nil {
			glog.Errorf("failed to apply iptable rules: %v", err)
			return
		}
		ctrl.rulesHash = currentHash
	}
}

// getPreferredNodeAddress is behaving like k8s' nodeutil.GetPreferredNodeAddress:
// returns the address of the provided node, using the provided preference order.
func getPreferredNodeAddress(node *corev1.Node) (string, error) {
	// TODO make sure this matches the actually used values
	// (maybe a const which is then also used to create the apiserver deployment)
	preferredAddressTypes := []corev1.NodeAddressType{corev1.NodeExternalIP}
	for _, addressType := range preferredAddressTypes {
		for _, address := range node.Status.Addresses {
			if address.Type == addressType {
				return address.Address, nil
			}

		}
	}
	return "", fmt.Errorf("no preferred addresses found; known addresses: %v", node.Status.Addresses)
}

// getRuleFromNode determines the used kubelet address of a node
// and creates a DnatRule from it.
func getRuleFromNode(node *corev1.Node) (*DnatRule, error) {
	host, err := getPreferredNodeAddress(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferred node address: %v", err)
	}
	port := int(node.Status.DaemonEndpoints.KubeletEndpoint.Port)
	if port <= 0 {
		port = provider.DefaultKubeletPort
	}
	return &DnatRule{host, strconv.FormatInt(int64(port), 10), nil}, nil
}

// hashRules sorts and hashes the given rules. This is used
// to detect the changes.
func hashRules(rules []*DnatRule) []byte {
	ruleStrings := make([]string, len(rules), len(rules))
	for _, rule := range rules {
		ruleStrings = append(ruleStrings, rule.RestoreLine(NodeTranslationChainName))
	}
	sort.Strings(ruleStrings)

	hasher := sha1.New()
	for _, s := range ruleStrings {
		hasher.Write([]byte(s))
	}
	return hasher.Sum(nil)
}

// applyRules creates a iptables-save file and pipes it to stdin of
// a iptables-restore process for atomically setting new rules.
func applyRules(rules []*DnatRule) error {
	restore := []string{"*nat", fmt.Sprintf(":%s - [0:0]", NodeTranslationChainName)}
	for _, rule := range rules {
		restore = append(restore, rule.RestoreLine(NodeTranslationChainName))
	}
	restore = append(restore, "COMMIT")

	rc, err := execRestore(restore)
	if err != nil {
		return err
	}
	if rc != 0 {
		return fmt.Errorf("iptables-restore returned non-zero for: %d", rc)
	}
	return nil
}

// ensureJump checks for the existens of a `-j node-translation` rule
// in OUTPUT chain and creates it if missing.
func ensureJump() error {
	// Check for the rule which jumps to the node-translation chain
	rc, err := execIptables([]string{
		"-t", "nat",
		"-C", "OUTPUT",
		"-j", NodeTranslationChainName,
	})
	if err != nil {
		return fmt.Errorf("failed to check for jump rule: %v", err)
	}
	if rc != 0 { // rule does not exist, create it
		rc2, err := execIptables([]string{
			"-t", "nat",
			"-I", "OUTPUT",
			"-j", NodeTranslationChainName,
		})
		if err != nil || rc2 != 0 {
			return fmt.Errorf("failed (%d) to insert jump rule: %v", rc2, err)
		}
		glog.V(2).Infof("Inserted OUTPUT rule to jump into node-translation.")
	}
	return nil
}
