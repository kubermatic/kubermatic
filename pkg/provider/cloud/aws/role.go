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

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
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

func reconcileWorkerRole(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster) error {
	policies := map[string]string{workerPolicyName: workerRolePolicy}
	roleName := workerRoleName(cluster.Name)

	return ensureRole(client, cluster, roleName, policies)
}

// func deleteWorkerRole(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster) error {
// 	return deleteRole(client, workerRoleName(cluster.Name))
// }

// /////////////////////////
// control plane role

func controlPlaneRoleName(clusterName string) string {
	return fmt.Sprintf("%s%s-control-plane", resourceNamePrefix, clusterName)
}

func reconcileControlPlaneRole(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	policy, err := getControlPlanePolicy(cluster.Name)
	if err != nil {
		return cluster, fmt.Errorf("failed to build the control plane policy: %v", err)
	}

	policies := map[string]string{controlPlanePolicyName: policy}

	// default the role name
	roleName := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
	if roleName == "" {
		roleName = controlPlaneRoleName(cluster.Name)
	}

	// ensure role exists and is assigned to the given policies
	if err := ensureRole(client, cluster, roleName, policies); err != nil {
		return cluster, err
	}

	return update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(cluster, controlPlaneRoleCleanupFinalizer)
		cluster.Spec.Cloud.AWS.ControlPlaneRoleARN = roleName
	})
}

func cleanUpControlPlaneRole(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster) error {
	// default the role name
	roleName := cluster.Spec.Cloud.AWS.ControlPlaneRoleARN
	if roleName == "" {
		roleName = controlPlaneRoleName(cluster.Name)
	}

	return deleteRole(client, cluster, roleName, []string{controlPlanePolicyName})
}

// /////////////////////////
// commonly shared functions

func ensureRole(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster, roleName string, policies map[string]string) error {
	// check if it still exists
	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	_, err := client.GetRole(getRoleInput)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// create missing role
	if isNotFound(err) {
		createRoleInput := &iam.CreateRoleInput{
			AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
			RoleName:                 aws.String(roleName),
			Tags:                     []*iam.Tag{iamOwnershipTag(cluster.Name)},
		}

		if _, err := client.CreateRole(createRoleInput); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}

	// attach policies
	for policyName, policyTpl := range policies {
		// The AWS API allows us to issue a PUT request, which has the create-or-update/upsert semantics
		putRolePolicyInput := &iam.PutRolePolicyInput{
			RoleName:       aws.String(roleName),
			PolicyName:     aws.String(policyName),
			PolicyDocument: aws.String(policyTpl),
		}

		if _, err := client.PutRolePolicy(putRolePolicyInput); err != nil {
			return fmt.Errorf("failed to ensure policy %q for role %q: %w", policyName, roleName, err)
		}
	}

	return nil
}

func deleteRole(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster, roleName string, policies []string) error {
	// check if it still exists
	output, err := client.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		// nothing more to do here
		if isNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get role: %w", err)
	}

	// delete policies
	if policies == nil {
		// list all custom policies
		listPoliciesOut, err := client.ListRolePolicies(&iam.ListRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			return fmt.Errorf("failed to list policies for role %q: %w", roleName, err)
		}

		policies := []string{}

		for _, policyName := range listPoliciesOut.PolicyNames {
			policies = append(policies, *policyName)
		}
	}

	// delete policies from role
	for _, policyName := range policies {
		deleteRolePolicyInput := &iam.DeleteRolePolicyInput{
			PolicyName: aws.String(policyName),
			RoleName:   aws.String(roleName),
		}
		if _, err = client.DeleteRolePolicy(deleteRolePolicyInput); err != nil {
			return fmt.Errorf("failed to delete role policy %q: %w", policyName, err)
		}
	}

	// Deleting the cluster policy above always needs to happen,
	// but unless we actually own the role, we must stop here and not
	// detach AWS policies or even delete the role entirely.
	if !hasIAMTag(iamOwnershipTag(cluster.Name), output.Role.Tags) {
		return nil
	}

	// detach potential AWS managed policies
	listAttachedPoliciesOut, err := client.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to list attached policies for role %q: %w", roleName, err)
	}

	for _, policy := range listAttachedPoliciesOut.AttachedPolicies {
		detachRolePolicyInput := &iam.DetachRolePolicyInput{
			RoleName:  aws.String(roleName),
			PolicyArn: policy.PolicyArn,
		}
		if _, err := client.DetachRolePolicy(detachRolePolicyInput); err != nil {
			return fmt.Errorf("failed to detach policy %q: %w", *policy.PolicyName, err)
		}
	}

	// delete the role itself
	_, err = client.DeleteRole(&iam.DeleteRoleInput{RoleName: aws.String(roleName)})

	return err
}
