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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/aws"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/azure"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/gcp"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/openstack"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/vsphere"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	// FakeVMWareUUIDKeyName is the name of the cloud-config configmap key
	// that holds the fake vmware uuid
	// It is required when activating the vsphere cloud-provider in the controller
	// manager on a non-ESXi host
	// Upstream issue: https://github.com/kubernetes/kubernetes/issues/65145
	FakeVMWareUUIDKeyName = "fakeVmwareUUID"
	FakeVMWareUUID        = "VMware-42 00 00 00 00 00 00 00-00 00 00 00 00 00 00 00"
)

type creatorData interface {
	DC() *kubermaticv1.Datacenter
	Cluster() *kubermaticv1.Cluster
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

// SecretReconciler returns a function to create the Secret containing the cloud-config.
func SecretReconciler(data creatorData, name string) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return name, func(cm *corev1.Secret) (*corev1.Secret, error) {
			if cm.Data == nil {
				cm.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			cloudConfig, err := createCloudConfig(data.Cluster(), data.DC(), credentials)
			if err != nil {
				return nil, fmt.Errorf("failed to create cloud-config: %w", err)
			}

			cm.Labels = resources.BaseAppLabels(resources.CloudConfigSeedSecretName, nil)
			cm.Data[resources.CloudConfigKey] = []byte(cloudConfig)
			cm.Data[FakeVMWareUUIDKeyName] = []byte(FakeVMWareUUID)

			return cm, nil
		}
	}
}

func KubeVirtInfraSecretReconciler(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
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

// createCloudConfig returns the cloud-config for the supplied data.
func createCloudConfig(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) (cloudConfig string, err error) {
	cloud := cluster.Spec.Cloud

	switch {
	case cloud.AWS != nil:
		cc := aws.ForCluster(cluster, dc)
		return cc.String()

	case cloud.Azure != nil:
		cc := azure.ForCluster(cluster, dc, credentials)
		return cc.String()

	case cloud.GCP != nil:
		cc, err := gcp.ForCluster(cluster, dc, credentials)
		if err != nil {
			return cloudConfig, err
		}

		return cc.String()

	case cloud.Kubevirt != nil:
		cc := kubevirt.ForCluster(cluster, dc)
		return cc.String()

	case cloud.Openstack != nil:
		cc := openstack.ForCluster(cluster, dc, credentials)
		return cc.String()

	case cloud.VMwareCloudDirector != nil:
		cc := vmwareclouddirector.ForCluster(cluster, dc, credentials)
		return cc.String()

	case cloud.VSphere != nil:
		cc, err := vsphere.ForCluster(cluster, dc, credentials)
		if err != nil {
			return cloudConfig, err
		}

		return cc.String()
	}

	return "", nil
}
