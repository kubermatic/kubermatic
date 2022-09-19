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
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"k8s.io/utils/pointer"
)

func TestIsNotFound(t *testing.T) {
	ctx := context.Background()

	cs := getTestClientSet(ctx, t)
	errs := []error{}

	_, err := cs.EC2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{"does-not-exist"},
	})
	errs = append(errs, err)

	_, err = cs.EC2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{"does-not-exist"},
	})
	errs = append(errs, err)

	_, err = cs.EC2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{"does-not-exist"},
	})
	errs = append(errs, err)

	_, err = cs.IAM.GetRole(ctx, &iam.GetRoleInput{RoleName: pointer.String("does-not-exist")})
	errs = append(errs, err)

	_, err = cs.IAM.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{InstanceProfileName: pointer.String("does-not-exist")})
	errs = append(errs, err)

	for _, err := range errs {
		if !isNotFound(err) {
			t.Errorf("%v should have been recoginized as a notFound error.", err)
		}
	}
}
