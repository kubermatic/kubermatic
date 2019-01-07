package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func VsphereNetworksEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VSphereNetworksReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereNetworksReq, got = %T", request)
		}

		return getVsphereNetworks(providers, req.Username, req.Password, req.DatacenterName)
	}
}

func VsphereNetworksNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereNetworksNoCredentialsReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
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

// VSphereNetworksReq represent a request for vsphere networks
type VSphereNetworksReq struct {
	Username       string
	Password       string
	DatacenterName string
}

func DecodeVSphereNetworksReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}

// VSphereNetworksNoCredentialsReq represent a request for vsphere networks
// swagger:parameters listVSphereNetworksNoCredentials
type VSphereNetworksNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeVSphereNetworksNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksNoCredentialsReq
	lr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = lr.(common.GetClusterReq)
	return req, nil
}
