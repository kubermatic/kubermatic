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
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var (
	fakeSuccessfulCreateInstanceProfile = func(input *iam.CreateInstanceProfileInput) (*iam.CreateInstanceProfileOutput, error) {
		return &iam.CreateInstanceProfileOutput{
			InstanceProfile: &iam.InstanceProfile{
				InstanceProfileName: input.InstanceProfileName,
				Path:                input.Path,
			},
		}, nil
	}
	fakeSuccessfulGetInstanceProfile = func(input *iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error) {
		return &iam.GetInstanceProfileOutput{
			InstanceProfile: &iam.InstanceProfile{
				InstanceProfileName: input.InstanceProfileName,
			},
		}, nil
	}
	fakeSuccessfulAddRoleToInstanceProfile = func(*iam.AddRoleToInstanceProfileInput) (*iam.AddRoleToInstanceProfileOutput, error) {
		return &iam.AddRoleToInstanceProfileOutput{}, nil
	}
	fakeSuccessfulCreateRole = func(input *iam.CreateRoleInput) (*iam.CreateRoleOutput, error) {
		return &iam.CreateRoleOutput{
			Role: &iam.Role{
				RoleName: input.RoleName,
			},
		}, nil
	}
	fakeSuccessfulGetRole = func(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
		return &iam.GetRoleOutput{
			Role: &iam.Role{
				RoleName: input.RoleName,
			},
		}, nil
	}
	fakeSuccessfulPutRolePolicy = func(input *iam.PutRolePolicyInput) (*iam.PutRolePolicyOutput, error) {
		return &iam.PutRolePolicyOutput{}, nil
	}
)

// fakeInstanceProfileClient is a fake client which allows overwriting certain functions.
// It defaults to successful responses
type fakeInstanceProfileClient struct {
	iamiface.IAMAPI
	createInstanceProfile    func(*iam.CreateInstanceProfileInput) (*iam.CreateInstanceProfileOutput, error)
	getInstanceProfile       func(*iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error)
	addRoleToInstanceProfile func(*iam.AddRoleToInstanceProfileInput) (*iam.AddRoleToInstanceProfileOutput, error)
	createRole               func(*iam.CreateRoleInput) (*iam.CreateRoleOutput, error)
	getRole                  func(*iam.GetRoleInput) (*iam.GetRoleOutput, error)
	putRolePolicy            func(*iam.PutRolePolicyInput) (*iam.PutRolePolicyOutput, error)
}

func (c *fakeInstanceProfileClient) CreateInstanceProfile(input *iam.CreateInstanceProfileInput) (*iam.CreateInstanceProfileOutput, error) {
	if c.createInstanceProfile != nil {
		return c.createInstanceProfile(input)
	}
	return fakeSuccessfulCreateInstanceProfile(input)
}

func (c *fakeInstanceProfileClient) GetInstanceProfile(input *iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error) {
	if c.getInstanceProfile != nil {
		return c.getInstanceProfile(input)
	}
	return fakeSuccessfulGetInstanceProfile(input)
}

func (c *fakeInstanceProfileClient) AddRoleToInstanceProfile(input *iam.AddRoleToInstanceProfileInput) (*iam.AddRoleToInstanceProfileOutput, error) {
	if c.addRoleToInstanceProfile != nil {
		return c.addRoleToInstanceProfile(input)
	}
	return fakeSuccessfulAddRoleToInstanceProfile(input)
}

func (c *fakeInstanceProfileClient) CreateRole(input *iam.CreateRoleInput) (*iam.CreateRoleOutput, error) {
	if c.createRole != nil {
		return c.createRole(input)
	}
	return fakeSuccessfulCreateRole(input)
}

func (c *fakeInstanceProfileClient) GetRole(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
	if c.getRole != nil {
		return c.getRole(input)
	}
	return fakeSuccessfulGetRole(input)
}

func (c *fakeInstanceProfileClient) PutRolePolicy(input *iam.PutRolePolicyInput) (*iam.PutRolePolicyOutput, error) {
	if c.putRolePolicy != nil {
		return c.putRolePolicy(input)
	}
	return fakeSuccessfulPutRolePolicy(input)
}

