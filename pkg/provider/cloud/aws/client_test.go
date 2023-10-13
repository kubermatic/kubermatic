//go:build integration

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
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

func TestIsNotFound(t *testing.T) {
	cs := getTestClientSet(t)
	errs := []error{}

	_, err := cs.EC2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice([]string{"does-not-exist"}),
	})
	errs = append(errs, err)

	_, err = cs.EC2.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		RouteTableIds: aws.StringSlice([]string{"does-not-exist"}),
	})
	errs = append(errs, err)

	_, err = cs.EC2.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: aws.StringSlice([]string{"does-not-exist"}),
	})
	errs = append(errs, err)

	_, err = cs.IAM.GetRole(&iam.GetRoleInput{RoleName: aws.String("does-not-exist")})
	errs = append(errs, err)

	_, err = cs.IAM.GetInstanceProfile(&iam.GetInstanceProfileInput{InstanceProfileName: aws.String("does-not-exist")})
	errs = append(errs, err)

	for _, err := range errs {
		if !isNotFound(err) {
			t.Errorf("%v should have been recognized as a notFound error.", err)
		}
	}
}
