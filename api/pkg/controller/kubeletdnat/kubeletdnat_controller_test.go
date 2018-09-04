package kubeletdnat

import (
	"net"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRuleGeneration(t *testing.T) {
	nodeAccessNetwork, _, err := net.ParseCIDR("10.254.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Controller{
		client:                   nil,
		nodeLister:               nil,
		queue:                    nil,
		nodeTranslationChainName: "test-chain",
		nodeAccessNetwork:        nodeAccessNetwork,
	}

	nodes := []*corev1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "one"},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.1.1.11"},
					{Type: corev1.NodeExternalIP, Address: "192.0.2.101"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "two"},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.1.1.12"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "three"},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.1.1.13"},
					{Type: corev1.NodeExternalIP, Address: "192.0.2.103"},
				},
			},
		},
	}

	rules := ctrl.getDesiredRules(nodes)

	expectedRules := []string{
		"-A test-chain -d 10.1.1.11/32 -p tcp -m tcp --dport 10250 -j DNAT --to-destination 10.254.1.11:10250",
		"-A test-chain -d 10.1.1.12/32 -p tcp -m tcp --dport 10250 -j DNAT --to-destination 10.254.1.12:10250",
		"-A test-chain -d 10.1.1.13/32 -p tcp -m tcp --dport 10250 -j DNAT --to-destination 10.254.1.13:10250",
		"-A test-chain -d 192.0.2.101/32 -p tcp -m tcp --dport 10250 -j DNAT --to-destination 10.254.1.11:10250",
		"-A test-chain -d 192.0.2.103/32 -p tcp -m tcp --dport 10250 -j DNAT --to-destination 10.254.1.13:10250",
	}
	if len(rules) != len(expectedRules) {
		t.Errorf("expected exactly %d rules from test nodes. got %d rules.", len(expectedRules), len(rules))
		return
	}
	for i, expectedRule := range expectedRules {
		if rules[i] != expectedRule {
			t.Errorf("unexpected rule #%d. expeceted: %q", i, rules[i])
		}
	}
}
