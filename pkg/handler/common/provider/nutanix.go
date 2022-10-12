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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	nutanixprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/nutanix"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

type NutanixCredentials struct {
	ProxyURL string
	Username string
	Password string
}

type NutanixClientSet interface {
	ListNutanixClusters(ctx context.Context) (apiv1.NutanixClusterList, error)
	ListNutanixProjects(ctx context.Context) (apiv1.NutanixProjectList, error)
	ListNutanixSubnets(ctx context.Context, clusterName, projectName string) (apiv1.NutanixSubnetList, error)
	ListNutanixCategories(ctx context.Context) (apiv1.NutanixCategoryList, error)
	ListNutanixCategoryValues(ctx context.Context, categoryName string) (apiv1.NutanixCategoryValueList, error)
}

type nutanixClientImpl struct {
	dc    *kubermaticv1.DatacenterSpecNutanix
	creds *NutanixCredentials
}

var NewNutanixClient = func(dc *kubermaticv1.DatacenterSpecNutanix, creds *NutanixCredentials) NutanixClientSet {
	return &nutanixClientImpl{
		dc:    dc,
		creds: creds,
	}
}

func (n *nutanixClientImpl) ListNutanixClusters(ctx context.Context) (apiv1.NutanixClusterList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(n.dc.Endpoint, n.dc.Port, &n.dc.AllowInsecure, n.creds.ProxyURL, n.creds.Username, n.creds.Password)
	if err != nil {
		return nil, err
	}

	clusterResp, err := nutanixprovider.GetClusters(ctx, clientSet)
	if err != nil {
		return nil, err
	}

	var clusters apiv1.NutanixClusterList
	for _, cluster := range clusterResp {
		if cluster.Status != nil && *cluster.Status.Name != "Unnamed" {
			clusters = append(clusters, apiv1.NutanixCluster{
				Name: *cluster.Status.Name,
			})
		}
	}

	return clusters, nil
}

func (n *nutanixClientImpl) ListNutanixProjects(ctx context.Context) (apiv1.NutanixProjectList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(n.dc.Endpoint, n.dc.Port, &n.dc.AllowInsecure, n.creds.ProxyURL, n.creds.Username, n.creds.Password)
	if err != nil {
		return nil, err
	}

	projectsResp, err := nutanixprovider.GetProjects(ctx, clientSet)
	if err != nil {
		return nil, err
	}

	var projects apiv1.NutanixProjectList
	for _, cluster := range projectsResp {
		projects = append(projects, apiv1.NutanixProject{
			Name: cluster.Status.Name,
		})
	}

	return projects, nil
}

func (n *nutanixClientImpl) ListNutanixSubnets(ctx context.Context, clusterName, projectName string) (apiv1.NutanixSubnetList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(n.dc.Endpoint, n.dc.Port, &n.dc.AllowInsecure, n.creds.ProxyURL, n.creds.Username, n.creds.Password)
	if err != nil {
		return nil, err
	}

	return listNutanixSubnets(ctx, clientSet, clusterName, projectName)
}

func listNutanixSubnets(ctx context.Context, client *nutanixprovider.ClientSet, clusterName, projectName string) (apiv1.NutanixSubnetList, error) {
	subnetResp, err := nutanixprovider.GetSubnets(ctx, client, clusterName, projectName)
	if err != nil {
		return nil, err
	}

	var subnets apiv1.NutanixSubnetList
	for _, subnet := range subnetResp {
		subnets = append(subnets, apiv1.NutanixSubnet{
			Name:   *subnet.Status.Name,
			Type:   *subnet.Status.Resources.SubnetType,
			VlanID: int(*subnet.Status.Resources.VlanID),
		})
	}

	return subnets, nil
}

func getClientSetFromCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (client *nutanixprovider.ClientSet, clusterName string, projectName string, err error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, "", "", err
	}
	if cluster.Spec.Cloud.Nutanix == nil {
		return nil, "", "", utilerrors.NewNotFound("no cloud spec for %s", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, "", "", utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, "", "", common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to find Datacenter %q: %w", datacenterName, err)
	}

	clusterName = cluster.Spec.Cloud.Nutanix.ClusterName
	projectName = cluster.Spec.Cloud.Nutanix.ProjectName

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	client, err = nutanixprovider.GetClientSet(datacenter.Spec.Nutanix, cluster.Spec.Cloud.Nutanix, secretKeySelector)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get client set: %w", err)
	}

	return
}

func (n *nutanixClientImpl) ListNutanixCategories(ctx context.Context) (apiv1.NutanixCategoryList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(n.dc.Endpoint, n.dc.Port, &n.dc.AllowInsecure, n.creds.ProxyURL, n.creds.Username, n.creds.Password)
	if err != nil {
		return nil, err
	}

	return listNutanixCategories(ctx, clientSet)
}

func listNutanixCategories(ctx context.Context, client *nutanixprovider.ClientSet) (apiv1.NutanixCategoryList, error) {
	categoryResp, err := nutanixprovider.GetCategories(ctx, client)
	if err != nil {
		return nil, err
	}

	var categories apiv1.NutanixCategoryList
	for _, category := range categoryResp {
		// do not list categories used by KKP or KubeOne; they should not be visible in the UI
		if category.Name != nil && *category.Name != nutanixprovider.ClusterCategoryName && *category.Name != nutanixprovider.ProjectCategoryName && *category.Name != "KubeOneCluster" {
			categories = append(categories, apiv1.NutanixCategory{
				Name:          *category.Name,
				Description:   *category.Description,
				SystemDefined: *category.SystemDefined,
			})
		}
	}

	return categories, nil
}

func (n *nutanixClientImpl) ListNutanixCategoryValues(ctx context.Context, categoryName string) (apiv1.NutanixCategoryValueList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(n.dc.Endpoint, n.dc.Port, &n.dc.AllowInsecure, n.creds.ProxyURL, n.creds.Username, n.creds.Password)
	if err != nil {
		return nil, err
	}

	return listNutanixCategoryValues(ctx, clientSet, categoryName)
}

func listNutanixCategoryValues(ctx context.Context, client *nutanixprovider.ClientSet, categoryName string) (apiv1.NutanixCategoryValueList, error) {
	categoryValueResp, err := nutanixprovider.GetCategoryValues(ctx, client, categoryName)
	if err != nil {
		return nil, err
	}

	var categoryValues apiv1.NutanixCategoryValueList
	for _, value := range categoryValueResp {
		categoryValues = append(categoryValues, apiv1.NutanixCategoryValue{
			Value: *value.Value,
		})
	}

	return categoryValues, nil
}
