package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func listUsersFromProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		kubermaticProject, err := getKubermaticProject(ctx, projectProvider, request)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		users, err := userProvider.ListByProject(kubermaticProject.Name)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		externalUsers := []*apiv1.NewUser{}
		for _, user := range users {
			externalUser := convertInternalUserToExternal(user)
			for _, pg := range externalUser.Projects {
				if pg.ID == kubermaticProject.Name {
					externalUser.Projects = []apiv1.ProjectGroup{pg}
					break
				}
			}
			externalUsers = append(externalUsers, externalUser)
		}

		return externalUsers, nil
	}
}

func addUserToProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddUserToProjectReq)
		authenticatedUser := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		apiUserFromRequest := req.Body
		if len(apiUserFromRequest.Email) == 0 {
			return nil, k8cerrors.NewBadRequest("the email address cannot be empty")
		}
		if len(req.Body.Projects) != 1 {
			return nil, k8cerrors.NewBadRequest("expected exactly one entry in \"Projects\" field, but received %d", len(apiUserFromRequest.Projects))
		}
		projectFromRequest := apiUserFromRequest.Projects[0]
		if len(projectFromRequest.ID) == 0 || len(projectFromRequest.GroupPrefix) == 0 {
			return nil, k8cerrors.NewBadRequest("both the project name and the group name fields are required")
		}
		if projectFromRequest.ID != req.ProjectID {
			return nil, k8cerrors.New(http.StatusForbidden, fmt.Sprintf("you can only assign the user to %s project", req.ProjectID))
		}
		if apiUserFromRequest.Email == authenticatedUser.Spec.Email {
			return nil, k8cerrors.New(http.StatusForbidden, "you cannot assign yourself to a different group")
		}
		if projectFromRequest.GroupPrefix == rbac.OwnerGroupNamePrefix {
			return nil, k8cerrors.New(http.StatusForbidden, fmt.Sprintf("the given user cannot be assigned to %s group", projectFromRequest.GroupPrefix))
		}

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

		for _, project := range userToInvite.Spec.Projects {
			if project.Name == req.ProjectID {
				return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot add the user = %s to the project %s because user is already in the project", req.Body.Email, req.ProjectID))
			}
		}

		authUserGroupName, err := authenticatedUser.GroupForProject(project.Name)
		if err != nil {
			return nil, k8cerrors.New(http.StatusForbidden, err.Error())
		}
		authUserGroupPrefix := rbac.ExtractGroupPrefix(authUserGroupName)
		if authUserGroupPrefix != rbac.OwnerGroupNamePrefix {
			return nil, k8cerrors.New(http.StatusForbidden, "only the owner of the project can invite the other users")
		}
		isRequestedGroupPrefixValid := false
		for _, existingGroupPrefix := range rbac.AllGroupsPrefixes {
			if existingGroupPrefix == projectFromRequest.GroupPrefix {
				isRequestedGroupPrefixValid = true
				break
			}
		}
		if !isRequestedGroupPrefixValid {
			return nil, k8cerrors.NewBadRequest("invalid group name %s", projectFromRequest.GroupPrefix)
		}
		updatedProjectGroups := []kubermaticapiv1.ProjectGroup{}
		for _, existingProjectGroup := range userToInvite.Spec.Projects {
			if existingProjectGroup.Name != projectFromRequest.ID {
				updatedProjectGroups = append(updatedProjectGroups, existingProjectGroup)
			}
		}
		generatedGroupName := rbac.GenerateActualGroupNameFor(project.Name, projectFromRequest.GroupPrefix)

		// Note:
		// since the users are not resources that belong to the project,
		// we use a privileged account to update the user.
		// Even if they were part of the project, that might be not practical
		// since a user might want to invite any user to the project.
		// Thus we would have to generate and maintain roles for N project and N users.
		userToInvite.Spec.Projects = append(updatedProjectGroups, kubermaticapiv1.ProjectGroup{Name: project.Name, Group: generatedGroupName})
		if _, err = userProvider.Update(userToInvite); err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return convertInternalUserToExternal(userToInvite), nil
	}
}

func getCurrentUserEndpoint(users provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authenticatedUser := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)

		return convertInternalUserToExternal(authenticatedUser), nil
	}
}

func convertInternalUserToExternal(internalUser *kubermaticapiv1.User) *apiv1.NewUser {
	apiUser := &apiv1.NewUser{
		NewObjectMeta: apiv1.NewObjectMeta{
			ID:                internalUser.Name,
			Name:              internalUser.Spec.Name,
			CreationTimestamp: internalUser.CreationTimestamp.Time,
		},
		Email: internalUser.Spec.Email,
	}
	for _, pg := range internalUser.Spec.Projects {
		apiUser.Projects = append(apiUser.Projects, apiv1.ProjectGroup{ID: pg.Name, GroupPrefix: pg.Group})
	}
	return apiUser
}

func (r Routing) userSaverMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			cAPIUser := ctx.Value(apiUserContextKey)
			if cAPIUser == nil {
				return nil, errors.New("no user in context found")
			}
			apiUser := cAPIUser.(apiv1.User)

			user, err := r.userProvider.UserByEmail(apiUser.Email)
			if err != nil {
				if err == provider.ErrNotFound {
					user, err = r.userProvider.CreateUser(apiUser.ID, apiUser.Name, apiUser.Email)
					if err != nil {
						return nil, kubernetesErrorToHTTPError(err)
					}
				} else {
					return nil, err
				}
			}

			// if for some reason ID and Name of the authenticated user
			// are different than the ones we have in our database update the record.
			if user.Spec.ID != apiUser.ID {
				user.Spec.ID = apiUser.ID
				if user.Spec.Name != apiUser.Name {
					user.Spec.Name = apiUser.Name
				}
				user, err = r.userProvider.Update(user)
				if err != nil {
					return nil, kubernetesErrorToHTTPError(err)
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

			// read group info
			var group string
			{
				// old approach, group was stored in user resource
				group, err = user.GroupForProject(projectID)
				if err != nil {
					// new approach read group info from the mapper
					group, err = r.userProjectMapper.MapUserToGroup(user.Spec.Email, projectID)
					if err != nil {
						// wrapping in k8s error to stay consisted with error messages returned from providers
						return nil, kubernetesErrorToHTTPError(err)
					}
				}
			}

			uInfo := &provider.UserInfo{
				Email: user.Spec.Email,
				Group: group,
			}

			return next(context.WithValue(ctx, userInfoContextKey, uInfo), request)
		}
	}
}

// IsAdmin tells if the user has the admin role
func IsAdmin(u apiv1.User) bool {
	_, ok := u.Roles[AdminRoleKey]
	return ok
}

// AddUserToProjectReq defines HTTP request for addUserToProject
// swagger:parameters addUserToProject
type AddUserToProjectReq struct {
	ProjectReq
	// in: body
	Body apiv1.NewUser
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
