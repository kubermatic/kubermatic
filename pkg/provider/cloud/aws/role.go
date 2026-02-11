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
	"path"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
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

func reconcileWorkerRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, accessKeyID, secretAccessKey, region string) (string, error) {
	policy, err := getWorkerPolicy(cluster.Name)
	if err != nil {
		return "", fmt.Errorf("failed to build the worker policy: %w", err)
	}

	// Get AWS account ID
	accountID, err := GetAccountID(ctx, accessKeyID, secretAccessKey, region)
	if err != nil {
		return "", fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	policies := map[string]string{workerPolicyName: policy}
	roleName := workerRoleName(cluster.Name)

	return ensureRole(ctx, client, cluster, roleName, policies, accountID)
}

// /////////////////////////
// control plane role

func controlPlaneRoleName(clusterName string) string {
	return fmt.Sprintf("%s%s-control-plane", resourceNamePrefix, clusterName)
}

func reconcileControlPlaneRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, accessKeyID, secretAccessKey, region string) (*kubermaticv1.Cluster, error) {
	policy, err := getControlPlanePolicy(cluster.Name)
	if err != nil {
		return cluster, fmt.Errorf("failed to build the control plane policy: %w", err)
	}

	// Get AWS account ID
	accountID, err := GetAccountID(ctx, accessKeyID, secretAccessKey, region)
	if err != nil {
		return cluster, fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	policies := map[string]string{controlPlanePolicyName: policy}

	// default the role name
	roleNameOrARN := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
	if roleNameOrARN == "" {
		roleNameOrARN = controlPlaneRoleName(cluster.Name)
	}

	// ensure role exists and is assigned to the given policies
	roleARN, err := ensureRole(ctx, client, cluster, roleNameOrARN, policies, accountID)
	if err != nil {
		return cluster, err
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.ControlPlaneRoleARN = roleARN
	})
}

func cleanUpControlPlaneRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster) error {
	// default the role name
	roleNameOrARN := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
	if roleNameOrARN == "" {
		roleNameOrARN = controlPlaneRoleName(cluster.Name)
	}

	return deleteRole(ctx, client, cluster, roleNameOrARN, []string{controlPlanePolicyName})
}

// /////////////////////////
// commonly shared functions

func getRole(ctx context.Context, client *iam.Client, roleNameOrARN string) (*iamtypes.Role, error) {
	roleName, err := decodeRoleARN(roleNameOrARN)
	if err != nil {
		return nil, err
	}

	getRoleInput := &iam.GetRoleInput{
		RoleName: ptr.To(roleName),
	}

	out, err := client.GetRole(ctx, getRoleInput)
	if err != nil {
		return nil, err
	}

	return out.Role, nil
}

func decodeRoleARN(nameOrARN string) (string, error) {
	if !arn.IsARN(nameOrARN) {
		return nameOrARN, nil
	}

	parsed, err := arn.Parse(nameOrARN)
	if err != nil {
		return "", fmt.Errorf("invalid role ARN %q: %w", nameOrARN, err)
	}

	// Roles resource names are stored as "role/...." in the ARN and we need
	// to strip the prefix.
	return path.Base(parsed.Resource), nil
}

func ensureRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, roleNameOrARN string, policies map[string]string, accountID string) (string, error) {
	// When ensuring an existing role, we are usually called with a full ARN,
	// but when creating a role initially, we are called with just the name.
	// To create/check the role, we first need to parse the nameOrARN.
	roleName, err := decodeRoleARN(roleNameOrARN)
	if err != nil {
		return "", err
	}

	// check if it still exists
	existingRole, err := getRole(ctx, client, roleName)
	if err != nil && !isNotFound(err) {
		return "", fmt.Errorf("failed to get role: %w", err)
	}

	// create missing role
	var roleARN string

	if isNotFound(err) {
		// Get templated assume role policy with account ID
		assumeRolePolicyDoc, err := getAssumeRolePolicy(accountID)
		if err != nil {
			return "", fmt.Errorf("failed to get assume role policy: %w", err)
		}

		createRoleInput := &iam.CreateRoleInput{
			AssumeRolePolicyDocument: ptr.To(assumeRolePolicyDoc),
			RoleName:                 ptr.To(roleName),
			Tags:                     []iamtypes.Tag{iamOwnershipTag(cluster.Name)},
		}

		output, err := client.CreateRole(ctx, createRoleInput)
		if err != nil {
			return "", fmt.Errorf("failed to create role: %w", err)
		}

		roleARN = *output.Role.Arn
	} else {
		roleARN = *existingRole.Arn
	}

	// attach policies
	for policyName, policyTpl := range policies {
		// The AWS API allows us to issue a PUT request, which has the create-or-update/upsert semantics
		putRolePolicyInput := &iam.PutRolePolicyInput{
			RoleName:       ptr.To(roleName),
			PolicyName:     ptr.To(policyName),
			PolicyDocument: ptr.To(policyTpl),
		}

		if _, err := client.PutRolePolicy(ctx, putRolePolicyInput); err != nil {
			return "", fmt.Errorf("failed to ensure policy %q for role %q: %w", policyName, roleNameOrARN, err)
		}
	}

	return roleARN, nil
}

func deleteRole(ctx context.Context, client *iam.Client, cluster *kubermaticv1.Cluster, roleNameOrARN string, policies []string) error {
	roleName, err := decodeRoleARN(roleNameOrARN)
	if err != nil {
		return err
	}

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
			RoleName: ptr.To(roleName),
		})
		if err != nil {
			return fmt.Errorf("failed to list policies for role %q: %w", roleNameOrARN, err)
		}

		policies = listPoliciesOut.PolicyNames
	}

	// delete policies from role
	for _, policyName := range policies {
		deleteRolePolicyInput := &iam.DeleteRolePolicyInput{
			PolicyName: ptr.To(policyName),
			RoleName:   ptr.To(roleName),
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
		RoleName: ptr.To(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to list attached policies for role %q: %w", roleNameOrARN, err)
	}

	for _, policy := range listAttachedPoliciesOut.AttachedPolicies {
		detachRolePolicyInput := &iam.DetachRolePolicyInput{
			RoleName:  ptr.To(roleName),
			PolicyArn: policy.PolicyArn,
		}
		if _, err := client.DetachRolePolicy(ctx, detachRolePolicyInput); err != nil {
			return fmt.Errorf("failed to detach policy %q: %w", *policy.PolicyName, err)
		}
	}

	// delete the role itself
	_, err = client.DeleteRole(ctx, &iam.DeleteRoleInput{RoleName: ptr.To(roleName)})

	return err
}
