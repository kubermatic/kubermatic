/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package csi

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/csi/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources/csi/nutanix"
	"k8c.io/kubermatic/v2/pkg/resources/csi/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/resources/csi/vsphere"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ResourcesForDeletion(cluster *kubermaticv1.Cluster) []ctrlruntimeclient.Object {
	resourceList := []ctrlruntimeclient.Object{}
	ns := cluster.Status.NamespaceName

	switch {
	case cluster.Spec.Cloud.VSphere != nil:
		resourceList = vsphere.ResourcesForDeletion(ns)
	case cluster.Spec.Cloud.VMwareCloudDirector != nil:
		resourceList = vmwareclouddirector.ResourcesForDeletion(ns)
	case cluster.Spec.Cloud.Nutanix != nil && cluster.Spec.Cloud.Nutanix.CSI != nil:
		resourceList = nutanix.ResourcesForDeletion(ns)
	case cluster.Spec.Cloud.Kubevirt != nil:
		resourceList = kubevirt.ResourcesForDeletion(ns)
	}
	return resourceList
}
