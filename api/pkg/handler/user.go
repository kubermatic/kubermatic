package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func addUserToProject(projectProvider provider.ProjectProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddUserToProjectReq)
		authenticatedUser := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		apiUserFromRequest := req.Body
		if len(apiUserFromRequest.Email) == 0 {
			return nil, k8cerrors.NewBadRequest("the email address cannot be empty")
		}
		if len(req.Body.Projects) != 1 {
			return nil, k8cerrors.NewBadRequest("expected exactly one entry in \"Projects\" field, but received %d", len(apiUserFromRequest.Projects))
		}
		projectFromRequest := apiUserFromRequest.Projects[0]
		if len(projectFromRequest.Name) == 0 || len(projectFromRequest.GroupPrefix) == 0 {
			return nil, k8cerrors.NewBadRequest("both the project name and the group name fields are required")
		}
		if projectFromRequest.Name != req.ProjectName {
			return nil, k8cerrors.NewBadRequest("you can only assign the user to %s project", req.ProjectName)
		}
		if apiUserFromRequest.Email == authenticatedUser.Spec.Email {
			return nil, k8cerrors.NewBadRequest("you cannot assign yourself to a different group")
		}
		if projectFromRequest.GroupPrefix == rbac.OwnerGroupNamePrefix {
			return nil, k8cerrors.NewBadRequest("the given user cannot be assigned to % group", projectFromRequest.GroupPrefix)
		}

		userToInvite, err := userProvider.UserByEmail(apiUserFromRequest.Email)
		if err != nil && err == provider.ErrNotFound {
			return nil, k8cerrors.NewBadRequest("cannot add the user = %s to the project %s because the user doesn't exist.", apiUserFromRequest.Email, projectFromRequest.Name)
		} else if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(authenticatedUser, projectFromRequest.Name)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		authUserGroupName, err := authenticatedUser.GroupForProject(project.Name)
		if err != nil {
			return nil, k8cerrors.New(http.StatusForbidden, err.Error())
		}
		authUserGroupPrefix := rbac.ExtractGroupPrefix(authUserGroupName)
		if authUserGroupPrefix != rbac.OwnerGroupNamePrefix {
			return nil, k8cerrors.New(http.StatusForbidden, "only the owner of the project can invite others")
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
		updatedProjectGropus := []kubermaticapiv1.ProjectGroup{}
		for _, existingProjectGroup := range userToInvite.Spec.Projects {
			if existingProjectGroup.Name != projectFromRequest.Name {
				updatedProjectGropus = append(updatedProjectGropus, existingProjectGroup)
			}
		}
		generatedGroupName := rbac.GenerateActualGroupNameFor(project.Name, projectFromRequest.GroupPrefix)

		// Note:
		// since the users are not resources that belong to the project,
		// we use a privileged account to update the user.
		// Even if they were part of the project, that might be not practical
		// since a user might want to invite any user to the project.
		// Thus we would have to generate and maintain roles for N project and N users.
		userToInvite.Spec.Projects = append(updatedProjectGropus, kubermaticapiv1.ProjectGroup{Name: project.Name, Group: generatedGroupName})
		if _, err = userProvider.Update(userToInvite); err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
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
						return nil, err
					}
				} else {
					return nil, err
				}
			}

			return next(context.WithValue(ctx, userCRContextKey, user), request)
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
	// in: path
	ProjectName string `json:"project_id"`
	// in: body
	Body apiv1.NewUser
}

func decodeAddUserToProject(c context.Context, r *http.Request) (interface{}, error) {
	var req AddUserToProjectReq

	projectName := mux.Vars(r)["project_id"]
	if projectName == "" {
		return "", fmt.Errorf("'project_id' parameter is required but was not provided")
	}

	req.ProjectName = projectName
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}
