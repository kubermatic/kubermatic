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
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	"k8s.io/utils/pointer"
)

func hasEC2Tag(expected ec2types.Tag, actual []ec2types.Tag) bool {
	for _, tag := range actual {
		if pointer.StringDeref(tag.Key, "") == pointer.StringDeref(expected.Key, "") &&
			pointer.StringDeref(tag.Value, "") == pointer.StringDeref(expected.Value, "") {
			return true
		}
	}

	return false
}

func hasIAMTag(expected iamtypes.Tag, actual []iamtypes.Tag) bool {
	for _, tag := range actual {
		if pointer.StringDeref(tag.Key, "") == pointer.StringDeref(expected.Key, "") &&
			pointer.StringDeref(tag.Value, "") == pointer.StringDeref(expected.Value, "") {
			return true
		}
	}

	return false
}
