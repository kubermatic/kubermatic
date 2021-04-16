/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"github.com/anexia-it/go-anxcloud/pkg/client"
	"github.com/anexia-it/go-anxcloud/pkg/vlan"
	"github.com/anexia-it/go-anxcloud/pkg/vsphere/provisioning/templates"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/anexia"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func ListAnexiaVlans(ctx context.Context, token string) (apiv1.AnexiaVlanList, error) {
	response := apiv1.AnexiaVlanList{}

	cli, err := getClient(token)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}
	v := vlan.NewAPI(cli)
	vlans, err := v.List(ctx, 1, 1000)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}

	for _, vlan := range vlans {
		apiVlan := apiv1.AnexiaVlan{
			ID: vlan.Identifier,
		}
		response = append(response, apiVlan)
	}

	return response, nil
}

func ListAnexiaTemplates(ctx context.Context, token, locationID string) (apiv1.AnexiaTemplateList, error) {
	response := apiv1.AnexiaTemplateList{}

	cli, err := getClient(token)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}
	t := templates.NewAPI(cli)
	templates, err := t.List(ctx, locationID, "templates", 1, 1000)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}

	for _, template := range templates {
		apiTemplate := apiv1.AnexiaTemplate{
			ID: template.ID,
		}
		response = append(response, apiTemplate)
	}

	return response, nil
}

func AnexiaVlansWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Anexia == nil {
		return nil, errors.NewNotFound("cloud spec for %s", clusterID)
	}
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	token, err := anexia.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	return ListAnexiaVlans(ctx, token)
}

func AnexiaTemplatesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Anexia == nil {
		return nil, errors.NewNotFound("cloud spec for %s", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	token, err := anexia.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to find Datacenter %q: %v", datacenterName, err))
	}

	return ListAnexiaTemplates(ctx, token, datacenter.Spec.Anexia.LocationID)
}

func getClient(token string) (client.Client, error) {
	tokenOpt := client.TokenFromString(token)
	return client.New(tokenOpt)
}
