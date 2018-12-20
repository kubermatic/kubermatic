package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func vsphereNetworksEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VSphereNetworksReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereNetworksReq, got = %T", request)
		}

		return getVsphereNetworks(providers, req.Username, req.Password, req.DatacenterName)
	}
}

func vsphereNetworksNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereNetworksNoCredentialsReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.VSphere == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName
		vSpec := cluster.Spec.Cloud.VSphere
		return getVsphereNetworks(providers, vSpec.Username, vSpec.Password, datacenterName)
	}
}

func getVsphereNetworks(providers provider.CloudRegistry, username, password, datacenterName string) ([]apiv1.VSphereNetwork, error) {
	vsProviderInterface, ok := providers[provider.VSphereCloudProvider]
	if !ok {
		return nil, fmt.Errorf("unable to get %s provider", provider.VSphereCloudProvider)
	}

	vsProvider, ok := vsProviderInterface.(*vsphere.Provider)
	if !ok {
		return nil, fmt.Errorf("unable to cast vsProviderInterface to *vsphere.Provider")
	}

	networks, err := vsProvider.GetNetworks(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		VSphere: &kubermaticv1.VSphereCloudSpec{
			InfraManagementUser: kubermaticv1.VSphereCredentials{
				Username: username,
				Password: password,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var apiNetworks []apiv1.VSphereNetwork
	for _, net := range networks {
		apiNetworks = append(apiNetworks, apiv1.VSphereNetwork{Name: net.Name})
	}

	return apiNetworks, nil
}
