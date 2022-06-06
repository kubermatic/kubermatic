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

package provider

import (
	"context"
	"fmt"
	"net/http"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	vcd "k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func VMwareCloudDirectorCatalogsWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {
	dc, creds, err := getVMwareCloudDirectorDataCenterAndCredentials(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return vcd.ListCatalogs(ctx, dc, creds.Username, creds.Password, creds.Organization, creds.VDC)
}

func VMwareCloudDirectorTemplatesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID, catalogName string) (interface{}, error) {
	dc, creds, err := getVMwareCloudDirectorDataCenterAndCredentials(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return vcd.ListTemplates(ctx, dc, creds.Username, creds.Password, creds.Organization, creds.VDC, catalogName)
}

func VMwareCloudDirectorNetworksWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {

	dc, creds, err := getVMwareCloudDirectorDataCenterAndCredentials(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return vcd.ListOVDCNetworks(ctx, dc, creds.Username, creds.Password, creds.Organization, creds.VDC)
}

func getClusterForVMwareCloudDirector(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, projectID string, clusterID string) (*kubermaticv1.Cluster, error) {
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.VMwareCloudDirector == nil {
		return nil, utilerrors.NewNotFound("cloud spec for ", clusterID)
	}
	return cluster, nil
}

func getVMwareCloudDirectorCredentials(ctx context.Context, cloudSpec kubermaticv1.CloudSpec) (*resources.VMwareCloudDirectorCredentials, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	credentials, err := vmwareclouddirector.GetCredentialsForCluster(cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func getVMwareCloudDirectorDataCenterAndCredentials(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, projectID, clusterID string) (*kubermaticv1.DatacenterSpecVMwareCloudDirector, *resources.VMwareCloudDirectorCredentials, error) {
	cluster, err := getClusterForVMwareCloudDirector(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, nil, err
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, nil, common.KubernetesErrorToHTTPError(err)
	}

	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting dc: %w", err)
	}

	if datacenter.Spec.VMwareCloudDirector == nil {
		return nil, nil, utilerrors.NewNotFound("cloud spec for ", clusterID)
	}

	creds, err := getVMwareCloudDirectorCredentials(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, nil, err
	}

	return datacenter.Spec.VMwareCloudDirector, creds, nil
}
