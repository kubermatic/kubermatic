package serviceaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	serviceaccount "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// serviceAccountGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for service account.
var serviceAccountGroupsPrefixes = []string{
	rbac.EditorGroupNamePrefix,
	rbac.ViewerGroupNamePrefix,
}

// CreateEndpoint adds the given service account to the given project
func CreateEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addReq)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		saFromRequest := req.Body
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// check if service account name is already reserved in the project
		existingSAList, err := serviceAccountProvider.List(userInfo, project, &provider.ServiceAccountListOptions{ServiceAccountName: saFromRequest.Name})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingSAList) > 0 {
			return nil, errors.NewAlreadyExists("service account", saFromRequest.Name)
		}

		groupName := rbac.GenerateActualGroupNameFor(project.Name, saFromRequest.Group)
		sa, err := serviceAccountProvider.Create(userInfo, project, saFromRequest.Name, groupName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalServiceAccountToExternal(sa), nil
	}
}

// ListEndpoint returns service accounts of the given project
func ListEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, memberMapper provider.ProjectMemberMapper, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		saList, err := serviceAccountProvider.List(userInfo, project, nil)
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
func UpdateEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, memberMapper provider.ProjectMemberMapper, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
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
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		sa, err := serviceAccountProvider.Get(userInfo, req.ServiceAccountID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// update the service account name
		if sa.Spec.Name != saFromRequest.Name {
			// check if service account name is already reserved in the project
			existingSAList, err := serviceAccountProvider.List(userInfo, project, &provider.ServiceAccountListOptions{ServiceAccountName: saFromRequest.Name})
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
			sa.Labels[serviceaccount.ServiceAccountLabelGroup] = newGroup
		}

		updatedSA, err := serviceAccountProvider.Update(userInfo, sa)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result := convertInternalServiceAccountToExternal(updatedSA)
		result.Group = newGroup
		return result, nil
	}
}

// DeleteEndpoint deletes the service account for the given project
func DeleteEndpoint(serviceAccountProvider provider.ServiceAccountProvider, projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		// check if project exist
		if _, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// check if service account exist before deleting it
		if _, err := serviceAccountProvider.Get(userInfo, req.ServiceAccountID, nil); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := serviceAccountProvider.Delete(userInfo, req.ServiceAccountID); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
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

func decodeServiceAccountIDReq(c context.Context, r *http.Request) (serviceAccountIDReq, error) {
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
