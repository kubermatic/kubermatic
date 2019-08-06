package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

const (
	workerPolicyName       = "kubernetes-worker"
	controlPlanePolicyName = "kubernetes-control-plane"
)

func workerRoleName(clusterName string) string {
	return fmt.Sprintf("%s%s-worker", resourceNamePrefix, clusterName)
}

func createWorkerRole(client iamiface.IAMAPI, clusterName string) (*iam.Role, error) {
	policies := map[string]string{workerPolicyName: workerRolePolicy}
	return createRole(client, workerRoleName(clusterName), assumeRolePolicy, policies)
}

func controlPlaneRoleName(clusterName string) string {
	return fmt.Sprintf("%s%s-control-plane", resourceNamePrefix, clusterName)
}

func createControlPlaneRole(client iamiface.IAMAPI, clusterName string) (*iam.Role, error) {
	policy, err := getControlPlanePolicy(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to build the control plane policy: %v", err)
	}
	policies := map[string]string{controlPlanePolicyName: policy}
	return createRole(client, controlPlaneRoleName(clusterName), assumeRolePolicy, policies)
}

func createRole(client iamiface.IAMAPI, roleName, assumeRolePolicy string, rolePolicies map[string]string) (*iam.Role, error) {
	createRoleInput := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
		RoleName:                 aws.String(roleName),
	}

	_, err := client.CreateRole(createRoleInput)
	// If the role already exists, we consider it as success.
	if err != nil && !isEntityAlreadyExists(err) {
		return nil, err
	}

	// We fetch the role here so we keep the logic here simple.
	// Sure it causes an additional API call, but the simpler code is preferable over 1 saved API call (the client retries those anyway).
	getRoleInput := &iam.GetRoleInput{RoleName: aws.String(roleName)}
	roleOut, err := client.GetRole(getRoleInput)
	if err != nil {
		return nil, fmt.Errorf("failed to load the created role %q: %v", roleName, err)
	}
	role := roleOut.Role

	for policyName, policyTpl := range rolePolicies {
		// The AWS API allows us to issue a PUT request, which has the create-or-update/upsert semantics
		putRolePolicyInput := &iam.PutRolePolicyInput{
			RoleName:       role.RoleName,
			PolicyName:     aws.String(policyName),
			PolicyDocument: aws.String(policyTpl),
		}
		if _, err := client.PutRolePolicy(putRolePolicyInput); err != nil {
			return nil, fmt.Errorf("failed to create/update the policy %q for role %q: %v", policyName, *role.RoleName, err)
		}
	}

	return role, nil
}

func deleteRole(client iamiface.IAMAPI, roleName string) error {
	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}
	if _, err := client.GetRole(getRoleInput); err != nil {
		// If the profile is already gone: Success!
		if isNoSuchEntity(err) {
			return nil
		}
		return fmt.Errorf("failed to get role %q: %v", roleName, err)
	}

	listPolicyOut, err := client.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to list policies for role %q: %v", roleName, err)
	}

	for _, policyName := range listPolicyOut.PolicyNames {
		deleteRolePolicyInput := &iam.DeleteRolePolicyInput{
			PolicyName: policyName,
			RoleName:   aws.String(roleName),
		}
		if _, err = client.DeleteRolePolicy(deleteRolePolicyInput); err != nil {
			return fmt.Errorf("failed to delete role policy %q: %v", *policyName, err)
		}
	}

	deleteRoleInput := &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	}
	_, err = client.DeleteRole(deleteRoleInput)
	return err
}
