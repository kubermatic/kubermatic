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

package externalcluster

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func GetUpgradesEndpoint(configGetter provider.KubermaticConfigurationGetter, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(GetClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCluster := convertClusterToAPIWithStatus(ctx, clusterProvider, privilegedClusterProvider, cluster)
		upgrades := make([]*apiv1.MasterVersion, 0)
		cloud := cluster.Spec.CloudSpec
		if cloud.ProviderName == "" {
			return upgrades, nil
		}
		if apiCluster.Status.State != apiv2.RUNNING {
			return upgrades, nil
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

		if cloud.GKE != nil {
			sa, err := secretKeySelector(cloud.GKE.CredentialsReference, resources.GCPServiceAccount)
			if err != nil {
				return nil, err
			}
			return gke.ListGKEUpgrades(ctx, sa, cloud.GKE.Zone, cloud.GKE.Name)
		}
		if cloud.AKS != nil {
			cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
			if err != nil {
				return nil, err
			}
			return providercommon.ListAKSUpgrades(ctx, cred, cloud.AKS.ResourceGroup, cloud.AKS.Name)
		}
		if cloud.KubeOne != nil {
			version, err := clusterProvider.GetVersion(ctx, cluster)
			if err != nil {
				return nil, err
			}
			return handlercommon.GetKubeOneUpgradesEndpoint(ctx, cluster, version, configGetter)
		}

		return nil, nil
	}
}

func GetMachineDeploymentUpgradesEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(machineDeploymentReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCluster := convertClusterToAPIWithStatus(ctx, clusterProvider, privilegedClusterProvider, cluster)
		upgrades := make([]*apiv1.MasterVersion, 0)
		cloud := cluster.Spec.CloudSpec
		if cloud.ProviderName == "" {
			return upgrades, nil
		}
		if apiCluster.Status.State != apiv2.RUNNING {
			return upgrades, nil
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

		if cloud.GKE != nil {
			sa, err := secretKeySelector(cloud.GKE.CredentialsReference, resources.GCPServiceAccount)
			if err != nil {
				return nil, err
			}
			return gke.ListGKEMachineDeploymentUpgrades(ctx,
				sa,
				cloud.GKE.Zone,
				cloud.GKE.Name,
				req.MachineDeploymentID)
		}
		if cloud.AKS != nil {
			cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
			if err != nil {
				return nil, err
			}
			return aks.ListAKSMachineDeploymentUpgrades(ctx,
				cred,
				cloud.AKS.Name,
				cloud.AKS.ResourceGroup,
				req.MachineDeploymentID)
		}
		if cloud.EKS != nil {
			accessKeyID, secretAccessKey, err := eks.GetCredentialsForCluster(*cloud, secretKeySelector)
			if err != nil {
				return nil, err
			}
			return eks.ListMachineDeploymentUpgrades(ctx,
				accessKeyID,
				secretAccessKey,
				cloud.EKS.Region,
				cloud.EKS.Name,
				req.MachineDeploymentID)
		}

		return nil, nil
	}
}
