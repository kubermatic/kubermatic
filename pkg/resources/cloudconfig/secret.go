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

package cloudconfig

import (
	"errors"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	vmwareclouddirectorcloudconfig "k8c.io/kubermatic/v2/pkg/resources/cloudconfig/vmwareclouddirector"
	vspherecloudconfig "k8c.io/kubermatic/v2/pkg/resources/cloudconfig/vsphere"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	// FakeVMWareUUIDKeyName is the name of the cloud-config configmap key
	// that holds the fake vmware uuid
	// It is required when activating the vsphere cloud-provider in the controller
	// manager on a non-ESXi host
	// Upstream issue: https://github.com/kubernetes/kubernetes/issues/65145
	FakeVMWareUUIDKeyName = "fakeVmwareUUID"
	fakeVMWareUUID        = "VMware-42 00 00 00 00 00 00 00-00 00 00 00 00 00 00 00"
)

type creatorData interface {
	DC() *kubermaticv1.Datacenter
	Cluster() *kubermaticv1.Cluster
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

// SecretCreator returns a function to create the Secret containing the cloud-config.
func SecretCreator(data creatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CloudConfigSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			if cm.Data == nil {
				cm.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			cloudConfig, err := CloudConfig(data.Cluster(), data.DC(), credentials)
			if err != nil {
				return nil, fmt.Errorf("failed to create cloud-config: %w", err)
			}

			cm.Labels = resources.BaseAppLabels(resources.CloudConfigSeedSecretName, nil)
			cm.Data[resources.CloudConfigKey] = []byte(cloudConfig)
			cm.Data[FakeVMWareUUIDKeyName] = []byte(fakeVMWareUUID)

			return cm, nil
		}
	}
}

func VsphereCSISecretCreator(data creatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CSICloudConfigSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			if cm.Data == nil {
				cm.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			vsphereCloudConfig, err := getVSphereCloudConfig(data.Cluster(), data.DC(), credentials)
			if err != nil {
				return nil, err
			}

			cloudConfig, err := vspherecloudconfig.CloudConfigCSIToString(vsphereCloudConfig)
			if err != nil {
				return nil, err
			}

			cm.Labels = resources.BaseAppLabels(resources.CSICloudConfigSecretName, nil)
			cm.Data[resources.CloudConfigKey] = []byte(cloudConfig)
			cm.Data[FakeVMWareUUIDKeyName] = []byte(fakeVMWareUUID)

			return cm, nil
		}
	}
}

func VMwareCloudDirectorCSISecretCreator(data creatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CSICloudConfigSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			if cm.Data == nil {
				cm.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			vcdCloudConfig, err := vmwareclouddirectorcloudconfig.GetVMwareCloudDirectorCSIConfig(data.Cluster(), data.DC(), credentials)
			if err != nil {
				return nil, err
			}

			cm.Labels = resources.BaseAppLabels(resources.CSICloudConfigSecretName, nil)
			cm.Data[resources.CloudConfigKey] = []byte(vcdCloudConfig)

			return cm, nil
		}
	}
}

func NutanixCSISecretCreator(data creatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CSICloudConfigSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			if cm.Data == nil {
				cm.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			if data.Cluster().Spec.Cloud.Nutanix.CSI.Port == nil {
				return nil, errors.New("CSI Port must not be nil")
			}

			nutanixCsiConf := fmt.Sprintf("%s:%d:%s:%s", data.Cluster().Spec.Cloud.Nutanix.CSI.Endpoint, *data.Cluster().Spec.Cloud.Nutanix.CSI.Port, credentials.Nutanix.CSIUsername, credentials.Nutanix.CSIPassword)

			cm.Labels = resources.BaseAppLabels(resources.CSICloudConfigSecretName, nil)
			cm.Data[resources.CloudConfigKey] = []byte(nutanixCsiConf)

			return cm, nil
		}
	}
}

func KubeVirtInfraSecretCreator(data *resources.TemplateData) reconciling.NamedSecretCreatorGetter {
	return func() (name string, create reconciling.SecretCreator) {
		return resources.KubeVirtInfraSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}
			se.Data[resources.KubeVirtInfraSecretKey] = []byte(credentials.Kubevirt.KubeConfig)
			return se, nil
		}
	}
}
