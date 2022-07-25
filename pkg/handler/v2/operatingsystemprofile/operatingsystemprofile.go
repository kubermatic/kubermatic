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
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
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

// seedReq represents a request for referencing a seed
// swagger:parameters listIPAMPools
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

func ListOperatingSystemProfilesEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("%s doesn't have admin rights", userInfo.Email))
		}

		privilegedOperatingSystemProfileProvider := ctx.Value(middleware.PrivilegedOperatingSystemProfileProviderContextKey).(provider.PrivilegedOperatingSystemProfileProvider)

		ospList, err := privilegedOperatingSystemProfileProvider.ListUnsecured(ctx)
		if err != nil {
			return nil, err
		}

		var resp []*apiv2.OperatingSystemProfile

		for _, osp := range ospList.Items {
			ospModel := &apiv2.OperatingSystemProfile{
				Name:      osp.Name,
				Namespace: osp.Namespace,
			}
			resp = append(resp, ospModel)
		}

		for _, osp := range defaultOperatingSystemProfiles {
			ospModel := osp
			resp = append(resp, &ospModel)
		}

		return resp, nil
	}
}
