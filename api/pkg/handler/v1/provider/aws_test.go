package provider_test

import (
	"reflect"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/provider"
)

func TestSetDefaultSubnet(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name               string
		machineDeployments *clusterv1alpha1.MachineDeploymentList
		subnets            apiv1.AWSSubnetList
		expectedResult     apiv1.AWSSubnetList
		expectedError      string
	}{
		{
			name: "test, no machines, set first as default",
			subnets: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
			machineDeployments: &clusterv1alpha1.MachineDeploymentList{
				Items: []clusterv1alpha1.MachineDeployment{},
			},
			expectedResult: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  true,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
		},
		{
			name: "test, no machines for eu-central-1a zone",
			subnets: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
			machineDeployments: &clusterv1alpha1.MachineDeploymentList{
				Items: []clusterv1alpha1.MachineDeployment{
					*test.GenTestMachineDeployment("md-1-b", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1b","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
					*test.GenTestMachineDeployment("md-1-c", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1c","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
				},
			},
			expectedResult: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  true,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
		},
		{
			name: "test, no machines for eu-central-1c zone",
			subnets: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
			machineDeployments: &clusterv1alpha1.MachineDeploymentList{
				Items: []clusterv1alpha1.MachineDeployment{
					*test.GenTestMachineDeployment("md-1-a", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
					*test.GenTestMachineDeployment("md-1-b", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1b","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
				},
			},
			expectedResult: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  true,
				},
			},
		},
		{
			name: "test, machines for all zones, set the first one",
			subnets: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
			machineDeployments: &clusterv1alpha1.MachineDeploymentList{
				Items: []clusterv1alpha1.MachineDeployment{
					*test.GenTestMachineDeployment("md-1-a", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
					*test.GenTestMachineDeployment("md-1-b", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1b","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
					*test.GenTestMachineDeployment("md-1-c", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1c","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
				},
			},
			expectedResult: apiv1.AWSSubnetList{
				{
					Name:             "subnet-a",
					AvailabilityZone: "eu-central-1a",
					IsDefaultSubnet:  true,
				},
				{
					Name:             "subnet-b",
					AvailabilityZone: "eu-central-1b",
					IsDefaultSubnet:  false,
				},
				{
					Name:             "subnet-c",
					AvailabilityZone: "eu-central-1c",
					IsDefaultSubnet:  false,
				},
			},
		},
		{
			name:    "test, subnet list empty",
			subnets: apiv1.AWSSubnetList{},
			machineDeployments: &clusterv1alpha1.MachineDeploymentList{
				Items: []clusterv1alpha1.MachineDeployment{
					*test.GenTestMachineDeployment("md-1-a", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
					*test.GenTestMachineDeployment("md-1-b", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1b","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
					*test.GenTestMachineDeployment("md-1-c", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1c","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
				},
			},
			expectedError: "the subnet list can not be empty",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			result, err := provider.SetDefaultSubnet(tc.machineDeployments, tc.subnets)
			if tc.expectedError != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if tc.expectedError != err.Error() {
					t.Fatalf("expected error: %v got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(tc.expectedResult, result) {
					t.Fatalf("expected: %v got %v", tc.expectedResult, result)
				}
			}
		})
	}
}
