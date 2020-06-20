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

package machine_test

import (
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/machine"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

func TestCredentialEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name           string
		distribution   *apiv1.OperatingSystemSpec
		cloudProvider  *apiv1.NodeCloudSpec
		expectedResult string
	}{
		{
			name: "test SSH login name for AWS:Ubuntu",
			distribution: &apiv1.OperatingSystemSpec{
				Ubuntu: &apiv1.UbuntuSpec{DistUpgradeOnBoot: false},
			},

			cloudProvider: &apiv1.NodeCloudSpec{
				AWS: &apiv1.AWSNodeSpec{},
			},
			expectedResult: "ubuntu",
		},
		{
			name: "test SSH login name for VSphere:ContainerLinux",
			distribution: &apiv1.OperatingSystemSpec{
				ContainerLinux: &apiv1.ContainerLinuxSpec{},
			},

			cloudProvider: &apiv1.NodeCloudSpec{
				VSphere: &apiv1.VSphereNodeSpec{},
			},
			expectedResult: "core",
		},
		{
			name: "test SSH login name for Openstack:CentOS",
			distribution: &apiv1.OperatingSystemSpec{
				CentOS: &apiv1.CentOSSpec{},
			},

			cloudProvider: &apiv1.NodeCloudSpec{
				Openstack: &apiv1.OpenstackNodeSpec{},
			},
			expectedResult: "centos",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			resultLoginName, err := machine.GetSSHUserName(tc.distribution, tc.cloudProvider)
			if err != nil {
				t.Fatal(err)
			}
			if tc.expectedResult != resultLoginName {
				t.Fatalf("expected %s got %s", tc.expectedResult, resultLoginName)
			}

		})
	}
}
