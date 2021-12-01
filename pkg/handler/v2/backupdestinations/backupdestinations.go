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

	v2 "k8c.io/kubermatic/v2/pkg/api/v2"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getBackupDestinationNamesReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		seeds, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		seedName, ok := c.Labels[resources.SeedLabelKey]
		if !ok {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("Seed label not present for Cluster %q", c.Name))
		}

		seed, ok := seeds[seedName]
		if !ok {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("Seed %q not found", seedName))
		}

		var destinations v2.BackupDestinationNames
		if seed.Spec.EtcdBackupRestore == nil {
			return destinations, nil
		}
		for name, _ := range seed.Spec.EtcdBackupRestore.Destinations {
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
