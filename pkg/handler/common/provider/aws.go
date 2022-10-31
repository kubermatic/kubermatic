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

package provider

import (
	"fmt"
	"strings"

	ec2 "github.com/cristim/ec2-instances-info"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

var data *ec2.InstanceData

// Due to big amount of data we are loading AWS instance types only once.
func init() {
	data, _ = ec2.Data()
}

func GetAWSInstance(instanceType string) (*apiv1.AWSSize, error) {
	if data == nil {
		return nil, fmt.Errorf("AWS instance type data not initialized")
	}

	for _, i := range *data {
		if strings.EqualFold(i.InstanceType, instanceType) {
			return &apiv1.AWSSize{
				Memory: i.Memory,
				VCPUs:  i.VCPU,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to find instance %q in aws instance type data", instanceType)
}
