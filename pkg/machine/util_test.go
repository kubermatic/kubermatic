package machine_test

import (
	"testing"

	"github.com/kubermatic/kubermatic/pkg/machine"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
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
