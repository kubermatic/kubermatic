package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func deleteMemberFromProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		req, ok := request.(DeleteUserFromProjectReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		if len(req.UserID) == 0 {
			return nil, k8cerrors.NewBadRequest("the user ID cannot be empty")
		}

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		user, err := userProvider.UserByID(req.UserID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		memberList, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{MemberEmail: user.Spec.Email})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
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
			return nil, kubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func editMemberOfProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		req, ok := request.(EditUserInProjectReq)
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
			return nil, kubernetesErrorToHTTPError(err)
		}
		memberToUpdate, err := userProvider.UserByEmail(currentMemberFromRequest.Email)
		if err != nil && err == provider.ErrNotFound {
			return nil, k8cerrors.NewBadRequest("cannot add the user = %s to the project %s because the user doesn't exist.", currentMemberFromRequest.Email, projectFromRequest.ID)
		} else if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		memberList, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{MemberEmail: currentMemberFromRequest.Email})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
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
			return nil, kubernetesErrorToHTTPError(err)
		}

		externalUser := convertInternalUserToExternal(memberToUpdate, updatedMemberBinding)
		externalUser = filterExternalUser(externalUser, project.Name)
		return externalUser, nil
	}
}

func listMembersOfProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		req, ok := request.(GetProjectRq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}

		if len(req.ProjectID) == 0 {
			return nil, k8cerrors.NewBadRequest("the name of the project cannot be empty")
		}

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		membersOfProjectBindings, err := memberProvider.List(userInfo, project, nil)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		externalUsers := []*apiv1.User{}
		for _, memberOfProjectBinding := range membersOfProjectBindings {
			user, err := userProvider.UserByEmail(memberOfProjectBinding.Spec.UserEmail)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
			externalUser := convertInternalUserToExternal(user, memberOfProjectBinding)
			externalUser = filterExternalUser(externalUser, project.Name)
			externalUsers = append(externalUsers, externalUser)
		}

		return externalUsers, nil
	}
}

func addUserToProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddUserToProjectReq)
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)

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
			return nil, kubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, projectFromRequest.ID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		memberList, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{MemberEmail: userToInvite.Spec.Email})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if len(memberList) > 0 {
			return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot add the user = %s to the project %s because user is already in the project", req.Body.Email, req.ProjectID))
		}

		generatedGroupName := rbac.GenerateActualGroupNameFor(project.Name, projectFromRequest.GroupPrefix)
		generatedBinding, err := memberProvider.Create(userInfo, project, userToInvite.Spec.Email, generatedGroupName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		externalUser := convertInternalUserToExternal(userToInvite, generatedBinding)
		externalUser = filterExternalUser(externalUser, project.Name)
		return externalUser, nil
	}
}

func getCurrentUserEndpoint(users provider.UserProvider, memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authenticatedUser := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)

		bindings, err := memberMapper.MappingsFor(authenticatedUser.Spec.Email)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return convertInternalUserToExternal(authenticatedUser, bindings...), nil
	}
}

func convertInternalUserToExternal(internalUser *kubermaticapiv1.User, bindings ...*kubermaticapiv1.UserProjectBinding) *apiv1.User {
	apiUser := &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalUser.Name,
			Name:              internalUser.Spec.Name,
			CreationTimestamp: internalUser.CreationTimestamp.Time,
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

func (r Routing) userSaverMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			cAPIUser := ctx.Value(apiUserContextKey)
			if cAPIUser == nil {
				return nil, errors.New("no user in context found")
			}
			apiUser := cAPIUser.(apiv1.LegacyUser)

			user, err := r.userProvider.UserByEmail(apiUser.Email)
			if err != nil {
				if err != provider.ErrNotFound {
					return nil, kubernetesErrorToHTTPError(err)
				}
				// handling ErrNotFound
				user, err = r.userProvider.CreateUser(apiUser.ID, apiUser.Name, apiUser.Email)
				if err != nil {
					if !kerrors.IsAlreadyExists(err) {
						return nil, kubernetesErrorToHTTPError(err)
					}
					if user, err = r.userProvider.UserByEmail(apiUser.Email); err != nil {
						return nil, kubernetesErrorToHTTPError(err)
					}
				}
			}
			return next(context.WithValue(ctx, userCRContextKey, user), request)
		}
	}
}