// func TestCreateWorkerInstanceProfile(t *testing.T) {
// 	tests := []struct {
// 		name                     string
// 		err                      error
// 		createInstanceProfile    func(*iam.CreateInstanceProfileInput) (*iam.CreateInstanceProfileOutput, error)
// 		getInstanceProfile       func(*iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error)
// 		addRoleToInstanceProfile func(*iam.AddRoleToInstanceProfileInput) (*iam.AddRoleToInstanceProfileOutput, error)
// 		createRole               func(*iam.CreateRoleInput) (*iam.CreateRoleOutput, error)
// 		getRole                  func(*iam.GetRoleInput) (*iam.GetRoleOutput, error)
// 		putRolePolicy            func(*iam.PutRolePolicyInput) (*iam.PutRolePolicyOutput, error)
// 	}{
// 		{
// 			name: "successfully created",
// 		},
// 		{
// 			name: "instance profile already exists",
// 			createInstanceProfile: func(input *iam.CreateInstanceProfileInput) (*iam.CreateInstanceProfileOutput, error) {
// 				return nil, awserr.New("EntityAlreadyExists", "test", errors.New("test"))
// 			},
// 		},
// 		{
// 			name: "create instance profile failed",
// 			err: errors.New(`failed to create instance profile: SomethingBadHappened: test
// caused by: test`),
// 			createInstanceProfile: func(input *iam.CreateInstanceProfileInput) (*iam.CreateInstanceProfileOutput, error) {
// 				return nil, awserr.New("SomethingBadHappened", "test", errors.New("test"))
// 			},
// 		},
// 		{
// 			name: "get instance profile failed",
// 			err: errors.New(`failed to create instance profile: failed to load the created instance profile "kubernetes-get instance profile failed": SomethingBadHappened: test
// caused by: test`),
// 			getInstanceProfile: func(input *iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error) {
// 				return nil, awserr.New("SomethingBadHappened", "test", errors.New("test"))
// 			},
// 		},
// 		{
// 			name: "role already exists",
// 			createRole: func(input *iam.CreateRoleInput) (*iam.CreateRoleOutput, error) {
// 				return nil, awserr.New("EntityAlreadyExists", "test", errors.New("test"))
// 			},
// 		},
// 		{
// 			name: "create role failed",
// 			err: errors.New(`failed to create worker role: SomethingBadHappened: test
// caused by: test`),
// 			createRole: func(input *iam.CreateRoleInput) (*iam.CreateRoleOutput, error) {
// 				return nil, awserr.New("SomethingBadHappened", "test", errors.New("test"))
// 			},
// 		},
// 		{
// 			name: "role was already attached",
// 			getInstanceProfile: func(input *iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error) {
// 				return &iam.GetInstanceProfileOutput{
// 					InstanceProfile: &iam.InstanceProfile{
// 						InstanceProfileName: input.InstanceProfileName,
// 						Roles: []*iam.Role{
// 							{
// 								// We use the test name as role name
// 								RoleName: aws.String(workerRoleName("role was already attached")),
// 							},
// 						},
// 					},
// 				}, nil
// 			},
// 			// Make the func return an error so we get an error when it gets called
// 			addRoleToInstanceProfile: func(input *iam.AddRoleToInstanceProfileInput) (*iam.AddRoleToInstanceProfileOutput, error) {
// 				return nil, awserr.New("SomethingBadHappened", "test", errors.New("test"))
// 			},
// 		},
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			client := &fakeInstanceProfileClient{
// 				createInstanceProfile:    test.createInstanceProfile,
// 				getInstanceProfile:       test.getInstanceProfile,
// 				createRole:               test.createRole,
// 				addRoleToInstanceProfile: test.addRoleToInstanceProfile,
// 				getRole:                  test.getRole,
// 				putRolePolicy:            test.putRolePolicy,
// 			}

// 			// We only test for the error as that's the only thing worth testing here.
// 			// Anything else would only test our mock implementation
// 			profile, err := createWorkerInstanceProfile(client, test.name)
// 			// String comparisons seem to be the simplest way of checking if errors are equal
// 			if fmt.Sprint(err) != fmt.Sprint(test.err) {
// 				t.Errorf("Got error \n%s\n Expected \n%s\n as error", err, test.err)
// 			}

// 			// We expect a valid instance profile if there was no error
// 			if err == nil && profile == nil {
// 				t.Error("returned instance profile is nil")
// 			}
// 		})
// 	}
// }
