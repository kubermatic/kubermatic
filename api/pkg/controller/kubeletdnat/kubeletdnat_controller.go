package kubeletdnat

import (
	"bytes"
	"crypto/sha1"
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

	corev1 "k8s.io/api/core/v1"
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
	// QueueKey is the constant key added to the queue for deduplication.
	QueueKey = "some node"
)

var (
	preferredAddressTypes = []corev1.NodeAddressType{corev1.NodeExternalIP, corev1.NodeInternalIP}
)

// Controller updates iptable rules to match node addresses.
// Every node address gets a translation to the respective node-access (vpn) address.
type Controller struct {
	client     kubernetes.Interface
	nodeLister k8slistersV1.NodeLister
	queue      workqueue.RateLimitingInterface

	nodeTranslationChainName string
	nodeAccessNetwork        net.IP
}

// DnatRule stores address+port before translation (match) and
// provides address+port after translation.
type DnatRule struct {
	OriginalTargetAddress string
	OriginalTargetPort    string
	TranslatedAddress     string
	TranslatedPort        string
}

// Equals returns true if the rule equals the given rule.
func (rule *DnatRule) Equals(other *DnatRule) bool {
	if other == nil ||
		rule.OriginalTargetAddress != other.OriginalTargetAddress ||
		rule.OriginalTargetPort != other.OriginalTargetPort ||
		rule.TranslatedAddress != other.TranslatedAddress ||
		rule.TranslatedPort != other.TranslatedPort {
		return false
	}
	return true
}

// NewController creates a new controller for the specified data.
func NewController(
	client kubernetes.Interface,
	nodeInformer k8sinformersV1.NodeInformer,
	nodeTranslationChainName string,
	nodeAccessNetwork net.IP) *Controller {

	ctrl := &Controller{
		client:     client,
		nodeLister: nodeInformer.Lister(),
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "nodes"),
		nodeTranslationChainName: nodeTranslationChainName,
		nodeAccessNetwork:        nodeAccessNetwork,
	}

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { ctrl.queue.Add(QueueKey) },
		DeleteFunc: func(_ interface{}) { ctrl.queue.Add(QueueKey) },
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRule, oldErr := ctrl.getRuleFromNode(oldObj.(*corev1.Node))
			if oldErr != nil {
				runtime.HandleError(fmt.Errorf("failed to get rule from old node: %v", oldErr))
				return
			}
			newRule, newErr := ctrl.getRuleFromNode(newObj.(*corev1.Node))
			if newErr != nil {
				runtime.HandleError(fmt.Errorf("failed to get rule from new node: %v", newErr))
				return
			}

			if oldRule.Equals(newRule) {
				return
			}
			ctrl.queue.Add(QueueKey)
		},
	})
	return ctrl
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed.
func (ctrl *Controller) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	go wait.Until(func() { ctrl.queue.Add(QueueKey) }, time.Second*30, stopCh)
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
	err := ctrl.syncDnatRules(key.(string))
	ctrl.handleErr(err, key)
	return true
}

func (ctrl *Controller) getDesiredRules(nodes []*corev1.Node) []string {
	rules := []string{}
	for _, node := range nodes {
		rule, err := ctrl.getRuleFromNode(node)
		if err != nil {
			glog.Errorf("failed to get dnat rule from node %s: %v", node.Name, err)
			continue
		}
		rules = append(rules, rule.RestoreLine(ctrl.nodeTranslationChainName))
	}
	sort.Strings(rules)
	return rules
}

// syncDnatRules will recreate the complete set of translation rules
// based on the list of nodes.
func (ctrl *Controller) syncDnatRules(key string) error {
	glog.V(6).Infof("Syncing DNAT rules as %s got modified", key)

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
	rc, allActualRules, err := execSave()
	if rc != 0 || err != nil {
		return fmt.Errorf("failed to read iptable rules: %d / %v", rc, err)
	}
	// filter out everything that's not relevant for us
	actualRules, haveJump, haveMasquerade := filterDnatRules(allActualRules, ctrl.nodeTranslationChainName)

	actualHash := hashLines(actualRules)
	desiredHash := hashLines(desiredRules)
	if !bytes.Equal(desiredHash, actualHash) {
		// Need to update chain in kernel.
		glog.V(6).Infof("Updating iptables chain in kernel (%d rules).", len(desiredRules))
		if err := ctrl.applyRules(desiredRules); err != nil {
			return fmt.Errorf("failed to apply iptable rules: %v", err)
		}
	}

	// Ensure to jump into the translation chain.
	if !haveJump {
		if err := execIptables([]string{"-t", "nat", "-I", "OUTPUT", "-j", ctrl.nodeTranslationChainName}); err != nil {
			return fmt.Errorf("failed to create jump rule in OUTPUT chain: %v", err)
		}
		glog.V(2).Infof("Inserted OUTPUT rule to jump into chain %s.", ctrl.nodeTranslationChainName)
	}

	// Ensure to masquerade outgoing vpn packets.
	if !haveMasquerade {
		if err := execIptables([]string{"-t", "nat", "-I", "POSTROUTING", "-o", "tun0", "-j", "MASQUERADE"}); err != nil {
			return fmt.Errorf("failed to create masquerade rule in POSTROUTING chain: %v", err)
		}
		glog.V(2).Infof("Inserted POSTROUTING rule to masquerade vpn traffic.")
	}

	return nil
}

