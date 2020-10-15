/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package constraint

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listConstraintsReq)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		constraintList, err := getConstraintList(ctx, userInfoGetter, clus, req.ProjectID, constraintProvider, privilegedConstraintProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiC := make([]*apiv2.Constraint, 0)
		for _, ct := range constraintList.Items {
			apiC = append(apiC, convertCToAPI(&ct))
		}

		return apiC, nil
	}
}

func getConstraintList(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	clus *v1.Cluster, projectID string, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider) (*v1.ConstraintList, error) {

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		constraintList, err := privilegedConstraintProvider.ListUnsecured(clus)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return constraintList, nil
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	constraintList, err := constraintProvider.List(userInfo, clus)
	if err != nil {
		return nil, err
	}
	return constraintList, nil
}

func convertCToAPI(c *v1.Constraint) *apiv2.Constraint {
	return &apiv2.Constraint{
		Name: c.Name,
		Spec: c.Spec,
	}
}

// listConstraintsReq defines HTTP request for list constraints endpoint
// swagger:parameters listConstraints
type listConstraintsReq struct {
	cluster.GetClusterReq
}

func DecodeListConstraintsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listConstraintsReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)

	return req, nil
}

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConstraintReq)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		constraint, err := getConstraint(ctx, userInfoGetter, clus, req.Name, req.ProjectID, constraintProvider, privilegedConstraintProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertCToAPI(constraint), nil
	}
}

func getConstraint(ctx context.Context, userInfoGetter provider.UserInfoGetter,
	clus *v1.Cluster, constraintName, projectID string, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider) (*v1.Constraint, error) {

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		constraint, err := privilegedConstraintProvider.GetUnsecured(clus, constraintName)
		if err != nil {
			return nil, err
		}
		return constraint, nil
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	constraint, err := constraintProvider.Get(userInfo, clus, constraintName)
	if err != nil {
		return nil, err
	}
	return constraint, nil
}

// getConstraintReq defines HTTP request for get constraint endpoint
// swagger:parameters getConstraint
type getConstraintReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	Name string `json:"constraint_name"`
}

func DecodeGetConstraintReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getConstraintReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)

	req.Name = mux.Vars(r)["constraint_name"]
	if req.Name == "" {
		return "", fmt.Errorf("'constraint_name' parameter is required but was not provided")
	}

	return req, nil
}
