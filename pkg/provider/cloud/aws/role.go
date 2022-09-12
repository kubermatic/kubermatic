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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/pointer"
)

const (
	workerPolicyName       = "kubernetes-worker"
	controlPlanePolicyName = "kubernetes-control-plane"
)

// /////////////////////////
// worker role (part of the worker instance profile)

func workerRoleName(clusterName string) string {
	return fmt.Sprintf("%s%s-worker", resourceNamePrefix, clusterName)
}

func reconcileWorkerRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster) error {
	policies := map[string]string{workerPolicyName: workerRolePolicy}
	roleName := workerRoleName(cluster.Name)

	return ensureRole(ctx, client, cluster, roleName, policies)
}

// /////////////////////////
// control plane role

func controlPlaneRoleName(clusterName string) string {
	return fmt.Sprintf("%s%s-control-plane", resourceNamePrefix, clusterName)
}

func reconcileControlPlaneRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	policy, err := getControlPlanePolicy(cluster.Name)
	if err != nil {
		return cluster, fmt.Errorf("failed to build the control plane policy: %w", err)
	}

	policies := map[string]string{controlPlanePolicyName: policy}

	// default the role name
	roleName := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
	if roleName == "" {
		roleName = controlPlaneRoleName(cluster.Name)
	}

	// ensure role exists and is assigned to the given policies
	if err := ensureRole(ctx, client, cluster, roleName, policies); err != nil {
		return cluster, err
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.ControlPlaneRoleARN = roleName
	})
}

func cleanUpControlPlaneRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster) error {
	// default the role name
	roleName := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
	if roleName == "" {
		roleName = controlPlaneRoleName(cluster.Name)
	}

	return deleteRole(ctx, client, cluster, roleName, []string{controlPlanePolicyName})
}

// /////////////////////////
// commonly shared functions

func getRole(ctx context.Context, client *iam.Client, roleName string) (*iamtypes.Role, error) {
	getRoleInput := &iam.GetRoleInput{
		RoleName: pointer.String(roleName),
	}

	out, err := client.GetRole(ctx, getRoleInput)
	if err != nil {
		return nil, err
	}

	return out.Role, nil
}

func ensureRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, roleName string, policies map[string]string) error {
	// check if it still exists
	_, err := getRole(ctx, client, roleName)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// create missing role
	if isNotFound(err) {
		createRoleInput := &iam.CreateRoleInput{
			AssumeRolePolicyDocument: pointer.String(assumeRolePolicy),
			RoleName:                 pointer.String(roleName),
			Tags:                     []iamtypes.Tag{iamOwnershipTag(cluster.Name)},
		}

		if _, err := client.CreateRole(ctx, createRoleInput); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}

	// attach policies
	for policyName, policyTpl := range policies {
		// The AWS API allows us to issue a PUT request, which has the create-or-update/upsert semantics
		putRolePolicyInput := &iam.PutRolePolicyInput{
			RoleName:       pointer.String(roleName),
			PolicyName:     pointer.String(policyName),
			PolicyDocument: pointer.String(policyTpl),
		}

		if _, err := client.PutRolePolicy(ctx, putRolePolicyInput); err != nil {
			return fmt.Errorf("failed to ensure policy %q for role %q: %w", policyName, roleName, err)
		}
	}

	return nil
}

func deleteRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, roleName string, policies []string) error {
	// check if it still exists
	role, err := getRole(ctx, client, roleName)
	if err != nil {
		// nothing more to do here
		if isNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get role: %w", err)
	}

	owned := hasIAMTag(iamOwnershipTag(cluster.Name), role.Tags)

	// delete policies; by default we only delete those that are specified, but when
	// we fully own the role, we must remove all policies, regardless of the policies
	// parameter
	if policies == nil || owned {
		// list all custom policies
		listPoliciesOut, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
			RoleName: pointer.String(roleName),
		})
		if err != nil {
			return fmt.Errorf("failed to list policies for role %q: %w", roleName, err)
		}

		policies = listPoliciesOut.PolicyNames
	}

	// delete policies from role
	for _, policyName := range policies {
		deleteRolePolicyInput := &iam.DeleteRolePolicyInput{
			PolicyName: pointer.String(policyName),
			RoleName:   pointer.String(roleName),
		}
		if _, err = client.DeleteRolePolicy(ctx, deleteRolePolicyInput); err != nil {
			return fmt.Errorf("failed to delete role policy %q: %w", policyName, err)
		}
	}

	// Deleting the cluster policy above always needs to happen,
	// but unless we actually own the role, we must stop here and not
	// detach AWS policies or even delete the role entirely.
	if !owned {
		return nil
	}

	// detach potential AWS managed policies
	listAttachedPoliciesOut, err := client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: pointer.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to list attached policies for role %q: %w", roleName, err)
	}

	for _, policy := range listAttachedPoliciesOut.AttachedPolicies {
		detachRolePolicyInput := &iam.DetachRolePolicyInput{
			RoleName:  pointer.String(roleName),
			PolicyArn: policy.PolicyArn,
		}
		if _, err := client.DetachRolePolicy(ctx, detachRolePolicyInput); err != nil {
			return fmt.Errorf("failed to detach policy %q: %w", *policy.PolicyName, err)
		}
	}

	// delete the role itself
	_, err = client.DeleteRole(ctx, &iam.DeleteRoleInput{RoleName: pointer.String(roleName)})

	return err
}
