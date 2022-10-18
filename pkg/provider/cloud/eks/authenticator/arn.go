/*
Copyright 2017-2020 by the contributors.

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

package authenticator

import (
	"fmt"
	"strings"

	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
)

// extracted from boto/botocore: https://github.com/boto/botocore/blob/51c5b99680c7db7f10b5b70b222b6f7860bebdd4/botocore/data/endpoints.json
var partitions = []string{"aws", "aws-cn", "aws-us-gov", "aws-iso", "aws-iso-b"}

// Canonicalize validates IAM resources are appropriate for the authenticator
// and converts STS assumed roles into the IAM role resource.
//
// Supported IAM resources are:
//   - AWS account: arn:aws:iam::123456789012:root
//   - IAM user: arn:aws:iam::123456789012:user/Bob
//   - IAM role: arn:aws:iam::123456789012:role/S3Access
//   - IAM Assumed role: arn:aws:sts::123456789012:assumed-role/Accounting-Role/Mary (converted to IAM role)
//   - Federated user: arn:aws:sts::123456789012:federated-user/Bob
func CanonicalizeARN(arn string) (string, error) {
	parsed, err := awsarn.Parse(arn)
	if err != nil {
		return "", fmt.Errorf("arn '%s' is invalid: '%v'", arn, err)
	}

	if err := checkPartition(parsed.Partition); err != nil {
		return "", fmt.Errorf("arn '%s' does not have a recognized partition", arn)
	}

	parts := strings.Split(parsed.Resource, "/")
	resource := parts[0]

	switch parsed.Service {
	case "sts":
		switch resource {
		case "federated-user":
			return arn, nil
		case "assumed-role":
			if len(parts) < 3 {
				return "", fmt.Errorf("assumed-role arn '%s' does not have a role", arn)
			}
			// IAM ARNs can contain paths, part[0] is resource, parts[len(parts)] is the SessionName.
			role := strings.Join(parts[1:len(parts)-1], "/")
			return fmt.Sprintf("arn:%s:iam::%s:role/%s", parsed.Partition, parsed.AccountID, role), nil
		default:
			return "", fmt.Errorf("unrecognized resource %s for service sts", parsed.Resource)
		}
	case "iam":
		switch resource {
		case "role", "user", "root":
			return arn, nil
		default:
			return "", fmt.Errorf("unrecognized resource %s for service iam", parsed.Resource)
		}
	}

	return "", fmt.Errorf("service %s in arn %s is not a valid service for identities", parsed.Service, arn)
}

func checkPartition(partition string) error {
	for _, p := range partitions {
		if partition == p {
			return nil
		}
	}
	return fmt.Errorf("partition %s is not recognized", partition)
}
