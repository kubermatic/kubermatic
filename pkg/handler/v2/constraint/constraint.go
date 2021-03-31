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
	"io/ioutil"
	"net/http"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"k8s.io/utils/pointer"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ConstraintsGroup      = "constraints.gatekeeper.sh"
	ConstraintsVersion    = "v1beta1"
	ConstraintNamespace   = "kubermatic"
	constraintStatusField = "status"
)

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listConstraintsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		clusterCli, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, clus, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		constraintList, err := constraintProvider.List(clus)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// collect constraint types
		cKinds := sets.String{}
		// create apiConstraint map
		apiConstraintMap := make(map[string]*apiv2.Constraint, len(constraintList.Items))
		for _, ct := range constraintList.Items {
			cKinds.Insert(ct.Spec.ConstraintType)

			apiConstraint := convertInternalToAPIConstraint(&ct)
			apiConstraint.Status = &apiv2.ConstraintStatus{Synced: pointer.BoolPtr(false)}

			apiConstraintMap[genConstraintKey(ct.Spec.ConstraintType, ct.Name)] = apiConstraint
		}

		// List all diffrerent gatekeeper constraints and get status
		for kind := range cKinds {
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   ConstraintsGroup,
				Version: ConstraintsVersion,
				Kind:    kind + "List",
			})
			if err := clusterCli.List(ctx, list); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			for _, uc := range list.Items {
				constraintStatus, err := getConstraintStatus(&uc)
				if err != nil {
					return nil, err
				}
				if apiConstraint, ok := apiConstraintMap[genConstraintKey(kind, uc.GetName())]; ok {
					apiConstraint.Status = constraintStatus
				}
			}
		}
		var apiConstraintList []*apiv2.Constraint
		for _, apiConstraint := range apiConstraintMap {
			apiConstraintList = append(apiConstraintList, apiConstraint)
		}

		return apiConstraintList, nil
	}
}

func genConstraintKey(constraintType, name string) string {
	return fmt.Sprintf("%s-%s", constraintType, name)
}

func getConstraintStatus(uc *unstructured.Unstructured) (*apiv2.ConstraintStatus, error) {
	status, _, err := unstructured.NestedFieldNoCopy(uc.Object, constraintStatusField)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("error getting status: %v", err))
	}

	constraintStatus := &apiv2.ConstraintStatus{}

	statusRaw, err := json.Marshal(status)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("error marshalling constraint status: %v", err))
	}

	err = json.Unmarshal(statusRaw, constraintStatus)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("error unmarshalling constraint status: %v", err))
	}

	constraintStatus.Synced = pointer.BoolPtr(true)
	return constraintStatus, nil
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
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		clusterCli, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, clus, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		constraint, err := constraintProvider.Get(clus, req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// convert to API constraint
		apiConstraint := convertInternalToAPIConstraint(constraint)

		instance := &unstructured.Unstructured{}
		instance.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   ConstraintsGroup,
			Version: ConstraintsVersion,
			Kind:    constraint.Spec.ConstraintType,
		})

		if err := clusterCli.Get(ctx, types.NamespacedName{Namespace: ConstraintNamespace, Name: constraint.Name}, instance); err != nil {
			// Can't get status, because the Kubermatic Constraint is not synced yet as a Gatekeeper Constraint on the user cluster
			apiConstraint.Status = &apiv2.ConstraintStatus{Synced: pointer.BoolPtr(false)}
			return apiConstraint, nil
		}

		cStatus, err := getConstraintStatus(instance)
		if err != nil {
			return nil, err
		}
		apiConstraint.Status = cStatus

		return apiConstraint, nil
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
	_, err := constraintTemplateProvider.Get(strings.ToLower(req.Body.Spec.ConstraintType))
	return err
}

func PatchEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchConstraintReq)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		// get Constraint
		originalConstraint, err := constraintProvider.Get(clus, req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		originalAPIConstraint := convertInternalToAPIConstraint(originalConstraint)

		// patch
		originalJSON, err := json.Marshal(originalAPIConstraint)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to convert current constraint: %v", err))
		}

		patchedJSON, err := jsonpatch.MergePatch(originalJSON, req.Patch)
		if err != nil {
			return nil, utilerrors.New(http.StatusBadRequest, fmt.Sprintf("failed to merge patch ct: %v", err))
		}

		var patched *apiv2.Constraint
		err = json.Unmarshal(patchedJSON, &patched)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshall patch ct: %v", err))
		}

		patchedConstraint := convertAPIToInternalConstraint(req.Name, clus.Status.NamespaceName, patched.Spec)

		// ConstraintType cannot be changed by patch
		patchedConstraint.Spec.ConstraintType = originalConstraint.Spec.ConstraintType

		// restore ResourceVersion to make patching safer and tests work more easily
		patchedConstraint.ResourceVersion = originalConstraint.ResourceVersion

		ct, err := updateConstraint(ctx, userInfoGetter, constraintProvider, privilegedConstraintProvider, req.ProjectID, patchedConstraint)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIConstraint(ct), nil
	}
}

func updateConstraint(ctx context.Context, userInfoGetter provider.UserInfoGetter, constraintProvider provider.ConstraintProvider,
	privilegedConstraintProvider provider.PrivilegedConstraintProvider, projectID string, constraint *v1.Constraint) (*v1.Constraint, error) {

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedConstraintProvider.UpdateUnsecured(constraint)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return constraintProvider.Update(userInfo, constraint)
}

// patchConstraintReq defines HTTP request for patching constraints
// swagger:parameters patchConstraint
type patchConstraintReq struct {
	constraintReq
	// in: body
	Patch json.RawMessage
}

// DecodePatchConstraintReq decodes http request into patchConstraintReq
func DecodePatchConstraintReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchConstraintReq

	ctReq, err := DecodeConstraintReq(c, r)
	if err != nil {
		return nil, err
	}
	req.constraintReq = ctReq.(constraintReq)

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}
