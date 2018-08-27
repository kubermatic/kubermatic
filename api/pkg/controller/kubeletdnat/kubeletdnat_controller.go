package kubeletdnat

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sinformersV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	k8slistersV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// queueKey is the constant key added to the queue for deduplication.
	queueKey = "some node"
)

// Controller updates iptable rules to match node addresses.
// Every node address gets a translation to the respective node-access (vpn) address.
type Controller struct {
	client     kubernetes.Interface
	nodeLister k8slistersV1.NodeLister
	queue      workqueue.RateLimitingInterface

	nodeTranslationChainName string
	nodeAccessNetwork        net.IP
	vpnInterface             string
}

// dnatRule stores address+port before translation (match) and
// provides address+port after translation.
type dnatRule struct {
	originalTargetAddress string
	originalTargetPort    string
	translatedAddress     string
	translatedPort        string
}

// NewController creates a new controller for the specified data.
func NewController(
	client kubernetes.Interface,
	nodeInformer k8sinformersV1.NodeInformer,
	nodeTranslationChainName string,
	nodeAccessNetwork net.IP,
	vpnInterface string) *Controller {

	ctrl := &Controller{
		client:     client,
		nodeLister: nodeInformer.Lister(),
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "nodes"),
		nodeTranslationChainName: nodeTranslationChainName,
		nodeAccessNetwork:        nodeAccessNetwork,
		vpnInterface:             vpnInterface,
	}

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { ctrl.queue.Add(queueKey) },
		DeleteFunc: func(_ interface{}) { ctrl.queue.Add(queueKey) },
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode := oldObj.(*corev1.Node)
			newNode := newObj.(*corev1.Node)
			if equality.Semantic.DeepEqual(oldNode.Status.Addresses, newNode.Status.Addresses) {
				return
			}
			ctrl.queue.Add(queueKey)
		},
	})
	return ctrl
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed.
func (ctrl *Controller) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	go wait.Until(func() { ctrl.queue.Add(queueKey) }, time.Second*30, stopCh)
	go wait.Until(ctrl.runWorker, time.Second, stopCh)
	<-stopCh
}

// handleErr checks if an error happened and makes sure we will retry later.
func (ctrl *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		ctrl.queue.Forget(key)
		return
	}

	glog.V(0).Infof("Error syncing node access dnat rules %v: %v", key, err)

	// Re-enqueue the key rate limited. Based on the rate limiter on the
	// queue and the re-enqueue history, the key will be processed later again.
	ctrl.queue.AddRateLimited(key)
}

func (ctrl *Controller) runWorker() {
	for ctrl.processNextItem() {
	}
}
func (ctrl *Controller) processNextItem() bool {
	key, quit := ctrl.queue.Get()
	if quit {
		return false
	}

	defer ctrl.queue.Done(key)
	err := ctrl.syncDnatRules()
	ctrl.handleErr(err, key)
	return true
}

func (ctrl *Controller) getDesiredRules(nodes []*corev1.Node) []string {
	rules := []string{}
	for _, node := range nodes {
		for _, rule := range ctrl.getRulesForNode(node) {
			rules = append(rules, rule.RestoreLine(ctrl.nodeTranslationChainName))
		}
	}
	sort.Strings(rules)
	return rules
}

// syncDnatRules will recreate the complete set of translation rules
// based on the list of nodes.
func (ctrl *Controller) syncDnatRules() error {
	glog.V(6).Infof("Syncing DNAT rules")

	// Get nodes from lister, make a copy.
	cachedNodes, err := ctrl.nodeLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to receive nodes from lister: %v", err)
	}
	nodes := make([]*corev1.Node, len(cachedNodes))
	for i := range cachedNodes {
		nodes[i] = cachedNodes[i].DeepCopy()
	}

	// Create the set of rules from all listed nodes.
	desiredRules := ctrl.getDesiredRules(nodes)

	// Get the actual state (current iptable rules)
	allActualRules, err := execSave()
	if err != nil {
		return fmt.Errorf("failed to read iptable rules: %v", err)
	}
	// filter out everything that's not relevant for us
	actualRules, haveJump, haveMasquerade := filterDnatRules(allActualRules, ctrl.nodeTranslationChainName)

	if !equality.Semantic.DeepEqual(actualRules, desiredRules) {
		// Need to update chain in kernel.
		glog.V(6).Infof("Updating iptables chain in kernel (%d rules).", len(desiredRules))
		if err := ctrl.applyRules(desiredRules); err != nil {
			return fmt.Errorf("failed to apply iptable rules: %v", err)
		}
	}

	// Ensure to jump into the translation chain.
	if !haveJump && len(desiredRules) > 0 {
		if err := execIptables([]string{"-t", "nat", "-I", "OUTPUT", "-j", ctrl.nodeTranslationChainName}); err != nil {
			return fmt.Errorf("failed to create jump rule in OUTPUT chain: %v", err)
		}
		glog.V(2).Infof("Inserted OUTPUT rule to jump into chain %s.", ctrl.nodeTranslationChainName)
	}

	// Ensure to masquerade outgoing vpn packets.
	if !haveMasquerade {
		if err := execIptables([]string{"-t", "nat", "-I", "POSTROUTING", "-o", ctrl.vpnInterface, "-j", "MASQUERADE"}); err != nil {
			return fmt.Errorf("failed to create masquerade rule in POSTROUTING chain: %v", err)
		}
		glog.V(2).Infof("Inserted POSTROUTING rule to masquerade vpn traffic.")
	}

	return nil
}

