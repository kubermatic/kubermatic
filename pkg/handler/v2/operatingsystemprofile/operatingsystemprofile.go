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

package operatingsystemprofile

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

// TODO: Find a way to populate these dynamically.
// These OSPs are created after cluster creation by OSM. We need a way to display these
// before cluster creation. For example, these are required in the cluster creation wizard for
// KKP dashboard.
// Namespace is purposefully left empty since we cannot determine namespace of these resources before cluster creation.
var defaultOperatingSystemProfiles = []apiv2.OperatingSystemProfile{
	{
		Name:                    "osp-amzn2",
		OperatingSystem:         "amzn2",
		SupportedCloudProviders: []string{"aws"},
	},
	{
		Name:                    "osp-centos",
		OperatingSystem:         "centos",
		SupportedCloudProviders: []string{"alibaba", "aws", "azure", "digitalocean", "equinixmetal", "hetzner", "kubevirt", "nutanix", "openstack", "vsphere"},
	},
	{
		Name:                    "osp-flatcar",
		OperatingSystem:         "flatcar",
		SupportedCloudProviders: []string{"aws", "azure", "equinixmetal", "kubevirt", "openstack", "vsphere"},
	},
	{
		Name:                    "osp-rhel",
		OperatingSystem:         "rhel",
		SupportedCloudProviders: []string{"aws", "azure", "equinixmetal", "kubevirt", "openstack", "vsphere"},
	},
	{
		Name:                    "osp-rockylinux",
		OperatingSystem:         "rockylinux",
		SupportedCloudProviders: []string{"aws", "azure", "digitalocean", "equinixmetal", "hetzner", "kubevirt", "openstack", "vsphere"},
	},
	{
		Name:                    "osp-sles",
		OperatingSystem:         "sles",
		SupportedCloudProviders: []string{"aws"},
	},
	{
		Name:                    "osp-ubuntu",
		OperatingSystem:         "ubuntu",
		SupportedCloudProviders: []string{"alibaba", "aws", "azure", "digitalocean", "equinixmetal", "gce", "hetzner", "kubevirt", "nutanix", "openstack", "vmware-cloud-director", "vsphere"},
	},
}

// listOperatingSystemProfilesReq defines HTTP request for listOperatingSystemProfilesForCluster
// swagger:parameters listOperatingSystemProfilesForCluster
type listOperatingSystemProfilesReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

// GetSeedCluster returns the SeedCluster object.
func (req listOperatingSystemProfilesReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// seedReq represents a request for referencing a seed
// swagger:parameters listOperatingSystemProfiles
type seedReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
}

func (req seedReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

func DecodeSeedReq(c context.Context, r *http.Request) (interface{}, error) {
	var req seedReq
	seedName := mux.Vars(r)["seed_name"]
	if seedName == "" {
		return nil, fmt.Errorf("'seed_name' parameter is required but was not provided")
	}
	req.SeedName = seedName
	return req, nil
}

func DecodeListOperatingSystemProfiles(c context.Context, r *http.Request) (interface{}, error) {
	var req listOperatingSystemProfilesReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	return req, nil
}

func ListOperatingSystemProfilesEndpointForCluster(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(listOperatingSystemProfilesReq)

		userClusterNamespace, err := clusterNamespaceFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		privilegedOperatingSystemProfileProvider := ctx.Value(middleware.PrivilegedOperatingSystemProfileProviderContextKey).(provider.PrivilegedOperatingSystemProfileProvider)

		ospList, err := privilegedOperatingSystemProfileProvider.ListUnsecuredForUserClusterNamespace(ctx, userClusterNamespace)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertOperatingSystemProfileToAPIResponse(ospList), nil
	}
}

func ListOperatingSystemProfilesEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		privilegedOperatingSystemProfileProvider := ctx.Value(middleware.PrivilegedOperatingSystemProfileProviderContextKey).(provider.PrivilegedOperatingSystemProfileProvider)

		ospList, err := privilegedOperatingSystemProfileProvider.ListUnsecured(ctx)
		if err != nil {
			return nil, err
		}

		resp := convertOperatingSystemProfileToAPIResponse(ospList)

		for _, osp := range defaultOperatingSystemProfiles {
			ospModel := osp
			resp = append(resp, &ospModel)
		}

		return resp, nil
	}
}

func convertOperatingSystemProfileToAPIResponse(ospList *osmv1alpha1.OperatingSystemProfileList) []*apiv2.OperatingSystemProfile {
	var resp []*apiv2.OperatingSystemProfile
	for _, osp := range ospList.Items {
		var supportedOperatingSystems []string
		for _, os := range osp.Spec.SupportedCloudProviders {
			supportedOperatingSystems = append(supportedOperatingSystems, string(os.Name))
		}

		ospModel := &apiv2.OperatingSystemProfile{
			Name:                    osp.Name,
			OperatingSystem:         string(osp.Spec.OSName),
			SupportedCloudProviders: supportedOperatingSystems,
		}
		resp = append(resp, ospModel)
	}
	return resp
}

func clusterNamespaceFromContext(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (string, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := clusterProvider.Get(ctx, userInfo, clusterID, &provider.ClusterGetOptions{})
	if err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}
	return cluster.Status.NamespaceName, nil
}
