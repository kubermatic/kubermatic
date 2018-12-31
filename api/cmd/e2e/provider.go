package main

import (
	"flag"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

var (
	awsClusterCloudSpec = kubermaticv1.CloudSpec{
		DatacenterName: "aws-eu-central-1a",
		AWS:            &kubermaticv1.AWSCloudSpec{},
	}

	awsNodeCloudSpec = kubermaticapiv1.NodeCloudSpec{
		AWS: &kubermaticapiv1.AWSNodeSpec{},
	}
)

func init() {
	// AWS
	flag.StringVar(&awsClusterCloudSpec.AWS.AccessKeyID, "aws-access-key-id", "", "AWS: AccessKeyID")
	flag.StringVar(&awsClusterCloudSpec.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS: SecretAccessKey")
	flag.StringVar(&awsClusterCloudSpec.DatacenterName, "aws-datacenter-name", "aws-eu-central-1a", "AWS: Datacenter name from the datacenter.yaml")
	flag.StringVar(&awsNodeCloudSpec.AWS.InstanceType, "aws-instance-type", "t2.medium", "AWS: Instance type")
	flag.StringVar(&awsNodeCloudSpec.AWS.VolumeType, "aws-volume-type", "gp2", "AWS: Instance volume type")
	flag.Int64Var(&awsNodeCloudSpec.AWS.VolumeSize, "aws-volume-size", 50, "AWS: Instance volume size")
}
