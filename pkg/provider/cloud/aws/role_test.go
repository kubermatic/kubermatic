//go:build integration

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

func assertRoleHasPolicy(ctx context.Context, t *testing.T, client *iam.Client, roleName, policyName, policyDocument string) {
	output, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
		RoleName:   pointer.String(roleName),
		PolicyName: pointer.String(policyName),
	})
	if err != nil {
		t.Errorf("failed to retrieve role policy: %v", err)
		return
	}

	if output.PolicyDocument == nil || *output.PolicyDocument != policyDocument {
		t.Error("role should have policy attached, but does not")
	}
}

func assertOwnership(ctx context.Context, t *testing.T, client *iam.Client, cluster *kubermaticv1.Cluster, roleName string, expectedOwnership bool) {
	// check if the role exists
	getRoleInput := &iam.GetRoleInput{
		RoleName: pointer.String(roleName),
	}

	role, err := client.GetRole(ctx, getRoleInput)
	if err != nil {
		t.Errorf("failed to retrieve role: %v", err)
		return
	}

	if hasIAMTag(iamOwnershipTag(cluster.Name), role.Role.Tags) != expectedOwnership {
		if expectedOwnership {
			t.Error("Role should have ownership tag, but does not.")
		} else {
			t.Error("Role should not have ownership tag, but does.")
		}
	}
}

func assertRoleIsGone(t *testing.T, client *iam.Client, roleName string) {
	if _, err := getRole(context.Background(), client, roleName); err == nil {
		t.Fatal("GetRole did not return an error, indicating that the role still exists.")
	}
}

func assertRolePolicies(ctx context.Context, t *testing.T, client *iam.Client, roleName string, expected sets.String) {
	listPoliciesOut, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: pointer.String(roleName),
	})
	if err != nil {
		t.Errorf("Failed to list policies for role %q: %v", roleName, err)
		return
	}

	current := sets.NewString()

	for _, policyName := range listPoliciesOut.PolicyNames {
		current.Insert(policyName)
	}

	if !current.Equal(expected) {
		t.Fatalf("Expected role to have %v policies, but it has %v", expected, current)
	}
}

func TestEnsureRole(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	t.Run("role-does-not-exist-yet", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
		roleName := controlPlaneRoleName(cluster.Name)

		policy, err := getControlPlanePolicy(cluster.Name)
		if err != nil {
			t.Fatalf("failed to build the worker policy: %v", err)
		}

		policies := map[string]string{controlPlanePolicyName: policy}

		if err := ensureRole(context.Background(), cs.IAM, cluster, roleName, policies); err != nil {
			t.Fatalf("ensureRole should have not errored, but returned %v", err)
		}

		assertOwnership(ctx, t, cs.IAM, cluster, roleName, true)
		assertRoleHasPolicy(ctx, t, cs.IAM, roleName, controlPlanePolicyName, policy)
	})

	t.Run("add-policy-to-foreign-existing-role", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
		roleName := controlPlaneRoleName(cluster.Name)

		policy, err := getControlPlanePolicy(cluster.Name)
		if err != nil {
			t.Fatalf("failed to build the worker policy: %v", err)
		}

		policies := map[string]string{controlPlanePolicyName: policy}

		// create a role that the controller is then hopefully going to adopt
		createRoleInput := &iam.CreateRoleInput{
			AssumeRolePolicyDocument: pointer.String(assumeRolePolicy),
			RoleName:                 pointer.String(roleName),
		}

		if _, err := cs.IAM.CreateRole(ctx, createRoleInput); err != nil {
			t.Fatalf("failed to create test role: %vv", err)
		}

		// reconcile role and check if the code successfully attaches the policy
		// to an existing role
		if err := ensureRole(ctx, cs.IAM, cluster, roleName, policies); err != nil {
			t.Fatalf("ensureRole should have not errored, but returned %v", err)
		}

		assertOwnership(ctx, t, cs.IAM, cluster, roleName, false) // role was pre-existing, so we should not add an owner tag
		assertRoleHasPolicy(ctx, t, cs.IAM, roleName, controlPlanePolicyName, policy)
	})
}

