//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	"reflect"
	"slices"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/utils/ptr"
)

func TestKubeSystemRoleAllowsLeaderElectionEvents(t *testing.T) {
	t.Parallel()

	_, reconcile := KubeSystemRoleReconciler()()
	role, err := reconcile(&rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}
	for _, apiGroup := range []string{"", "events.k8s.io"} {
		for _, verb := range []string{"create", "patch"} {
			allowed := slices.ContainsFunc(role.Rules, func(rule rbacv1.PolicyRule) bool {
				return slices.Contains(rule.APIGroups, apiGroup) && slices.Contains(rule.Resources, "events") && slices.Contains(rule.Verbs, verb) && len(rule.ResourceNames) == 0
			})
			if !allowed {
				t.Errorf("leader election cannot %s events in API group %q", verb, apiGroup)
			}
		}
	}
}

// The v1.5 proxy and WAF controllers start independently of Gateway API and
// secret synchronization. Their informer and reconciliation permissions must
// therefore be present with every combination of those feature switches.
func TestClusterRoleControllerPermissions(t *testing.T) {
	t.Parallel()

	read := []string{"get", "list", "watch"}
	crud := []string{"create", "delete", "get", "list", "patch", "update", "watch"}
	status := []string{"get", "patch", "update"}
	permissions := []struct {
		apiGroup string
		resource string
		verbs    []string
	}{
		{"", "nodes", read},
		{"", "pods", read},
		{"", "serviceaccounts", []string{"create", "delete", "get", "list", "watch"}},
		{"rbac.authorization.k8s.io", "clusterroles", crud},
		{"rbac.authorization.k8s.io", "clusterrolebindings", crud},
		{"", "secrets", crud},
		{"", "configmaps", crud},
		{"apps", "daemonsets", crud},
		{"apps", "deployments", crud},
		{"networking.k8s.io", "networkpolicies", crud},
		{"", "services", crud},
		{"", "services/status", status},
		{"scheduling.k8s.io", "priorityclasses", read},
		{"", "events", nil}, // Legacy leader-election events only need the kube-system Role.
		{"events.k8s.io", "events", []string{"create", "patch"}},
		{"kubelb.k8c.io", "syncsecrets", crud},
		{"kubelb.k8c.io", "tenantwafpolicies", []string{"get", "list", "patch", "update", "watch"}},
		{"kubelb.k8c.io", "tenantwafpolicies/status", status},
		{"networking.k8s.io", "ingresses", crud},
		{"networking.k8s.io", "ingresses/status", status},
	}

	for _, tc := range []struct {
		name               string
		gatewayAPI         *bool
		secretSynchronizer bool
	}{
		{name: "defaults"},
		{name: "both disabled", gatewayAPI: ptr.To(false)},
		{name: "gateway only", gatewayAPI: ptr.To(true)},
		{name: "secret synchronizer only", gatewayAPI: ptr.To(false), secretSynchronizer: true},
		{name: "both enabled", gatewayAPI: ptr.To(true), secretSynchronizer: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dc := kubermaticv1.Datacenter{}
			cluster := &kubermaticv1.Cluster{}
			if tc.gatewayAPI != nil {
				dc.Spec.KubeLB = &kubermaticv1.KubeLBDatacenterSettings{EnableSecretSynchronizer: tc.secretSynchronizer}
				cluster.Spec.KubeLB = &kubermaticv1.KubeLB{EnableGatewayAPI: tc.gatewayAPI}
			}
			_, reconcile := ClusterRoleReconciler(dc, cluster)()
			role, err := reconcile(&rbacv1.ClusterRole{})
			if err != nil {
				t.Fatal(err)
			}
			for _, permission := range permissions {
				assertRoleVerbs(t, role, permission.apiGroup, permission.resource, permission.verbs)
			}

			var gatewayVerbs, secretFinalizerVerbs []string
			if ptr.Deref(tc.gatewayAPI, false) {
				gatewayVerbs = read
			}
			if tc.secretSynchronizer {
				secretFinalizerVerbs = status
			}
			for _, resource := range []string{"referencegrants", "gatewayclasses"} {
				assertRoleVerbs(t, role, "gateway.networking.k8s.io", resource, gatewayVerbs)
			}
			assertRoleVerbs(t, role, "", "secrets/finalizers", secretFinalizerVerbs)
		})
	}
}

func TestClusterRoleReconciliationAfterDisablingFeatures(t *testing.T) {
	t.Parallel()

	dc := kubermaticv1.Datacenter{}
	dc.Spec.KubeLB = &kubermaticv1.KubeLBDatacenterSettings{EnableSecretSynchronizer: true}
	cluster := &kubermaticv1.Cluster{}
	cluster.Spec.KubeLB = &kubermaticv1.KubeLB{EnableGatewayAPI: ptr.To(true)}
	_, reconcileEnabled := ClusterRoleReconciler(dc, cluster)()
	role, err := reconcileEnabled(&rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}

	dc.Spec.KubeLB.EnableSecretSynchronizer = false
	cluster.Spec.KubeLB.EnableGatewayAPI = ptr.To(false)
	_, reconcileDisabled := ClusterRoleReconciler(dc, cluster)()
	role, err = reconcileDisabled(role)
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []string{"referencegrants", "gatewayclasses", "gateways", "httproutes"} {
		assertRoleVerbs(t, role, "gateway.networking.k8s.io", resource, nil)
	}
	assertRoleVerbs(t, role, "", "secrets/finalizers", nil)
	assertRoleVerbs(t, role, "", "secrets", []string{"create", "delete", "get", "list", "patch", "update", "watch"})
	assertRoleVerbs(t, role, "kubelb.k8c.io", "tenantwafpolicies", []string{"get", "list", "patch", "update", "watch"})

	previous := role.DeepCopy()
	role, err = reconcileDisabled(role)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(previous, role) {
		t.Fatal("repeated reconciliation changed ClusterRole")
	}
}

func assertRoleVerbs(t *testing.T, role *rbacv1.ClusterRole, apiGroup, resource string, expected []string) {
	t.Helper()

	var actual []string
	for _, rule := range role.Rules {
		if slices.Contains(rule.APIGroups, apiGroup) && slices.Contains(rule.Resources, resource) {
			if len(rule.ResourceNames) != 0 {
				t.Errorf("permissions for %s/%s must not be limited to named resources", apiGroup, resource)
			}
			actual = append(actual, rule.Verbs...)
		}
	}
	slices.Sort(actual)
	actual = slices.Compact(actual)
	expected = slices.Clone(expected)
	slices.Sort(expected)
	if !slices.Equal(actual, expected) {
		t.Errorf("permissions for %s/%s: got %v, want %v", apiGroup, resource, actual, expected)
	}
}
