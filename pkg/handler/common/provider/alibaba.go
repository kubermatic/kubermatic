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
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func getAlibabaClient(accessKeyID, accessKeySecret, region string) (*ecs.Client, error) {
	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to create client: %v", err))
	}
	return client, err
}

func DescribeAlibabaInstanceTypes(accessKeyID, accessKeySecret, region, instanceType string) (*ecs.DescribeInstanceTypesResponse, error) {
	client, err := getAlibabaClient(accessKeyID, accessKeySecret, region)
	if err != nil {
		return nil, err
	}
	requestInstanceTypes := ecs.CreateDescribeInstanceTypesRequest()
	instanceTypes := []string{instanceType}
	requestInstanceTypes.InstanceTypes = &instanceTypes

	instTypes, err := client.DescribeInstanceTypes(requestInstanceTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance types, error: %w", err)
	}
	return instTypes, nil
}