func TestReconcileWorkerRole(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
	roleName := workerRoleName(cluster.Name)

	if err := reconcileWorkerRole(ctx, cs.IAM, cluster); err != nil {
		t.Fatalf("reconcileWorkerRole should have not errored, but returned %v", err)
	}

	policy, err := getWorkerPolicy(cluster.Name)
	if err != nil {
		t.Fatalf("failed to build the worker policy: %v", err)
	}

	assertOwnership(ctx, t, cs.IAM, cluster, roleName, true)
	assertRoleHasPolicy(ctx, t, cs.IAM, roleName, workerPolicyName, policy)
}

func TestReconcileControlPlaneRole(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	t.Run("no-role-yet", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
		updater := testClusterUpdater(cluster)

		policy, err := getControlPlanePolicy(cluster.Name)
		if err != nil {
			t.Fatalf("failed to build the control plane policy: %v", err)
		}

		cluster, err = reconcileControlPlaneRole(ctx, cs.IAM, cluster, updater)
		if err != nil {
			t.Fatalf("reconcileControlPlaneRole should have not errored, but returned %v", err)
		}

		expectedRole := controlPlaneRoleName(cluster.Name)
		if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN != expectedRole {
			t.Errorf("cloud spec should have been updated to include role name %q, but is now %q", expectedRole, cluster.Spec.Cloud.AWS.ControlPlaneRoleARN)
		}

		assertOwnership(ctx, t, cs.IAM, cluster, expectedRole, true)
		assertRoleHasPolicy(ctx, t, cs.IAM, expectedRole, controlPlanePolicyName, policy)
	})

	t.Run("role-set-but-does-not-exist", func(t *testing.T) {
		roleName := "does-not-exist-yet-" + rand.String(10)
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			ControlPlaneRoleARN: roleName,
		})
		updater := testClusterUpdater(cluster)

		cluster, err := reconcileControlPlaneRole(context.Background(), cs.IAM, cluster, updater)
		if err != nil {
			t.Fatalf("reconcileControlPlaneRole should have not errored, but returned %v", err)
		}

		// the code should keep the user-supplied name and not replace it with the default name
		if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN != roleName {
			t.Errorf("cloud spec should have been updated to include role name %q, but is now %q", roleName, cluster.Spec.Cloud.AWS.ControlPlaneRoleARN)
		}

		assertOwnership(ctx, t, cs.IAM, cluster, roleName, true)
	})
}

func TestDeleteRole(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	t.Run("fully-owned-role", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
		updater := testClusterUpdater(cluster)

		// reconcile to create the control plane role
		cluster, err := reconcileControlPlaneRole(ctx, cs.IAM, cluster, updater)
		if err != nil {
			t.Fatalf("reconcileControlPlaneRole should have not errored, but returned %v", err)
		}

		// ensure the role exists now
		expectedRole := controlPlaneRoleName(cluster.Name)
		if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN != expectedRole {
			t.Errorf("cloud spec should have been updated to include role name %q, but is now %q", expectedRole, cluster.Spec.Cloud.AWS.ControlPlaneRoleARN)
		}

		assertOwnership(ctx, t, cs.IAM, cluster, expectedRole, true)

		// and let's nuke it (we do not specify a list of policies, so the code should
		// be smart enough to figure out that it needs to remove all policies)
		if err := deleteRole(ctx, cs.IAM, cluster, expectedRole, nil); err != nil {
			t.Fatalf("deleteRole should not have errored, but returned %v", err)
		}

		assertRoleIsGone(t, cs.IAM, expectedRole)
	})

	t.Run("foreign-owned-role", func(t *testing.T) {
		ctx := context.Background()

		roleName := "my-role-" + rand.String(10)
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{ControlPlaneRoleARN: roleName})
		updater := testClusterUpdater(cluster)

		// create a foreign role
		createRoleInput := &iam.CreateRoleInput{
			AssumeRolePolicyDocument: pointer.String(assumeRolePolicy),
			RoleName:                 pointer.String(roleName),
		}

		if _, err := cs.IAM.CreateRole(ctx, createRoleInput); err != nil {
			t.Fatalf("failed to create test role: %vv", err)
		}

		// reconcile the role to assign policies to it, but not the owner tag
		cluster, err := reconcileControlPlaneRole(ctx, cs.IAM, cluster, updater)
		if err != nil {
			t.Fatalf("reconcileControlPlaneRole should have not errored, but returned %v", err)
		}

		// ensure no owner tag
		assertOwnership(ctx, t, cs.IAM, cluster, roleName, false)

		// and let's "nuke" it; normally we would always specify a list of policies here,
		// but for this test case we just try to see what the code does it no list was given;
		// it should remove all policies from the role.
		if err := deleteRole(ctx, cs.IAM, cluster, roleName, nil); err != nil {
			t.Fatalf("deleteRole should not have errored, but returned %v", err)
		}

		// check if the role exists
		getRoleInput := &iam.GetRoleInput{
			RoleName: pointer.String(roleName),
		}

		if _, err := cs.IAM.GetRole(ctx, getRoleInput); err != nil {
			t.Fatalf("GetRole did return an error, indicating that the role was removed when it should not have been: %v", err)
		}

		// assert that our policy (in fact, for this test case, _all_ policies) was removed from the role
		assertRolePolicies(ctx, t, cs.IAM, roleName, sets.NewString())
	})
}

