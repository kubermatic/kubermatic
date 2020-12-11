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

package provider

import (
	"context"
	"fmt"
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vsphere"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func VsphereNetworksWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {

	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.VSphere == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	username, password, err := vsphere.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.VSphere)
	if err != nil {
		return nil, err
	}
	return GetVsphereNetworks(userInfo, seedsGetter, username, password, datacenterName)

}

func GetVsphereNetworks(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, datacenterName string) ([]apiv1.VSphereNetwork, error) {
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	networks, err := vsphere.GetNetworks(datacenter.Spec.VSphere, username, password)
	if err != nil {
		return nil, err
	}

	var apiNetworks []apiv1.VSphereNetwork
	for _, net := range networks {
		apiNetworks = append(apiNetworks, apiv1.VSphereNetwork{
			Name:         net.Name,
			Type:         net.Type,
			RelativePath: net.RelativePath,
			AbsolutePath: net.AbsolutePath,
		})
	}

	return apiNetworks, nil
}

func VsphereFoldersWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {

	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.VSphere == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	username, password, err := vsphere.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.VSphere)
	if err != nil {
		return nil, err
	}
	return GetVsphereFolders(userInfo, seedsGetter, username, password, datacenterName)

}

func GetVsphereFolders(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, datacenterName string) ([]apiv1.VSphereFolder, error) {
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	folders, err := vsphere.GetVMFolders(datacenter.Spec.VSphere, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %v", err)
	}

	var apiFolders []apiv1.VSphereFolder
	for _, folder := range folders {
		apiFolders = append(apiFolders, apiv1.VSphereFolder{Path: folder.Path})
	}

	return apiFolders, nil
}
