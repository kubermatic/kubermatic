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

func VsphereNetworksEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VSphereNetworksReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereNetworksReq, got = %T", request)
		}

		username := req.Username
		password := req.Password

		if len(req.Credential) > 0 && credentialManager.GetPresets().VSphere.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().VSphere.Credentials {
				if credential.Name == req.Credential {
					username = credential.Username
					password = credential.Password
					break
				}
			}
		}

		return getVsphereNetworks(seedsGetter, username, password, req.DatacenterName)
	}
}

func VsphereNetworksNoCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
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
		return getVsphereNetworks(seedsGetter, vSpec.Username, vSpec.Password, datacenterName)
	}
}

func getVsphereNetworks(seedsGetter provider.SeedsGetter, username, password, datacenterName string) ([]apiv1.VSphereNetwork, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}
	vsProvider, err := vsphere.NewCloudProvider(datacenter)
	if err != nil {
		return nil, err
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
		apiNetworks = append(apiNetworks, apiv1.VSphereNetwork{
			Name:         net.Name,
			Type:         net.Type,
			RelativePath: net.RelativePath,
			AbsolutePath: net.AbsolutePath,
		})
	}

	return apiNetworks, nil
}

func VsphereFoldersEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VSphereFoldersReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereFoldersReq, got = %T", request)
		}

		username := req.Username
		password := req.Password

		if len(req.Credential) > 0 && credentialManager.GetPresets().VSphere.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().VSphere.Credentials {
				if credential.Name == req.Credential {
					username = credential.Username
					password = credential.Password
					break
				}
			}
		}

		return getVsphereFolders(seedsGetter, username, password, req.DatacenterName)
	}
}

func VsphereFoldersNoCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereFoldersNoCredentialsReq)
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
		return getVsphereFolders(seedsGetter, vSpec.Username, vSpec.Password, datacenterName)
	}
}

func getVsphereFolders(seedsGetter provider.SeedsGetter, username, password, datacenterName string) ([]apiv1.VSphereFolder, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}
	vsProvider, err := vsphere.NewCloudProvider(datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to create new cloud provider: %v", err)
	}

	folders, err := vsProvider.GetVMFolders(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		VSphere: &kubermaticv1.VSphereCloudSpec{
			InfraManagementUser: kubermaticv1.VSphereCredentials{
				Username: username,
				Password: password,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %v", err)
	}

	var apiFolders []apiv1.VSphereFolder
	for _, folder := range folders {
		apiFolders = append(apiFolders, apiv1.VSphereFolder{Path: folder.Path})
	}

	return apiFolders, nil
}

// VSphereNetworksReq represent a request for vsphere networks
type VSphereNetworksReq struct {
	Username       string
	Password       string
	DatacenterName string
	Credential     string
}

func DecodeVSphereNetworksReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")

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

// VSphereFoldersReq represent a request for vsphere folders
type VSphereFoldersReq struct {
	Username       string
	Password       string
	DatacenterName string
	Credential     string
}

func DecodeVSphereFoldersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereFoldersReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// VSphereFoldersNoCredentialsReq represent a request for vsphere folders
// swagger:parameters listVSphereFoldersNoCredentials
type VSphereFoldersNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeVSphereFoldersNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereFoldersNoCredentialsReq
	lr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = lr.(common.GetClusterReq)
	return req, nil
}
