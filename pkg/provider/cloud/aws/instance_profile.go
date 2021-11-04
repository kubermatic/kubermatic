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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func workerInstanceProfileName(clusterName string) string {
	return resourceNamePrefix + clusterName
}

func getInstanceProfile(client iamiface.IAMAPI, name string) (*iam.InstanceProfile, error) {
	getProfileInput := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(name),
	}

	profileOut, err := client.GetInstanceProfile(getProfileInput)
	if err != nil {
		return nil, err
	}

	return profileOut.InstanceProfile, nil
}

func reconcileWorkerInstanceProfile(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// Even though the profile depends upon the role (the role is assigned to it),
	// the decision whether or not to reconcile any role depends on whether KKP
	// owns the profile. If a user-supplied profile is used, then no role will
	// be created by KKP at all.

	profileName := cluster.Spec.Cloud.AWS.InstanceProfileName
	if profileName == "" {
		profileName = workerInstanceProfileName(cluster.Name)
	}

	profile, err := ensureInstanceProfile(client, cluster, profileName)
	if err != nil {
		return cluster, fmt.Errorf("failed to ensure instance profile %q: %w", profileName, err)
	}

	// if we own the profile, we must also take care of the worker role
	if hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
		// ensure the role exists
		if err := reconcileWorkerRole(client, cluster); err != nil {
			return nil, fmt.Errorf("failed to reconcile worker role: %w", err)
		}

		// and assign it to this profile
		roleName := workerRoleName(cluster.Name)
		exists := false

		for _, profileRole := range profile.Roles {
			if *profileRole.RoleName == roleName {
				exists = true
				break
			}
		}

		if !exists {
			addRoleInput := &iam.AddRoleToInstanceProfileInput{
				InstanceProfileName: aws.String(profileName),
				RoleName:            aws.String(roleName),
			}

			if _, err = client.AddRoleToInstanceProfile(addRoleInput); err != nil {
				return cluster, fmt.Errorf("failed to add role to the instance profile: %w", err)
			}
		}
	}

	return update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.InstanceProfileName = profileName
	})
}

func ensureInstanceProfile(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster, profileName string) (*iam.InstanceProfile, error) {
	// check if it exists
	profile, err := getInstanceProfile(client, profileName)
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("failed to get instance profile %q: %w", profileName, err)
	}

	// found it
	if err == nil {
		return profile, nil
	}

	// create missing profile
	createProfileInput := &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		Tags:                []*iam.Tag{iamOwnershipTag(cluster.Name)},
	}

	output, err := client.CreateInstanceProfile(createProfileInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance profile: %w", err)
	}

	return output.InstanceProfile, nil
}

func cleanUpWorkerInstanceProfile(client iamiface.IAMAPI, cluster *kubermaticv1.Cluster) error {
	profileName := cluster.Spec.Cloud.AWS.InstanceProfileName
	if profileName == "" {
		profileName = workerInstanceProfileName(cluster.Name)
	}

	// check if the profile still exists
	profile, err := getInstanceProfile(client, profileName)
	if err != nil {
		// the profile is already gone
		if isNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get instance profile: %w", err)
	}

	// we only clean up if we actually own the profile
	if !hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
		return nil
	}

	// before deleting an instance profile we must first remove all roles from it
	for _, role := range profile.Roles {
		removeRoleInput := &iam.RemoveRoleFromInstanceProfileInput{
			RoleName:            role.RoleName,
			InstanceProfileName: aws.String(profileName),
		}
		if _, err = client.RemoveRoleFromInstanceProfile(removeRoleInput); err != nil {
			return fmt.Errorf("failed to remove role %q from instance profile %q: %w", *role.RoleName, profileName, err)
		}
	}

	// delete the worker-role we created
	if err := deleteRole(client, cluster, workerRoleName(cluster.Name), nil); err != nil {
		return fmt.Errorf("failed to delete worker role: %w", err)
	}

	// delete the profile itself
	_, err = client.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{InstanceProfileName: aws.String(profileName)})

	return err
}
