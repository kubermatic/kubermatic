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

package backupdestinations

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getBackupDestinationNamesReq)

		seed, err := getSeed(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		var destinations apiv2.BackupDestinationNames
		if seed.Spec.EtcdBackupRestore == nil {
			return destinations, nil
		}
		for name := range seed.Spec.EtcdBackupRestore.Destinations {
			destinations = append(destinations, name)
		}
		return destinations, nil
	}
}

// getBackupDestinationNamesReq defines HTTP request for getting backup destination names for a cluster
// swagger:parameters getBackupDestinationNames
type getBackupDestinationNamesReq struct {
	cluster.GetClusterReq
}

func DecodeGetBackupDestinationNamesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getBackupDestinationNamesReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}

func getSeed(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string,
) (*kubermaticv1.Seed, error) {
	clusterProvider, ok := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "no cluster in request")
	}
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

	// check if user can access the cluster
	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, err = handlercommon.GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	seed, ok := seeds[clusterProvider.GetSeedName()]
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("Seed %q not found", clusterProvider.GetSeedName()))
	}
	return seed, nil
}
