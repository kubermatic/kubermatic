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

package alibaba

import (
	"context"
	"errors"
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type Alibaba struct {
	dc                *kubermaticv1.DatacenterSpecAlibaba
	secretKeySelector provider.SecretKeySelectorValueFunc
}

var _ provider.CloudProvider = &Alibaba{}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Alibaba, error) {
	if dc.Spec.Alibaba == nil {
		return nil, errors.New("datacenter is not an Alibaba datacenter")
	}
	return &Alibaba{
		dc:                dc.Spec.Alibaba,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func (a *Alibaba) DefaultCloudSpec(_ context.Context, _ *kubermaticv1.ClusterSpec) error {
	return nil
}

func (a *Alibaba) ValidateCloudSpec(_ context.Context, spec kubermaticv1.CloudSpec) error {
	accessKeyID, accessKeySecret, err := GetCredentialsForCluster(spec, a.secretKeySelector, a.dc)
	if err != nil {
		return err
	}

	_, err = ecs.NewClientWithAccessKey(a.dc.Region, accessKeyID, accessKeySecret)
	if err != nil {
		return fmt.Errorf("failed to get Alibaba cloud client: %w", err)
	}
	return nil
}

func (a *Alibaba) InitializeCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (a *Alibaba) CleanUpCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (a *Alibaba) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, dc *kubermaticv1.DatacenterSpecAlibaba) (accessKeyID string, accessKeySecret string, err error) {
	accessKeyID = cloud.Alibaba.AccessKeyID
	accessKeySecret = cloud.Alibaba.AccessKeySecret

	if accessKeyID == "" {
		if cloud.Alibaba.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided, accessKeyID cannot be empty")
		}
		accessKeyID, err = secretKeySelector(cloud.Alibaba.CredentialsReference, resources.AlibabaAccessKeyID)
		if err != nil {
			return "", "", err
		}
	}

	if accessKeySecret == "" {
		if cloud.Alibaba.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided, accessKeySecret cannot be empty")
		}
		accessKeySecret, err = secretKeySelector(cloud.Alibaba.CredentialsReference, resources.AlibabaAccessKeySecret)
		if err != nil {
			return "", "", err
		}
	}

	return accessKeyID, accessKeySecret, nil
}

func ValidateCredentials(region, accessKeyID, accessKeySecret string) error {
	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return err
	}

	requestZones := ecs.CreateDescribeZonesRequest()
	requestZones.Scheme = "https"

	_, err = client.DescribeZones(requestZones)
	return err
}

func DescribeInstanceType(accessKeyID, accessKeySecret, region, instanceType string) (*provider.NodeCapacity, error) {
	client, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	requestInstanceTypes := ecs.CreateDescribeInstanceTypesRequest()
	filter := []string{instanceType}
	requestInstanceTypes.InstanceTypes = &filter

	instanceTypes, err := client.DescribeInstanceTypes(requestInstanceTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance types: %w", err)
	}

	if len(instanceTypes.InstanceTypes.InstanceType) == 0 {
		return nil, fmt.Errorf("unknown instance type %q", instanceType)
	}

	instance := instanceTypes.InstanceTypes.InstanceType[0]

	capacity := provider.NewNodeCapacity()
	capacity.WithCPUCount(instance.CpuCoreCount)

	if err := capacity.WithMemory(int(instance.MemorySize), "G"); err != nil {
		return nil, fmt.Errorf("failed to parse memory size: %w", err)
	}

	return capacity, nil
}