// TestCleanUpControlPlaneRole is very similar to TestDeleteRole, but nonetheless we test it.
func TestCleanUpControlPlaneRole(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	t.Run("fully-owned-role", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
		updater := testClusterUpdater(cluster)

		// reconcile to create the control plane role
		cluster, err := reconcileControlPlaneRole(context.Background(), cs.IAM, cluster, updater)
		if err != nil {
			t.Fatalf("reconcileControlPlaneRole should have not errored, but returned %v", err)
		}

		roleName := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
		assertOwnership(ctx, t, cs.IAM, cluster, roleName, true)

		// and let's nuke it
		if err := cleanUpControlPlaneRole(context.Background(), cs.IAM, cluster); err != nil {
			t.Fatalf("deleteRole should not have errored, but returned %v", err)
		}

		assertRoleIsGone(t, cs.IAM, roleName)
	})

	t.Run("foreign-owned-role", func(t *testing.T) {
		roleName := "my-role-" + rand.String(10)
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{ControlPlaneRoleARN: roleName})
		updater := testClusterUpdater(cluster)

		// create a foreign role
		createRoleInput := &iam.CreateRoleInput{
			AssumeRolePolicyDocument: pointer.String(assumeRolePolicy),
			RoleName:                 pointer.String(roleName),
		}

		if _, err := cs.IAM.CreateRole(ctx, createRoleInput); err != nil {
			t.Fatalf("failed to create test role: %vv", err)
		}

		// reconcile the role to assign policies to it, but not the owner tag
		cluster, err := reconcileControlPlaneRole(context.Background(), cs.IAM, cluster, updater)
		if err != nil {
			t.Fatalf("reconcileControlPlaneRole should have not errored, but returned %v", err)
		}

		// ensure no owner tag
		assertOwnership(ctx, t, cs.IAM, cluster, roleName, false)

		// and let's "nuke" it
		if err := cleanUpControlPlaneRole(context.Background(), cs.IAM, cluster); err != nil {
			t.Fatalf("deleteRole should not have errored, but returned %v", err)
		}

		// check if the role exists
		getRoleInput := &iam.GetRoleInput{
			RoleName: pointer.String(roleName),
		}

		if _, err := cs.IAM.GetRole(ctx, getRoleInput); err != nil {
			t.Fatalf("GetRole did return an error, indicating that the role was removed when it should not have been: %v", err)
		}

		// assert that our policy (in fact, for this test case, _all_ policies) was removed from the role
		assertRolePolicies(ctx, t, cs.IAM, roleName, sets.NewString())
	})
}
