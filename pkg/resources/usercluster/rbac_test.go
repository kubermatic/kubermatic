/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package usercluster

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestClusterRoleAllowsPolicyBindingStatusPatches(t *testing.T) {
	t.Parallel()

	_, reconciler := ClusterRole()()
	role, err := reconciler(&rbacv1.ClusterRole{})
	if err != nil {
		t.Fatalf("failed to reconcile ClusterRole: %v", err)
	}

	if !hasRule(role.Rules, "kubermatic.k8c.io", "policybindings/status", "patch") {
		t.Fatal("ClusterRole does not allow patching PolicyBinding status")
	}
}

func hasRule(rules []rbacv1.PolicyRule, apiGroup, resource, verb string) bool {
	for _, rule := range rules {
		if contains(rule.APIGroups, apiGroup) && contains(rule.Resources, resource) && (contains(rule.Verbs, verb) || contains(rule.Verbs, "*")) {
			return true
		}
	}

	return false
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}

	return false
}
