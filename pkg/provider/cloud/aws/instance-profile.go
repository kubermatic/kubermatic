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
)

func instanceProfileName(clusterName string) string {
	return resourceNamePrefix + clusterName
}

func createInstanceProfile(client iamiface.IAMAPI, profileName string) (*iam.InstanceProfile, error) {
	createProfileInput := &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	}

	_, err := client.CreateInstanceProfile(createProfileInput)
	// If the role already exists, we consider it as success.
	if err != nil && !isEntityAlreadyExists(err) {
		return nil, err
	}

	// We fetch the role here to compensate for any "already exists" error.
	// That way we can ensure that this function is idempotent.
	getProfileInput := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	}
	profileOutput, err := client.GetInstanceProfile(getProfileInput)
	if err != nil {
		return nil, fmt.Errorf("failed to load the created instance profile %q: %v", profileName, err)
	}

	return profileOutput.InstanceProfile, nil
}

func createWorkerInstanceProfile(client iamiface.IAMAPI, clusterName string) (*iam.InstanceProfile, error) {
	workerRole, err := createWorkerRole(client, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker role: %v", err)
	}

	workerInstanceProfile, err := createInstanceProfile(client, instanceProfileName(clusterName))
	if err != nil {
		return nil, fmt.Errorf("failed to create instance profile: %v", err)
	}

	// We need to attach the workerRole to the workerInstanceProfile.
	// If it is already attached we're done.
	for _, profileRole := range workerInstanceProfile.Roles {
		if *profileRole.RoleName == *workerRole.RoleName {
			return workerInstanceProfile, nil
		}
	}

	addRoleInput := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: workerInstanceProfile.InstanceProfileName,
		RoleName:            workerRole.RoleName,
	}
	if _, err = client.AddRoleToInstanceProfile(addRoleInput); err != nil {
		return nil, fmt.Errorf("failed to add the worker role to the worker instance profile: %v", err)
	}

	return workerInstanceProfile, nil
}

func deleteInstanceProfile(client iamiface.IAMAPI, profileName string) error {
	getProfileInput := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	}
	profileOutput, err := client.GetInstanceProfile(getProfileInput)
	if err != nil {
		// If the profile is already gone: Success!
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get instance profile %q: %v", profileName, err)
	}
	profile := profileOutput.InstanceProfile

	// Before deleting an instance profile we must first remove all roles from it
	for _, role := range profile.Roles {
		removeRoleInput := &iam.RemoveRoleFromInstanceProfileInput{
			RoleName:            role.RoleName,
			InstanceProfileName: aws.String(profileName),
		}
		if _, err = client.RemoveRoleFromInstanceProfile(removeRoleInput); err != nil {
			return fmt.Errorf("failed to remove role %q from instance profile %q: %v", *role.RoleName, profileName, err)
		}
	}

	deleteProfileInput := &iam.DeleteInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	}
	_, err = client.DeleteInstanceProfile(deleteProfileInput)
	return err
}
