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

package data

import (
	"fmt"
	"strings"

	ec2 "github.com/cristim/ec2-instances-info"
	"k8c.io/kubermatic/v2/pkg/provider"
)

var data *ec2.InstanceData

// Due to big amount of data we are loading AWS instance types only once.
func init() {
	var err error

	data, err = ec2.Data()
	if err != nil {
		panic(fmt.Sprintf("failed to init EC2 data: %v", err))
	}
}

func GetInstanceSize(instanceType string) (*provider.NodeCapacity, error) {
	if data == nil {
		return nil, fmt.Errorf("AWS instance type data not initialized")
	}

	for _, i := range *data {
		if strings.EqualFold(i.InstanceType, instanceType) {
			cap := provider.NewNodeCapacity()
			cap.WithCPUCount(i.VCPU)
			cap.WithGPUCount(i.GPU)

			if err := cap.WithMemory(int(i.Memory), "G"); err != nil {
				return nil, fmt.Errorf("error parsing machine GPU quantity: %w", err)
			}

			return cap, nil
		}
	}

	return nil, fmt.Errorf("failed to find instance %q in aws instance type data", instanceType)
}
