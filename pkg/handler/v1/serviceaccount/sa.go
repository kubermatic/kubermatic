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

package serviceaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	serviceaccount "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// serviceAccountGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for service account.
var serviceAccountGroupsPrefixes = []string{
	rbac.EditorGroupNamePrefix,
	rbac.ViewerGroupNamePrefix,
	rbac.ProjectManagerGroupNamePrefix,
}

// CreateEndpoint adds the given service account to the given project
func CreateEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		saFromRequest := req.Body
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// check if service account name is already reserved in the project
		existingSAList, err := listSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, &saFromRequest)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingSAList) > 0 {
			return nil, errors.NewAlreadyExists("service account", saFromRequest.Name)
		}

		sa, err := createSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, saFromRequest)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalServiceAccountToExternal(sa), nil
	}
}

func listSA(ctx context.Context, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, sa *apiv1.ServiceAccount) ([]*kubermaticapiv1.User, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}

	var options *provider.ServiceAccountListOptions

	if sa != nil {
		options = &provider.ServiceAccountListOptions{ServiceAccountName: sa.Name}
	}

	if adminUserInfo.IsAdmin {
		return privilegedServiceAccount.ListUnsecuredProjectServiceAccount(project, options)
	}

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return serviceAccountProvider.ListProjectServiceAccount(userInfo, project, options)
}

func createSA(ctx context.Context, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, sa apiv1.ServiceAccount) (*kubermaticapiv1.User, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	groupName := rbac.GenerateActualGroupNameFor(project.Name, sa.Group)
	if adminUserInfo.IsAdmin {
		return privilegedServiceAccount.CreateUnsecuredProjectServiceAccount(project, sa.Name, groupName)
	}

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return serviceAccountProvider.CreateProjectServiceAccount(userInfo, project, sa.Name, groupName)
}

// ListEndpoint returns service accounts of the given project
func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, memberMapper provider.ProjectMemberMapper, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		saList, err := listSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var errorList []string
		response := make([]*apiv1.ServiceAccount, 0)
		for _, sa := range saList {
			externalSA := convertInternalServiceAccountToExternal(sa)
			if apiv1.ServiceAccountInactive == externalSA.Status {
				response = append(response, externalSA)
				continue
			}

			group, err := memberMapper.MapUserToGroup(sa.Spec.Email, project.Name)
			if err != nil {
				errorList = append(errorList, err.Error())
			} else {
				externalSA.Group = group
				response = append(response, externalSA)
			}
		}
		if len(errorList) > 0 {
			return response, errors.NewWithDetails(http.StatusInternalServerError, "failed to get some service accounts, please examine details field for more info", errorList)
		}

		return response, nil
	}
}

// UpdateEndpoint changes the service account group and/or name in the given project
func UpdateEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, memberMapper provider.ProjectMemberMapper, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		saFromRequest := req.Body

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		sa, err := getSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, req.ServiceAccountID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// update the service account name
		if sa.Spec.Name != saFromRequest.Name {
			// check if service account name is already reserved in the project
			existingSAList, err := listSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, &saFromRequest)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			if len(existingSAList) > 0 {
				return nil, errors.NewAlreadyExists("service account", saFromRequest.Name)
			}
			sa.Spec.Name = saFromRequest.Name
		}

		currentGroup, err := memberMapper.MapUserToGroup(sa.Spec.Email, project.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)

		}

		newGroup := rbac.GenerateActualGroupNameFor(project.Name, saFromRequest.Group)
		if newGroup != currentGroup {
			if sa.Labels == nil {
				sa.Labels = map[string]string{}
			}
			sa.Labels[serviceaccount.ServiceAccountLabelGroup] = newGroup
		}

		updatedSA, err := updateSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, sa)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result := convertInternalServiceAccountToExternal(updatedSA)
		result.Group = newGroup
		return result, nil
	}
}

func updateSA(ctx context.Context, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, sa *kubermaticapiv1.User) (*kubermaticapiv1.User, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}

	if adminUserInfo.IsAdmin {
		return privilegedServiceAccount.UpdateUnsecuredProjectServiceAccount(sa)
	}

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return serviceAccountProvider.UpdateProjectServiceAccount(userInfo, sa)
}

func getSA(ctx context.Context, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, serviceAccountID string, options *provider.ServiceAccountGetOptions) (*kubermaticapiv1.User, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}

	if adminUserInfo.IsAdmin {
		return privilegedServiceAccount.GetUnsecuredProjectServiceAccount(serviceAccountID, options)
	}

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return serviceAccountProvider.GetProjectServiceAccount(userInfo, serviceAccountID, options)
}

