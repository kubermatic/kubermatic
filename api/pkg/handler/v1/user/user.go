package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// DeleteEndpoint deletes the given user/member from the given project
func DeleteEndpoint(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		req, ok := request.(DeleteReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		if len(req.UserID) == 0 {
			return nil, k8cerrors.NewBadRequest("the user ID cannot be empty")
		}

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		user, err := userProvider.UserByID(req.UserID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		memberList, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{MemberEmail: user.Spec.Email})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(memberList) == 0 {
			return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot delete the user = %s from the project %s because the user is not a member of the project", user.Spec.Email, req.ProjectID))
		}
		if len(memberList) != 1 {
			return nil, k8cerrors.New(http.StatusInternalServerError, fmt.Sprintf("cannot delete the user user %s from the project, inconsistent state in database", user.Spec.Email))
		}

		bindingForRequestedMember := memberList[0]
		if strings.EqualFold(bindingForRequestedMember.Spec.UserEmail, userInfo.Email) {
			return nil, k8cerrors.New(http.StatusForbidden, "you cannot delete yourself from the project")
		}

		err = memberProvider.Delete(userInfo, bindingForRequestedMember.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// EditEndpoint changes the group the given user/member belongs in the given project
func EditEndpoint(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		req, ok := request.(EditReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}

		err := req.Validate(userInfo)
		if err != nil {
			return nil, err
		}
		currentMemberFromRequest := req.Body
		projectFromRequest := currentMemberFromRequest.Projects[0]

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		memberToUpdate, err := userProvider.UserByEmail(currentMemberFromRequest.Email)
		if err != nil && err == provider.ErrNotFound {
			return nil, k8cerrors.NewBadRequest("cannot add the user = %s to the project %s because the user doesn't exist.", currentMemberFromRequest.Email, projectFromRequest.ID)
		} else if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		memberList, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{MemberEmail: currentMemberFromRequest.Email})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(memberList) == 0 {
			return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot change the membership of the user = %s for the project %s because the user is not a member of the project", currentMemberFromRequest.Email, req.ProjectID))
		}
		if len(memberList) != 1 {
			return nil, k8cerrors.New(http.StatusInternalServerError, fmt.Sprintf("cannot change the membershp of the user %s, inconsistent state in database", currentMemberFromRequest.Email))
		}

		currentMemberBinding := memberList[0]
		generatedGroupName := rbac.GenerateActualGroupNameFor(project.Name, projectFromRequest.GroupPrefix)
		currentMemberBinding.Spec.Group = generatedGroupName
		updatedMemberBinding, err := memberProvider.Update(userInfo, currentMemberBinding)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalUser := convertInternalUserToExternal(memberToUpdate, updatedMemberBinding)
		externalUser = filterExternalUser(externalUser, project.Name)
		return externalUser, nil
	}
}

// ListEndpoint returns user/members of the given project
func ListEndpoint(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}

		if len(req.ProjectID) == 0 {
			return nil, k8cerrors.NewBadRequest("the name of the project cannot be empty")
		}

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		membersOfProjectBindings, err := memberProvider.List(userInfo, project, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalUsers := []*apiv1.User{}
		for _, memberOfProjectBinding := range membersOfProjectBindings {
			user, err := userProvider.UserByEmail(memberOfProjectBinding.Spec.UserEmail)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			externalUser := convertInternalUserToExternal(user, memberOfProjectBinding)
			externalUser = filterExternalUser(externalUser, project.Name)
			externalUsers = append(externalUsers, externalUser)
		}

		return externalUsers, nil
	}
}

// AddEndpoint adds the given user to the given group within the given project
func AddEndpoint(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		err := req.Validate(userInfo)
		if err != nil {
			return nil, err
		}
		apiUserFromRequest := req.Body
		projectFromRequest := apiUserFromRequest.Projects[0]

		userToInvite, err := userProvider.UserByEmail(apiUserFromRequest.Email)
		if err != nil && err == provider.ErrNotFound {
			return nil, k8cerrors.NewBadRequest("cannot add the user = %s to the project %s because the user doesn't exist.", apiUserFromRequest.Email, projectFromRequest.ID)
		} else if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, projectFromRequest.ID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		memberList, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{MemberEmail: userToInvite.Spec.Email})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(memberList) > 0 {
			return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot add the user = %s to the project %s because user is already in the project", req.Body.Email, req.ProjectID))
		}

		generatedGroupName := rbac.GenerateActualGroupNameFor(project.Name, projectFromRequest.GroupPrefix)
		generatedBinding, err := memberProvider.Create(userInfo, project, userToInvite.Spec.Email, generatedGroupName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalUser := convertInternalUserToExternal(userToInvite, generatedBinding)
		externalUser = filterExternalUser(externalUser, project.Name)
		return externalUser, nil
	}
}

// GetEndpoint returns info about the current user
func GetEndpoint(memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authenticatedUser := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)

		bindings, err := memberMapper.MappingsFor(authenticatedUser.Spec.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalUserToExternal(authenticatedUser, bindings...), nil
	}
}