// getPreferredNodeAddress is behaving like k8s' nodeutil.GetPreferredNodeAddress:
// returns the address of the provided node, using the provided preference order.
func getPreferredNodeAddress(node *corev1.Node) (string, error) {
	for _, addressType := range preferredAddressTypes {
		for _, address := range node.Status.Addresses {
			if address.Type == addressType {
				return address.Address, nil
			}

		}
	}
	return "", fmt.Errorf("no preferred addresses found; known addresses: %v", node.Status.Addresses)
}

func getInternalNodeAddress(node *corev1.Node) (string, error) {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address, nil
		}
	}
	return "", fmt.Errorf("no internal address found; known addresses: %v", node.Status.Addresses)
}

// getRuleFromNode determines the used kubelet address of a node
// and creates a DnatRule from it.
func (ctrl *Controller) getRuleFromNode(node *corev1.Node) (*DnatRule, error) {
	if node == nil {
		return nil, fmt.Errorf("invalid/nil node reference")
	}
	rule := &DnatRule{}
	// Set matching part of the rule (original address).
	host, err := getPreferredNodeAddress(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferred node address: %v", err)
	}
	port := int(node.Status.DaemonEndpoints.KubeletEndpoint.Port)
	if port <= 0 {
		port = provider.DefaultKubeletPort
	}
	rule.OriginalTargetAddress = host
	rule.OriginalTargetPort = strconv.FormatInt(int64(port), 10)

	// Set translation part of the rule (new destination)
	//    This implements the current node-access-network translations by
	//    changing the first two octets of the node-ip-address into the
	//    respective two octets of the node-access-network.
	internalIP, err := getInternalNodeAddress(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get internal node address: %v", err)
	}
	octets := strings.Split(internalIP, ".")

	l := len(ctrl.nodeAccessNetwork)
	newAddress := fmt.Sprintf("%d.%d.%s.%s",
		ctrl.nodeAccessNetwork[l-4], ctrl.nodeAccessNetwork[l-3],
		octets[2], octets[3])
	rule.TranslatedAddress = newAddress
	rule.TranslatedPort = rule.OriginalTargetPort

	return rule, nil
}

// hashRules sorts and hashes the given rules. This is used to detect the changes.
func hashLines(lines []string) []byte {
	hasher := sha1.New()
	for _, s := range lines {
		if _, err := hasher.Write([]byte(s)); err != nil {
			glog.Errorf("failed to hash bytes: %v", err)
			return nil
		}
	}
	return hasher.Sum(nil)
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

func execSave() (int, []string, error) {
	cmd := exec.Command("iptables-save", []string{"-t", "nat"}...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, strings.Split(string(out), "\n"), nil
	}
	if xErr, ok := err.(*exec.ExitError); ok {
		wstat := xErr.Sys().(syscall.WaitStatus)
		if wstat.Exited() {
			return wstat.ExitStatus(), []string{}, nil
		}
	}
	return -1, []string{}, err
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
// rule's OriginalTargetAddress and Port.
func (rule *DnatRule) GetMatchArgs() []string {
	return []string{
		"-d", rule.OriginalTargetAddress + "/32",
		"-p", "tcp",
		"-m", "tcp",
		"--dport", rule.OriginalTargetPort,
	}
}

// GetTargetArgs returns iptables arguments to specify the
// rule's target after translation.
func (rule *DnatRule) GetTargetArgs() []string {
	var target string
	if len(rule.TranslatedAddress) > 0 {
		target = rule.TranslatedAddress
	}
	target = target + ":"
	if len(rule.TranslatedPort) > 0 {
		target = target + rule.TranslatedPort
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
func (rule *DnatRule) RestoreLine(chain string) string {
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