// DeleteEndpoint deletes the service account for the given project
func DeleteEndpoint(serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		// check if project exist
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// check if service account exist before deleting it
		_, err = getSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, req.ServiceAccountID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := deleteSA(ctx, serviceAccountProvider, privilegedServiceAccount, userInfoGetter, project, req.ServiceAccountID); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func deleteSA(ctx context.Context, serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccount provider.PrivilegedServiceAccountProvider, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, serviceAccountID string) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}

	if adminUserInfo.IsAdmin {
		return privilegedServiceAccount.DeleteUnsecuredProjectServiceAccount(serviceAccountID)
	}

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return err
	}
	return serviceAccountProvider.DeleteProjectServiceAccount(userInfo, serviceAccountID)
}

// addReq defines HTTP request for addServiceAccountToProject
// swagger:parameters addServiceAccountToProject
type addReq struct {
	common.ProjectReq
	// in: body
	Body apiv1.ServiceAccount
}

// serviceAccountIDReq represents a request that contains service account ID in the path
type serviceAccountIDReq struct {
	// in: path
	ServiceAccountID string `json:"serviceaccount_id"`
}

// updateReq defines HTTP request for updateServiceAccount
// swagger:parameters updateServiceAccount
type updateReq struct {
	addReq
	serviceAccountIDReq
}

// deleteReq defines HTTP request for deleteServiceAccount
// swagger:parameters deleteServiceAccount
type deleteReq struct {
	common.ProjectReq
	serviceAccountIDReq
}

// Validate validates DeleteEndpoint request
func (r deleteReq) Validate() error {
	if len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("the service account ID cannot be empty")
	}
	return nil
}

// Validate validates UpdateEndpoint request
func (r updateReq) Validate() error {
	err := r.addReq.Validate()
	if err != nil {
		return err
	}
	if r.ServiceAccountID != r.Body.ID {
		return fmt.Errorf("service account ID mismatch, you requested to update ServiceAccount = %s but body contains ServiceAccount = %s", r.ServiceAccountID, r.Body.ID)
	}
	return nil
}

// Validate validates addReq request
func (r addReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.Body.Name) == 0 || len(r.Body.Group) == 0 {
		return fmt.Errorf("the name, project ID and group cannot be empty")
	}

	for _, existingGroupPrefix := range serviceAccountGroupsPrefixes {
		if existingGroupPrefix == r.Body.Group {
			return nil
		}
	}
	return fmt.Errorf("invalid group name %s", r.Body.Group)
}

// DecodeAddReq  decodes an HTTP request into addReq
func DecodeAddReq(c context.Context, r *http.Request) (interface{}, error) {
	var req addReq

	prjReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// DecodeUpdateReq  decodes an HTTP request into updateReq
func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateReq

	prjReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	saIDReq, err := decodeServiceAccountIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ServiceAccountID = saIDReq.ServiceAccountID

	return req, nil
}

// DecodeDeleteeReq  decodes an HTTP request into deleteReq
func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteReq

	prjReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(common.ProjectReq)

	saIDReq, err := decodeServiceAccountIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ServiceAccountID = saIDReq.ServiceAccountID

	return req, nil
}

func decodeServiceAccountIDReq(_ context.Context, r *http.Request) (serviceAccountIDReq, error) {
	var req serviceAccountIDReq

	saID, ok := mux.Vars(r)["serviceaccount_id"]
	if !ok {
		return req, fmt.Errorf("'serviceaccount_id' parameter is required")
	}
	req.ServiceAccountID = saID

	return req, nil
}

func convertInternalServiceAccountToExternal(internal *kubermaticapiv1.User) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internal.Name,
			Name:              internal.Spec.Name,
			CreationTimestamp: apiv1.NewTime(internal.CreationTimestamp.Time),
		},
		Group:  internal.Labels[serviceaccount.ServiceAccountLabelGroup],
		Status: getStatus(internal),
	}
}

func getStatus(serviceAccount *kubermaticapiv1.User) string {
	if _, ok := serviceAccount.Labels[serviceaccount.ServiceAccountLabelGroup]; ok {
		return apiv1.ServiceAccountInactive
	}
	return apiv1.ServiceAccountActive
}