func convertInternalUserToExternal(internalUser *kubermaticapiv1.User, bindings ...*kubermaticapiv1.UserProjectBinding) *apiv1.User {
	apiUser := &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalUser.Name,
			Name:              internalUser.Spec.Name,
			CreationTimestamp: apiv1.NewTime(internalUser.CreationTimestamp.Time),
		},
		Email: internalUser.Spec.Email,
	}
	for _, pg := range internalUser.Spec.Projects {
		groupPrefix := rbac.ExtractGroupPrefix(pg.Group)
		apiUser.Projects = append(apiUser.Projects, apiv1.ProjectGroup{ID: pg.Name, GroupPrefix: groupPrefix})
	}

	for _, binding := range bindings {
		bindingAlreadyExists := false
		for _, pg := range apiUser.Projects {
			if pg.ID == binding.Spec.ProjectID && pg.GroupPrefix == binding.Spec.Group {
				bindingAlreadyExists = true
				break
			}
		}
		if !bindingAlreadyExists {
			groupPrefix := rbac.ExtractGroupPrefix(binding.Spec.Group)
			apiUser.Projects = append(apiUser.Projects, apiv1.ProjectGroup{ID: binding.Spec.ProjectID, GroupPrefix: groupPrefix})
		}
	}

	return apiUser
}

func filterExternalUser(externalUser *apiv1.User, projectID string) *apiv1.User {
	// remove all projects except requested one
	// in the future user resources will not contain projects bindings
	for _, pg := range externalUser.Projects {
		if pg.ID == projectID {
			externalUser.Projects = []apiv1.ProjectGroup{pg}
			break
		}
	}
	return externalUser
}

// AddReq defines HTTP request for addUserToProject
// swagger:parameters addUserToProject
type AddReq struct {
	common.ProjectReq
	// in: body
	Body apiv1.User
}

// Validate validates AddReq request
func (r AddReq) Validate(authenticatesUserInfo *provider.UserInfo) error {
	if len(r.ProjectID) == 0 {
		return k8cerrors.NewBadRequest("the name of the project cannot be empty")
	}
	apiUserFromRequest := r.Body
	if len(apiUserFromRequest.Email) == 0 {
		return k8cerrors.NewBadRequest("the email address cannot be empty")
	}
	if _, err := mail.ParseAddress(apiUserFromRequest.Email); err != nil {
		return k8cerrors.NewBadRequest("incorrect email format: %v", err)
	}
	if len(r.Body.Projects) != 1 {
		return k8cerrors.NewBadRequest("expected exactly one entry in \"Projects\" field, but received %d", len(apiUserFromRequest.Projects))
	}
	projectFromRequest := apiUserFromRequest.Projects[0]
	if len(projectFromRequest.ID) == 0 || len(projectFromRequest.GroupPrefix) == 0 {
		return k8cerrors.NewBadRequest("both the project name and the group name fields are required")
	}
	if projectFromRequest.ID != r.ProjectID {
		return k8cerrors.New(http.StatusForbidden, fmt.Sprintf("you can only assign the user to %s project", r.ProjectID))
	}
	if strings.EqualFold(apiUserFromRequest.Email, authenticatesUserInfo.Email) {
		return k8cerrors.New(http.StatusForbidden, "you cannot assign yourself to a different group")
	}
	if projectFromRequest.GroupPrefix == rbac.OwnerGroupNamePrefix {
		return k8cerrors.New(http.StatusForbidden, fmt.Sprintf("the given user cannot be assigned to %s group", projectFromRequest.GroupPrefix))
	}
	isRequestedGroupPrefixValid := false
	for _, existingGroupPrefix := range rbac.AllGroupsPrefixes {
		if existingGroupPrefix == projectFromRequest.GroupPrefix {
			isRequestedGroupPrefixValid = true
			break
		}
	}
	if !isRequestedGroupPrefixValid {
		return k8cerrors.NewBadRequest("invalid group name %s", projectFromRequest.GroupPrefix)
	}
	return nil
}

// DecodeAddReq  decodes an HTTP request into AddReq
func DecodeAddReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AddReq

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

// IDReq represents a request that contains userID in the path
type IDReq struct {
	// in: path
	UserID string `json:"user_id"`
}

func decodeUserIDReq(c context.Context, r *http.Request) (IDReq, error) {
	var req IDReq

	userID, ok := mux.Vars(r)["user_id"]
	if !ok {
		return req, fmt.Errorf("'user_id' parameter is required")
	}
	req.UserID = userID

	return req, nil
}

// EditReq defines HTTP request for editUserInProject
// swagger:parameters editUserInProject
type EditReq struct {
	AddReq
	IDReq
}

// Validate validates EditUserToProject request
func (r EditReq) Validate(authenticatesUserInfo *provider.UserInfo) error {
	err := r.AddReq.Validate(authenticatesUserInfo)
	if err != nil {
		return err
	}
	if r.UserID != r.Body.ID {
		return k8cerrors.NewBadRequest(fmt.Sprintf("userID mismatch, you requested to update user = %s but body contains user = %s", r.UserID, r.Body.ID))
	}
	return nil
}

// DecodeEditReq  decodes an HTTP request into EditReq
func DecodeEditReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EditReq

	prjReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	userIDReq, err := decodeUserIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.UserID = userIDReq.UserID

	return req, nil
}

// DeleteReq defines HTTP request for deleteUserFromProject
// swagger:parameters deleteUserFromProject
type DeleteReq struct {
	common.ProjectReq
	IDReq
}

// DecodeDeleteReq  decodes an HTTP request into DeleteReq
func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteReq

	prjReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = prjReq.(common.ProjectReq)

	userIDReq, err := decodeUserIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.UserID = userIDReq.UserID

	return req, nil
}
