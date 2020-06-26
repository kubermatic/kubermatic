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
	"errors"
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/resources"
)

type Alibaba struct {
	dc                *kubermaticv1.DatacenterSpecAlibaba
	secretKeySelector provider.SecretKeySelectorValueFunc
}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Alibaba, error) {
	if dc.Spec.Alibaba == nil {
		return nil, errors.New("datacenter is not an Alibaba datacenter")
	}
	return &Alibaba{
		dc:                dc.Spec.Alibaba,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func (a *Alibaba) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (a *Alibaba) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	accessKeyID, accessKeySecret, err := GetCredentialsForCluster(spec, a.secretKeySelector, a.dc)
	if err != nil {
		return err
	}

	_, err = ecs.NewClientWithAccessKey(a.dc.Region, accessKeyID, accessKeySecret)
	if err != nil {
		return fmt.Errorf("failed to get Alibaba cloud client: %v", err)
	}
	return nil
}

func (a *Alibaba) InitializeCloudProvider(c *kubermaticv1.Cluster, p provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return c, nil
}

func (a *Alibaba) CleanUpCloudProvider(c *kubermaticv1.Cluster, p provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return c, nil
}

func (a *Alibaba) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
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
