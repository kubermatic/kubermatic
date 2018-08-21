package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
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
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	k8slistersV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// TODO:
// cleanup leftovers
// use comments for rules (also better matching)
// compare to real-world state (actual rules in kernel)

const (
	// NodeTranslationChainName is the name of the iptables chain holding the translation rules.
	NodeTranslationChainName = "node-translation"
)

type DnatController struct {
	queue  workqueue.RateLimitingInterface
	client kubernetes.Interface

	stopCh                    <-chan struct{}
	kubeMasterInformerFactory kuberinformers.SharedInformerFactory
	nodeLister                k8slistersV1.NodeLister

	dnatTranslator func(rule *DnatRule) string
	rulesHash      []byte
}

// DnatRule stores address+port before translation (match) and
// provides address+port after translation.
type DnatRule struct {
	OriginalTargetAddress string
	OriginalTargetPort    string
	Translate             func(*DnatRule) string
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed.
func (ctrl *DnatController) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	go wait.Until(ctrl.runWorker, time.Second, stopCh)
	<-stopCh
}

// handleErr checks if an error happened and makes sure we will retry later.
func (ctrl *DnatController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		ctrl.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if ctrl.queue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing node access dnat rules %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		ctrl.queue.AddRateLimited(key)
		return
	}

	ctrl.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the queue: %v", key, err)
}

func (ctrl *DnatController) runWorker() {
	for ctrl.processNextItem() {
	}
}
func (ctrl *DnatController) processNextItem() bool {
	key, quit := ctrl.queue.Get()
	if quit {
		return false
	}

	defer ctrl.queue.Done(key)
	err := ctrl.syncDnatRules(key.(string))
	ctrl.handleErr(err, key)
	return true
}

func (ctrl *DnatController) enqueue(n *corev1.Node) {
	ctrl.enqueueAfter(n, 0)
}
func (ctrl *DnatController) enqueueAfter(n *corev1.Node, duration time.Duration) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(n)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", n, err))
		return
	}
	ctrl.queue.AddAfter(key, duration)
}

// syncDnatRules will recreate the complete set of translation rules
// based on the list of nodes.
func (ctrl *DnatController) syncDnatRules(key string) error {
	glog.V(6).Infof("Syncing DNAT rules as %s got modified", key)

	// Check that we have a rule in OUTPUT chain which jumps to our node-translation chain
	if err := ensureJump(); err != nil {
		return fmt.Errorf("failed to ensure jump-rule in OUTPUT chain: %v", err)
	}

	cachedNodes, err := ctrl.nodeLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to receive nodes from lister: %v", err)
	}
	nodes := make([]*corev1.Node, len(cachedNodes))
	for i := range cachedNodes {
		nodes[i] = cachedNodes[i].DeepCopy()
	}

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
			return fmt.Errorf("failed to apply iptable rules: %v", err)
		}
		ctrl.rulesHash = currentHash
	}
	return nil
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
	ruleStrings := make([]string, len(rules))
	for _, rule := range rules {
		ruleStrings = append(ruleStrings, rule.RestoreLine(NodeTranslationChainName))
	}
	sort.Strings(ruleStrings)

	hasher := sha1.New()
	for _, s := range ruleStrings {
		if _, err := hasher.Write([]byte(s)); err != nil {
			glog.Errorf("failed to hash bytes: %v", err)
			return nil
		}
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
		return err
	}
	if rc != 0 { // rule does not exist, create it
		rc2, err := execIptables([]string{
			"-t", "nat",
			"-I", "OUTPUT",
			"-j", NodeTranslationChainName,
		})
		if err != nil || rc2 != 0 {
			return err
		}
		glog.V(2).Infof("Inserted OUTPUT rule to jump into node-translation.")
	}
	return nil
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
	if _, err := io.WriteString(stdin, strings.Join(rules, "\n")+"\n"); err != nil {
		return -1, fmt.Errorf("failed to write to iptables-restore stdin: %v", err)
	}
	if err := stdin.Close(); err != nil {
		return -1, fmt.Errorf("failed to close iptables-restore stdin: %v", err)
	}

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

// GetMatchArgs returns iptables arguments to match for the
// rule's OriginalTargetAddress and Port.
func (rule *DnatRule) GetMatchArgs() []string {
	return []string{
		"-p", "tcp",
		"-d", rule.OriginalTargetAddress,
		"--dport", rule.OriginalTargetPort,
	}
}

// GetTargetArgs returns iptables arguments to specify the
// rule's target after translation.
func (rule *DnatRule) GetTargetArgs() []string {
	if rule.Translate == nil {
		return []string{}
	}
	return []string{
		"-j", "DNAT",
		"--to", rule.Translate(rule),
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

// Insert executes iptables binary and inserts the rule.
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
