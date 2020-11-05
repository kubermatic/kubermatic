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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listConstraintsReq)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		constraintList, err := constraintProvider.List(clus)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiC := make([]*apiv2.Constraint, len(constraintList.Items))
		for i, ct := range constraintList.Items {
			apiC[i] = convertInternalToAPIConstraint(&ct)
		}

		return apiC, nil
	}
}

func convertInternalToAPIConstraint(c *v1.Constraint) *apiv2.Constraint {
	return &apiv2.Constraint{
		Name: c.Name,
		Spec: c.Spec,
	}
}

func convertAPIToInternalConstraint(name, namespace string, spec v1.ConstraintSpec) *v1.Constraint {
	return &v1.Constraint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
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
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(constraintReq)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		constraint, err := constraintProvider.Get(clus, req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToAPIConstraint(constraint), nil
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(constraintReq)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		err = deleteConstraint(ctx, userInfoGetter, constraintProvider, privilegedConstraintProvider, clus, req.ProjectID, req.Name)
		return nil, common.KubernetesErrorToHTTPError(err)
	}
}

func deleteConstraint(ctx context.Context, userInfoGetter provider.UserInfoGetter, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider, cluster *v1.Cluster, projectID, constraintName string) error {

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		return privilegedConstraintProvider.DeleteUnsecured(cluster, constraintName)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return err
	}

	return constraintProvider.Delete(cluster, userInfo, constraintName)
}

// constraintReq defines HTTP request for a constraint endpoint
// swagger:parameters getConstraint deleteConstraint
type constraintReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	Name string `json:"constraint_name"`
}

func DecodeConstraintReq(c context.Context, r *http.Request) (interface{}, error) {
	var req constraintReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)

	req.Name = mux.Vars(r)["constraint_name"]
	if req.Name == "" {
		return "", errors.New("'constraint_name' parameter is required but was not provided")
	}

	return req, nil
}

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider,
	constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createConstraintReq)

		err := req.ValidateCreateConstraintReq(constraintTemplateProvider)
		if err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Sprintf("Validation failed, constraint needs to have an existing constraint template: %v", err))
		}

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		constraint := convertAPIToInternalConstraint(req.Body.Name, clus.Status.NamespaceName, req.Body.Spec)

		ct, err := createConstraint(ctx, userInfoGetter, constraintProvider, privilegedConstraintProvider, req.ProjectID, constraint)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIConstraint(ct), nil
	}
}

func createConstraint(ctx context.Context, userInfoGetter provider.UserInfoGetter, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider, projectID string, constraint *v1.Constraint) (*v1.Constraint, error) {

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedConstraintProvider.CreateUnsecured(constraint)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return constraintProvider.Create(userInfo, constraint)
}

// swagger:parameters createConstraint
type createConstraintReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body constraintBody
}

type constraintBody struct {
	// Name is the name for the constraint
	Name string `json:"name"`
	// Spec is the constraint specification
	Spec v1.ConstraintSpec
}

func DecodeCreateConstraintReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createConstraintReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func (req *createConstraintReq) ValidateCreateConstraintReq(constraintTemplateProvider provider.ConstraintTemplateProvider) error {
	_, err := constraintTemplateProvider.Get(req.Body.Spec.ConstraintType)
	return err
}
