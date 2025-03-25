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

package vmwareclouddirector

import (
	"context"
	"errors"
	"fmt"

	"github.com/vmware/go-vcloud-director/v2/govcd"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	ResourceNamePattern    = "kubernetes-%s"
	vappDescriptionPattern = "Kubernetes cluster %s"
)

func reconcileVApp(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, vdc *govcd.Vdc) (*kubermaticv1.Cluster, error) {
	var err error
	// Ensure that finalizer exists
	if !kuberneteshelper.HasFinalizer(cluster, vappFinalizer) {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, vappFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	// Check if the vApp already exists
	vAppName := cluster.Spec.Cloud.VMwareCloudDirector.VApp
	if vAppName == "" {
		vAppName = fmt.Sprintf(ResourceNamePattern, cluster.Name)
	}

	_, err = vdc.GetVAppByNameOrId(vAppName, true)
	if err != nil && !errors.Is(err, govcd.ErrorEntityNotFound) {
		return nil, fmt.Errorf("failed to get vApp '%s': %w", vAppName, err)
	}

	// We need to create the vApp
	if err != nil && errors.Is(err, govcd.ErrorEntityNotFound) {
		_, err = vdc.CreateRawVApp(vAppName, fmt.Sprintf(vappDescriptionPattern, cluster.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to create vApp '%s': %w", vAppName, err)
		}
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.VMwareCloudDirector.VApp = vAppName
		})
	}
	return cluster, nil
}

func reconcileNetwork(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, vdc *govcd.Vdc) (*kubermaticv1.Cluster, error) {
	var err error

	// Ensure that all the ovdc networks are attached to the vApp
	ovdcNetworks, err := getOrgVDCNetworks(vdc, *cluster.Spec.Cloud.VMwareCloudDirector)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization VDC networks: %w", err)
	}

	for _, ovdcNetwork := range ovdcNetworks {
		if err := reconcileNetworkForVApp(ctx, cluster, update, vdc, ovdcNetwork); err != nil {
			return nil, fmt.Errorf("failed to reconcile network for vApp: %w", err)
		}
	}

	return cluster, nil
}

func reconcileNetworkForVApp(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, vdc *govcd.Vdc, network *govcd.OrgVDCNetwork) error {
	// Check if the network is already present in vApp.
	vApp, err := vdc.GetVAppByNameOrId(cluster.Spec.Cloud.VMwareCloudDirector.VApp, true)
	if err != nil {
		return err
	}

	exists := false
	if vApp.VApp.NetworkConfigSection != nil && vApp.VApp.NetworkConfigSection.NetworkConfig != nil {
		for _, net := range vApp.VApp.NetworkConfigSection.NetworkNames() {
			if net == network.OrgVDCNetwork.Name {
				exists = true
			}
		}
	}

	// We need to attach the network to vApp
	if !exists {
		if _, err := vApp.AddOrgNetwork(&govcd.VappNetworkSettings{}, network.OrgVDCNetwork, false); err != nil {
			return fmt.Errorf("failed to attach organization VDC network '%s' to vApp '%s': %w", network.OrgVDCNetwork.Name, cluster.Spec.Cloud.VMwareCloudDirector.VApp, err)
		}
	}

	return nil
}