func (r Routing) userInfoMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			user, ok := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
			if !ok {
				return nil, k8cerrors.New(http.StatusInternalServerError, "unable to get authenticated user object")
			}
			prjIDGetter, ok := request.(ProjectIDGetter)
			if !ok {
				return nil, k8cerrors.NewBadRequest("you can only use userInfoMiddleware for endpoints that interact with project")
			}
			projectID := prjIDGetter.GetProjectID()

			uInfo, err := r.createUserInfo(user, projectID)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}

			return next(context.WithValue(ctx, userInfoContextKey, uInfo), request)
		}
	}
}

// userInfoMiddlewareUnauthorized tries to build userInfo for not authenticated (token) user
// instead it reads the user_id from the request and finds the associated user in the database
func (r Routing) userInfoMiddlewareUnauthorized() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			userIDGetter, ok := request.(UserIDGetter)
			if !ok {
				return nil, k8cerrors.NewBadRequest("you can only use userInfoMiddlewareUnauthorized for endpoints that accepts user ID")
			}
			prjIDGetter, ok := request.(ProjectIDGetter)
			if !ok {
				return nil, k8cerrors.NewBadRequest("you can only use userInfoMiddlewareUnauthorized for endpoints that accepts project ID")
			}
			userID := userIDGetter.GetUserID()
			projectID := prjIDGetter.GetProjectID()
			user, err := r.userProvider.UserByID(userID)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}

			uInfo, err := r.createUserInfo(user, projectID)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
			return next(context.WithValue(ctx, userInfoContextKey, uInfo), request)
		}
	}
}

func (r Routing) createUserInfo(user *kubermaticapiv1.User, projectID string) (*provider.UserInfo, error) {
	var group string
	{
		var err error
		group, err = r.userProjectMapper.MapUserToGroup(user.Spec.Email, projectID)
		if err != nil {
			return nil, err
		}
	}

	return &provider.UserInfo{Email: user.Spec.Email, Group: group}, nil
}

// IsAdmin tells if the user has the admin role
func IsAdmin(u apiv1.LegacyUser) bool {
	_, ok := u.Roles[AdminRoleKey]
	return ok
}

// AddUserToProjectReq defines HTTP request for addUserToProject
// swagger:parameters addUserToProject
type AddUserToProjectReq struct {
	ProjectReq
	// in: body
	Body apiv1.User
}

// Validate validates AddUserToProjectReq request
func (r AddUserToProjectReq) Validate(authenticatesUserInfo *provider.UserInfo) error {
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

func decodeAddUserToProject(c context.Context, r *http.Request) (interface{}, error) {
	var req AddUserToProjectReq

	prjReq, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// UserIDReq represents a request that contains userID in the path
type UserIDReq struct {
	// in: path
	UserID string `json:"user_id"`
}

func decodeUserIDReq(c context.Context, r *http.Request) (UserIDReq, error) {
	var req UserIDReq

	userID, ok := mux.Vars(r)["user_id"]
	if !ok {
		return req, fmt.Errorf("'user_id' parameter is required")
	}
	req.UserID = userID

	return req, nil
}

// EditUserInProjectReq defines HTTP request for editUserInProject
// swagger:parameters editUserInProject
type EditUserInProjectReq struct {
	AddUserToProjectReq
	UserIDReq
}

// Validate validates EditUserToProject request
func (r EditUserInProjectReq) Validate(authenticatesUserInfo *provider.UserInfo) error {
	err := r.AddUserToProjectReq.Validate(authenticatesUserInfo)
	if err != nil {
		return err
	}
	if r.UserID != r.Body.ID {
		return k8cerrors.NewBadRequest(fmt.Sprintf("userID mismatch, you requested to update user = %s but body contains user = %s", r.UserID, r.Body.ID))
	}
	return nil
}

func decodeEditUserToProject(c context.Context, r *http.Request) (interface{}, error) {
	var req EditUserInProjectReq

	prjReq, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(ProjectReq)

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

// DeleteUserFromProjectReq defines HTTP request for deleteUserFromProject
// swagger:parameters deleteUserFromProject
type DeleteUserFromProjectReq struct {
	ProjectReq
	UserIDReq
}

func decodeDeleteUserFromProject(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteUserFromProjectReq

	prjReq, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = prjReq.(ProjectReq)

	userIDReq, err := decodeUserIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.UserID = userIDReq.UserID

	return req, nil
}
