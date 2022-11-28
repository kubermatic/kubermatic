/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package provider

import (
	"testing"

	aws "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

func TestAWSConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewAWSConfig().
		WithRegion("region").
		WithAvailabilityZone("az").
		WithVpcID("vpcID").
		WithSubnetID("subnet").
		WithAMI("ami").
		WithInstanceProfile("profileName").
		WithInstanceType("itype").
		WithDiskType("diskType").
		WithDiskSize(10).
		WithDiskIops(10).
		WithAssignPublicIP(true).
		WithAssignPublicIP(false).
		WithSpotInstanceMaxPrice("0.5").
		WithSpotInstanceMaxPrice("").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.InstanceType.Value != "itype" {
		t.Fatal("Builder did not apply instance type to the config.")
	}
}

type awsTestcase struct {
	baseTestcase[aws.RawConfig, kubermaticv1.DatacenterSpecAWS]

	os providerconfig.OperatingSystem
}

func (tt *awsTestcase) Run(cluster *kubermaticv1.Cluster) (*aws.RawConfig, error) {
	return CompleteAWSProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[aws.RawConfig] = &awsTestcase{}

func TestCompleteAWSProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecAWS{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteAWSProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.AWS = &kubermaticv1.AWSCloudSpec{}
		if _, err := CompleteAWSProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching AWS, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.AWSCloudProvider),
		AWS: &kubermaticv1.AWSCloudSpec{
			VPCID:               "vpcID",
			RouteTableID:        "rtID",
			SecurityGroupID:     "sgID",
			InstanceProfileName: "ipn",
		},
	})

	defaultMachine := NewAWSConfig().
		WithDiskType(awsDefaultDiskType).
		WithDiskSize(awsDefaultDiskSize).
		WithEBSVolumeEncrypted(awsDefaultEBSVolumeEncrypted).
		WithTag("system/cluster", goodCluster.Name).
		WithTag("system/project", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey]).
		WithTag("kubernetes.io/cluster/"+goodCluster.Name, "")

	// good machine is the base machine, but with values from the cluster already applied
	goodMachine := cloneBuilder(defaultMachine).
		WithVpcID(goodCluster.Spec.Cloud.AWS.VPCID).
		WithSecurityGroupID(goodCluster.Spec.Cloud.AWS.SecurityGroupID).
		WithInstanceProfile(goodCluster.Spec.Cloud.AWS.InstanceProfileName)

	testcases := []testcase[aws.RawConfig]{
		&awsTestcase{
			baseTestcase: baseTestcase[aws.RawConfig, kubermaticv1.DatacenterSpecAWS]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecAWS{
					Region: "testregion-aws",
				},
				expected: cloneBuilder(goodMachine).
					WithRegion("testregion-aws").
					WithAvailabilityZone("testregion-awsa"),
			},
		},
		&awsTestcase{
			baseTestcase: baseTestcase[aws.RawConfig, kubermaticv1.DatacenterSpecAWS]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecAWS{
					Region: "testregion-aws",
				},
				inputSpec: cloneBuilder(defaultMachine).WithRegion("keep-me-aws"),
				expected:  cloneBuilder(goodMachine).WithRegion("keep-me-aws").WithAvailabilityZone("keep-me-awsa"),
			},
		},
		&awsTestcase{
			baseTestcase: baseTestcase[aws.RawConfig, kubermaticv1.DatacenterSpecAWS]{
				name: "should select the correct AMI based on the OS",
				datacenter: &kubermaticv1.DatacenterSpecAWS{
					Images: kubermaticv1.ImageList{
						providerconfig.OperatingSystemFlatcar: "testimage",
					},
				},
				expected: cloneBuilder(goodMachine).WithAMI("testimage"),
			},
			os: providerconfig.OperatingSystemFlatcar,
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
