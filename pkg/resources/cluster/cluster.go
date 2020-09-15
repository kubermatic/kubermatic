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

package cluster

import (
	"errors"
	"fmt"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/validation"
)

// Spec builds ClusterSpec kubermatic Custom Resource from API Cluster
func Spec(apiCluster apiv1.Cluster, dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*kubermaticv1.ClusterSpec, error) {
	spec := &kubermaticv1.ClusterSpec{
		HumanReadableName:                   apiCluster.Name,
		Cloud:                               apiCluster.Spec.Cloud,
		MachineNetworks:                     apiCluster.Spec.MachineNetworks,
		OIDC:                                apiCluster.Spec.OIDC,
		UpdateWindow:                        apiCluster.Spec.UpdateWindow,
		Version:                             apiCluster.Spec.Version,
		UsePodSecurityPolicyAdmissionPlugin: apiCluster.Spec.UsePodSecurityPolicyAdmissionPlugin,
		UsePodNodeSelectorAdmissionPlugin:   apiCluster.Spec.UsePodNodeSelectorAdmissionPlugin,
		AuditLogging:                        apiCluster.Spec.AuditLogging,
		Openshift:                           apiCluster.Spec.Openshift,
		AdmissionPlugins:                    apiCluster.Spec.AdmissionPlugins,
		OPAIntegration:                      apiCluster.Spec.OPAIntegration,
	}

	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("invalid cloud spec: %v", err)
	}
	if providerName == "" {
		return nil, errors.New("cluster has no cloudprovider")
	}
	cloudProvider, err := cloud.Provider(dc, secretKeyGetter)
	if err != nil {
		return nil, err
	}

	if err := defaulting.DefaultCreateClusterSpec(spec, cloudProvider); err != nil {
		return nil, err
	}

	return spec, validation.ValidateCreateClusterSpec(spec, dc, cloudProvider)
}
