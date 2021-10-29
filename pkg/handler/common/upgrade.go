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

package common

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver/v3"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetUpgradesEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, configGetter provider.KubermaticConfigurationGetter) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		// Happens during cluster creation when the CRD is not setup yet
		if _, ok := err.(*meta.NoKindMatchError); ok {
			return nil, nil
		}
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	providerName, err := resources.GetCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get the cloud provider name: %v", err)
	}
	var updateConditions []operatorv1alpha1.ConditionType
	externalCloudProvider := cluster.Spec.Features[v1.ClusterFeatureExternalCloudProvider]
	if externalCloudProvider {
		updateConditions = append(updateConditions, operatorv1alpha1.ExternalCloudProviderCondition)
	}

	config, err := configGetter(ctx)
	if err != nil {
		return nil, err
	}

	versionManager := version.NewFromConfiguration(config)

	versions, err := versionManager.GetPossibleUpdates(cluster.Spec.Version.String(), apiv1.KubernetesClusterType, kubermaticv1.ProviderType(providerName), updateConditions...)
	if err != nil {
		return nil, err
	}

	upgrades := make([]*apiv1.MasterVersion, 0)
	for _, v := range versions {
		isRestricted := false
		isRestricted, err = isRestrictedByKubeletVersions(v, machineDeployments.Items)
		if err != nil {
			return nil, err
		}
		upgrades = append(upgrades, &apiv1.MasterVersion{
			Version:                    v.Version,
			RestrictedByKubeletVersion: isRestricted,
		})
	}

	return upgrades, nil
}

func UpgradeNodeDeploymentsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, version apiv1.MasterVersion, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	requestedKubeletVersion, err := semver.NewVersion(version.Version.String())
	if err != nil {
		return nil, errors.NewBadRequest(err.Error())
	}

	if err = nodeupdate.EnsureVersionCompatible(cluster.Spec.Version.Semver(), requestedKubeletVersion); err != nil {
		return nil, errors.NewBadRequest(err.Error())
	}

	client, err := clusterProvider.GetAdminClientForCustomerCluster(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var updateErrors []string
	for _, machineDeployment := range machineDeployments.Items {
		machineDeployment.Spec.Template.Spec.Versions.Kubelet = version.Version.String()
		if err := client.Update(ctx, &machineDeployment); err != nil {
			updateErrors = append(updateErrors, err.Error())
		}
	}

	if len(updateErrors) > 0 {
		return nil, errors.NewWithDetails(http.StatusInternalServerError, "failed to update some node deployments", updateErrors)
	}

	return nil, nil
}

func isRestrictedByKubeletVersions(controlPlaneVersion *version.Version, mds []clusterv1alpha1.MachineDeployment) (bool, error) {
	for _, md := range mds {
		kubeletVersion, err := semver.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
		if err != nil {
			return false, err
		}

		if err = nodeupdate.EnsureVersionCompatible(controlPlaneVersion.Version, kubeletVersion); err != nil {
			return true, nil
		}
	}
	return false, nil
}
