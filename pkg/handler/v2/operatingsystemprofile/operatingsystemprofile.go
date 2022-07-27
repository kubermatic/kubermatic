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
	"net/http"

	"github.com/go-kit/kit/endpoint"

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
		Name: "osp-amzn2",
	},
	{
		Name: "osp-centos",
	},
	{
		Name: "osp-flatcar",
	},
	{
		Name: "osp-rhel",
	},
	{
		Name: "osp-rockylinux",
	},
	{
		Name: "osp-sles",
	},
	{
		Name: "osp-ubuntu",
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

func ListOperatingSystemProfilesEndpointForCluster(userInfoGetter provider.UserInfoGetter, operatingSystemProfile provider.OperatingSystemProfileProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(listOperatingSystemProfilesReq)

		userClusterNamespace, err := clusterNamespaceFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		ospList, err := operatingSystemProfile.ListUnsecured(ctx, userClusterNamespace)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var resp []*apiv2.OperatingSystemProfile
		for _, osp := range ospList.Items {
			ospModel := &apiv2.OperatingSystemProfile{
				Name: osp.Name,
			}
			resp = append(resp, ospModel)
		}

		return resp, nil
	}
}

func ListOperatingSystemProfilesEndpoint(seedsGetter provider.SeedsGetter, operatingSystemProfile provider.OperatingSystemProfileProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Fetch OSPs for all the seeds.
		var osps []osmv1alpha1.OperatingSystemProfile
		for _, seed := range seeds {
			ospList, err := operatingSystemProfile.ListUnsecured(ctx, seed.Namespace)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			osps = append(osps, ospList.Items...)
		}

		// Add all custom OSPs to the response.
		var resp []*apiv2.OperatingSystemProfile
		for _, osp := range osps {
			ospModel := &apiv2.OperatingSystemProfile{
				Name: osp.Name,
			}
			resp = append(resp, ospModel)
		}

		// Add all default OSPs to the response.
		for _, osp := range defaultOperatingSystemProfiles {
			ospModel := osp
			resp = append(resp, &ospModel)
		}

		return resp, nil
	}
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
