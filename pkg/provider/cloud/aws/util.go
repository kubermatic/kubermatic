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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

func hasEC2Tag(expected *ec2.Tag, actual []*ec2.Tag) bool {
	for _, tag := range actual {
		if tag.String() == expected.String() {
			return true
		}
	}

	return false
}

func hasIAMTag(expected *iam.Tag, actual []*iam.Tag) bool {
	for _, tag := range actual {
		if tag.String() == expected.String() {
			return true
		}
	}

	return false
}