// getNodeAddresses returns all relevant addresses of a node.
func getNodeAddresses(node *corev1.Node) []string {
	addressTypes := []corev1.NodeAddressType{corev1.NodeExternalIP, corev1.NodeInternalIP}
	addresses := []string{}
	for _, addressType := range addressTypes {
		for _, address := range node.Status.Addresses {
			if address.Type == addressType {
				addresses = append(addresses, address.Address)
			}

		}
	}
	return addresses
}

func getInternalNodeAddress(node *corev1.Node) (string, error) {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address, nil
		}
	}
	return "", fmt.Errorf("no internal address found; known addresses: %v", node.Status.Addresses)
}

// getRulesForNode determines the used kubelet address of a node
// and creates a dnatRule from it.
func (ctrl *Controller) getRulesForNode(node *corev1.Node) []*dnatRule {
	rules := []*dnatRule{}

	port := int(node.Status.DaemonEndpoints.KubeletEndpoint.Port)
	if port <= 0 {
		port = provider.DefaultKubeletPort
	}

	internalIP, err := getInternalNodeAddress(node)
	if err != nil {
		glog.Errorf("failed to get internal node address: %v", err)
		return rules
	}
	octets := strings.Split(internalIP, ".")

	for _, address := range getNodeAddresses(node) {
		rule := &dnatRule{}

		// Set matching part of the rule (original address).
		rule.originalTargetAddress = address
		rule.originalTargetPort = strconv.FormatInt(int64(port), 10)

		// Set translation part of the rule (new destination)
		//    This implements the current node-access-network translations by
		//    changing the first two octets of the node-ip-address into the
		//    respective two octets of the node-access-network.
		//    The last two octets are the last two octets of the internal address
		l := len(ctrl.nodeAccessNetwork)
		newAddress := fmt.Sprintf("%d.%d.%s.%s",
			ctrl.nodeAccessNetwork[l-4], ctrl.nodeAccessNetwork[l-3],
			octets[2], octets[3])
		rule.translatedAddress = newAddress
		rule.translatedPort = rule.originalTargetPort

		rules = append(rules, rule)
	}
	return rules
}

// applyRules creates a iptables-save file and pipes it to stdin of
// a iptables-restore process for atomically setting new rules.
// This function replaces a complete chain (removing all pre-existing rules).
func (ctrl *Controller) applyRules(rules []string) error {
	restore := []string{"*nat", fmt.Sprintf(":%s - [0:0]", ctrl.nodeTranslationChainName)}
	restore = append(restore, rules...)
	restore = append(restore, "COMMIT")

	return execRestore(restore)
}

func execIptables(args []string) error {
	cmd := exec.Command("iptables", args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if len(out) > 0 {
		return fmt.Errorf("iptables with arguments %v failed: %v (output: %s)", args, err, string(out))
	}
	return fmt.Errorf("iptables with arguments %v failed: %v", args, err)
}

func execSave() ([]string, error) {
	cmd := exec.Command("iptables-save", []string{"-t", "nat"}...)
	out, err := cmd.CombinedOutput()
	return strings.Split(string(out), "\n"), err
}

func execRestore(rules []string) error {
	cmd := exec.Command("iptables-restore", []string{"--noflush", "-v", "-T", "nat"}...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if _, err := io.WriteString(stdin, strings.Join(rules, "\n")+"\n"); err != nil {
		return fmt.Errorf("failed to write to iptables-restore stdin: %v", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close iptables-restore stdin: %v", err)
	}

	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if len(out) > 0 {
		return fmt.Errorf("iptables-restore failed: %v (output: %s)", err, string(out))
	}
	return fmt.Errorf("iptables-restore failed: %v", err)
}

// GetMatchArgs returns iptables arguments to match for the
// rule's originalTargetAddress and Port.
func (rule *dnatRule) GetMatchArgs() []string {
	return []string{
		"-d", rule.originalTargetAddress + "/32",
		"-p", "tcp",
		"-m", "tcp",
		"--dport", rule.originalTargetPort,
	}
}

// GetTargetArgs returns iptables arguments to specify the
// rule's target after translation.
func (rule *dnatRule) GetTargetArgs() []string {
	var target string
	if len(rule.translatedAddress) > 0 {
		target = rule.translatedAddress
	}
	target = target + ":"
	if len(rule.translatedPort) > 0 {
		target = target + rule.translatedPort
	}
	if len(target) == 0 {
		return []string{}
	}
	return []string{
		"-j", "DNAT",
		"--to-destination", target,
	}
}

// RestoreLine returns a line of `iptables-save`-file representing
// the rule.
func (rule *dnatRule) RestoreLine(chain string) string {
	args := []string{"-A", chain}
	args = append(args, rule.GetMatchArgs()...)
	args = append(args, rule.GetTargetArgs()...)
	return strings.Join(args, " ")
}

// filterDnatRules enumerates through all given rules and returns all
// rules matching the given chain. It also returns two booleans to
// indicate if the jump and the masquerade rule are present.
func filterDnatRules(rules []string, chain string) ([]string, bool, bool) {
	out := []string{}
	haveJump := false
	haveMasquerade := false

	rulePrefix := fmt.Sprintf("-A %s ", chain)
	jumpPattern := fmt.Sprintf("-A OUTPUT -j %s", chain)
	masqPattern := fmt.Sprintf("-A POSTROUTING -o tun0 -j MASQUERADE")
	for _, rule := range rules {
		if rule == jumpPattern {
			haveJump = true
		}
		if rule == masqPattern {
			haveMasquerade = true
		}
		if !strings.HasPrefix(rule, rulePrefix) {
			continue
		}
		out = append(out, rule)
	}
	return out, haveJump, haveMasquerade
}
