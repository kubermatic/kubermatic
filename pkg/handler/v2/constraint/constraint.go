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
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider, privilegedConstraintProvider provider.PrivilegedConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listConstraintsReq)
		var constraintList *v1.ConstraintList

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if adminUserInfo.IsAdmin {
			constraintList, err = privilegedConstraintProvider.ListUnsecured(cluster)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, err
		}
		constraintList, err = constraintProvider.List(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Conversion and return
		apiC := make([]*apiv2.Constraint, 0)
		for _, ct := range constraintList.Items {
			apiC = append(apiC, convertCToAPI(&ct))
		}

		return apiC, nil
	}
}

func convertCToAPI(c *v1.Constraint) *apiv2.Constraint {
	return &apiv2.Constraint{
		Name: c.Name,
		Spec: c.ConstraintSpec,
	}
}

// listConstraintsReq defines HTTP request for list constraints endpoint
// swagger:parameters listConstraints
type listConstraintsReq struct {
	common.GetClusterReq
}

func DecodeListConstraintsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listConstraintsReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)

	return req, nil
}

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider, privilegedConstraintProvider provider.PrivilegedConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConstraintReq)
		var constraint *v1.Constraint

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if adminUserInfo.IsAdmin {
			constraint, err = privilegedConstraintProvider.GetUnsecured(cluster, req.Name)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, err
		}
		constraint, err = constraintProvider.Get(userInfo, cluster, req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertCToAPI(constraint), nil
	}
}

// getConstraintReq defines HTTP request for get constraint endpoint
// swagger:parameters getConstraint
type getConstraintReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	Name string `json:"constraint_name"`
}

func DecodeGetConstraintReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getConstraintReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)

	req.Name = mux.Vars(r)["constraint_name"]
	if req.Name == "" {
		return "", fmt.Errorf("'constraint_name' parameter is required but was not provided")
	}

	return req, nil
}
