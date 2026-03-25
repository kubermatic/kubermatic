/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubeletdnatcontroller

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-kubeletdnat-controller"
)

// Reconciler updates iptable rules to match node addresses.
// Every node address gets a translation to the respective node-access (vpn) address.
type Reconciler struct {
	ctrlruntimeclient.Client

	nodeTranslationChainName string
	nodeAccessNetwork        net.IP
	vpnInterface             string

	log *zap.SugaredLogger
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
func Add(
	mgr manager.Manager,
	nodeTranslationChainName string,
	nodeAccessNetwork net.IP,
	log *zap.SugaredLogger,
	vpnInterface string,
) error {
	reconciler := &Reconciler{
		Client:                   mgr.GetClient(),
		nodeTranslationChainName: nodeTranslationChainName,
		nodeAccessNetwork:        nodeAccessNetwork,
		vpnInterface:             vpnInterface,
		log:                      log,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Watches(&corev1.Node{}, &handler.Funcs{
			CreateFunc: func(_ context.Context, _ event.TypedCreateEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				queue.Add(reconcile.Request{})
			},
			DeleteFunc: func(_ context.Context, _ event.TypedDeleteEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				queue.Add(reconcile.Request{})
			},
			GenericFunc: func(_ context.Context, _ event.TypedGenericEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				queue.Add(reconcile.Request{})
			},
			UpdateFunc: func(_ context.Context, e event.TypedUpdateEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				newNode, ok := e.ObjectNew.(*corev1.Node)
				if !ok {
					log.Warnf("Object from event was not a *corev1.Node. Instead got %T. Triggering a sync anyway", e.ObjectNew)
					queue.Add(reconcile.Request{})
				}
				oldNode, ok := e.ObjectOld.(*corev1.Node)
				if !ok {
					log.Warnf("Object from event was not a *corev1.Node. Instead got %T. Triggering a sync anyway", e.ObjectOld)
					queue.Add(reconcile.Request{})
				}

				// Only sync if nodes changed their addresses. Since Nodes get updated every 5 sec due to the HeartBeat
				// it would otherwise cause a lot of useless syncs
				if !equality.Semantic.DeepEqual(newNode.Status.Addresses, oldNode.Status.Addresses) {
					queue.Add(reconcile.Request{})
				}
			},
		}).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Add a wrapping here so we can emit an event on error
	err := r.syncDnatRules(ctx)
	if err != nil {
		r.log.Errorw("Failed reconciling", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) getDesiredRules(nodes []corev1.Node) []string {
	rules := []string{}
	for _, node := range nodes {
		nodeRules, err := r.getRulesForNode(node)
		if err != nil {
			r.log.Errorw("could not generate rules for node, skipping", "node", node.Name, zap.Error(err))
			continue
		}
		for _, rule := range nodeRules {
			rules = append(rules, rule.RestoreLine(r.nodeTranslationChainName))
		}
	}
	sort.Strings(rules)
	return rules
}

// syncDnatRules will recreate the complete set of translation rules
// based on the list of nodes.
func (r *Reconciler) syncDnatRules(ctx context.Context) error {
	// Get nodes from lister, make a copy.
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// Create the set of rules from all listed nodes.
	desiredRules := r.getDesiredRules(nodeList.Items)

	// Get the actual state (current iptable rules)
	allActualRules, err := execSave(ctx)
	if err != nil {
		return fmt.Errorf("failed to read iptable rules: %w", err)
	}
	// filter out everything that's not relevant for us
	actualRules, haveJump, haveMasquerade := r.filterDnatRules(allActualRules, r.nodeTranslationChainName)

	if !equality.Semantic.DeepEqual(actualRules, desiredRules) || !haveJump || !haveMasquerade {
		// Need to update chain in kernel.
		r.log.Infow("Updating iptables chain in kernel", "rules-count", len(desiredRules))
		if err := r.applyDNATRules(ctx, desiredRules, haveJump, haveMasquerade); err != nil {
			return fmt.Errorf("failed to apply iptable rules: %w", err)
		}
	}

	return nil
}

// getNodeAddresses returns all relevant addresses of a node.
// Only IPv4 addresses are returned as OpenVPN connection to the worker nodes is IPv4-only.
func getNodeAddresses(node corev1.Node) []string {
	addressTypes := []corev1.NodeAddressType{corev1.NodeExternalIP, corev1.NodeInternalIP}
	addresses := []string{}
	for _, addressType := range addressTypes {
		for _, address := range node.Status.Addresses {
			if address.Type == addressType && isIPv4(address.Address) {
				addresses = append(addresses, address.Address)
			}
		}
	}
	return addresses
}

func getInternalNodeAddress(node corev1.Node) (string, error) {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP && isIPv4(address.Address) {
			return address.Address, nil
		}
	}
	return "", fmt.Errorf("no internal address found; known addresses: %v", node.Status.Addresses)
}

func isIPv4(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() != nil
}

// getRulesForNode determines the used kubelet address of a node
// and creates a dnatRule from it.
func (r *Reconciler) getRulesForNode(node corev1.Node) ([]*dnatRule, error) {
	rules := []*dnatRule{}

	port := int(node.Status.DaemonEndpoints.KubeletEndpoint.Port)
	if port <= 0 {
		port = provider.DefaultKubeletPort
	}

	internalIP, err := getInternalNodeAddress(node)
	if err != nil {
		return rules, fmt.Errorf("failed to get internal node address: %w", err)
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
		l := len(r.nodeAccessNetwork)
		newAddress := fmt.Sprintf("%d.%d.%s.%s",
			r.nodeAccessNetwork[l-4], r.nodeAccessNetwork[l-3],
			octets[2], octets[3])
		rule.translatedAddress = newAddress
		rule.translatedPort = rule.originalTargetPort

		rules = append(rules, rule)
	}
	return rules, nil
}

// applyRules creates a iptables-save file and pipes it to stdin of
// a iptables-restore process for atomically setting new rules.
// This function replaces a complete chain (removing all pre-existing rules).
func (r *Reconciler) applyDNATRules(ctx context.Context, rules []string, haveJump, haveMasquerade bool) error {
	restore := []string{
		"*nat",
		fmt.Sprintf(":%s - [0:0]", r.nodeTranslationChainName)}

	if !haveJump {
		restore = append(restore,
			fmt.Sprintf("-I OUTPUT -j %s", r.nodeTranslationChainName))
	}

	if !haveMasquerade {
		restore = append(restore,
			fmt.Sprintf("-I POSTROUTING -o %s -j MASQUERADE", r.vpnInterface))
	}

	restore = append(restore, rules...)
	restore = append(restore, "COMMIT")

	return execRestore(ctx, restore)
}

func execSave(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "iptables-save", []string{"-t", "nat"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute %q: %w. Output: \n%s", strings.Join(cmd.Args, " "), err, out)
	}
	return strings.Split(string(out), "\n"), err
}

func execRestore(ctx context.Context, rules []string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "iptables-restore", []string{"--noflush", "-v", "-T", "nat"}...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	// if input is bigger than OS' pipe buffer size, the write call blocks indefinitely
	go func() {
		// ignore errors here, since it would likely lead to either the command to time out or to fail
		_, _ = io.WriteString(stdin, strings.Join(rules, "\n")+"\n")
		stdin.Close()
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("iptables-restore failed: %w (output: %s)", err, string(out))
		}
		return fmt.Errorf("iptables-restore failed: %w", err)
	}

	return nil
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
	target += ":"
	if len(rule.translatedPort) > 0 {
		target += rule.translatedPort
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
func (r *Reconciler) filterDnatRules(rules []string, chain string) ([]string, bool, bool) {
	out := []string{}
	haveJump := false
	haveMasquerade := false

	rulePrefix := fmt.Sprintf("-A %s ", chain)
	jumpPattern := fmt.Sprintf("-A OUTPUT -j %s", chain)
	masqPattern := fmt.Sprintf("-A POSTROUTING -o %s -j MASQUERADE", r.vpnInterface)
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
