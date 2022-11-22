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

package csi

import (
	"context"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/csi/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources/csi/nutanix"
	"k8c.io/kubermatic/v2/pkg/resources/csi/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/resources/csi/vsphere"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

// SecretsCreators returns the function to create and update the secrets needed for CSI.
func SecretsCreators(ctx context.Context, data *resources.TemplateData) []reconciling.NamedSecretCreatorGetter {
	creatorGetters := []reconciling.NamedSecretCreatorGetter{}

	switch {
	case data.Cluster().Spec.Cloud.VSphere != nil:
		creatorGetters = vsphere.SecretsCreators(data)
	case data.Cluster().Spec.Cloud.VMwareCloudDirector != nil:
		creatorGetters = vmwareclouddirector.SecretsCreators(data)
	case data.Cluster().Spec.Cloud.Nutanix != nil && data.Cluster().Spec.Cloud.Nutanix.CSI != nil:
		creatorGetters = nutanix.SecretsCreators(data)
	case data.Cluster().Spec.Cloud.Kubevirt != nil:
		creatorGetters = kubevirt.SecretsCreators(ctx, data)
	}

	return creatorGetters
}
