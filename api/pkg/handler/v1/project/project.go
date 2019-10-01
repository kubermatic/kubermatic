package project

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// CreateEndpoint defines an HTTP endpoint that creates a new project in the system
func CreateEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectRq, ok := request.(projectReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(projectRq.Body.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		user := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)
		kubermaticProject, err := projectProvider.New(user, projectRq.Body.Name, projectRq.Body.Labels)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiv1.Project{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                kubermaticProject.Name,
				Name:              kubermaticProject.Spec.Name,
				CreationTimestamp: apiv1.NewTime(kubermaticProject.CreationTimestamp.Time),
			},
			Status: kubermaticProject.Status.Phase,
			Labels: kubermaticProject.Labels,
			Owners: []apiv1.User{
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: user.Spec.Name,
					},
					Email: user.Spec.Email,
				},
			},
		}, nil
	}
}

// ListEndpoint defines an HTTP endpoint for listing projects
func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, memberMapper provider.ProjectMemberMapper, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)
		projects := []*apiv1.Project{}

		userMappings, err := memberMapper.MappingsFor(user.Spec.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var errorList []string
		for _, mapping := range userMappings {
			userInfo := &provider.UserInfo{Email: mapping.Spec.UserEmail, Group: mapping.Spec.Group}
			projectInternal, err := projectProvider.Get(userInfo, mapping.Spec.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
			if err != nil {
				// Request came from the specified user. Instead `Not found` error status the `Forbidden` is returned.
				// Next request with privileged user checks if the project doesn't exist or some other error occurred.
				if !isStatus(err, http.StatusForbidden) {
					errorList = append(errorList, err.Error())
					continue
				}
				_, errGetUnsecured := privilegedProjectProvider.GetUnsecured(mapping.Spec.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
				if !isStatus(errGetUnsecured, http.StatusNotFound) {
					// store original error
					errorList = append(errorList, err.Error())
				}
				continue
			}

			projectOwners, err := getOwnersForProject(userInfo, projectInternal, memberProvider, userProvider)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			projects = append(projects, convertInternalProjectToExternal(projectInternal, projectOwners))

		}

		if len(errorList) > 0 {
			return nil, errors.NewWithDetails(http.StatusInternalServerError, "failed to get some projects, please examine details field for more info", errorList)
		}
		return projects, nil
	}
}

func isStatus(err error, status int32) bool {
	if kubernetesError, ok := err.(*kerrors.StatusError); ok {
		if status == kubernetesError.Status().Code {
			return true
		}
	}
	return false
}

// DeleteEndpoint defines an HTTP endpoint for deleting a project
func DeleteEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		err := projectProvider.Delete(userInfo, req.ProjectID)
		return nil, common.KubernetesErrorToHTTPError(err)
	}
}

// UpdateEndpoint defines an HTTP endpoint that updates an existing project in the system
// in the current implementation only project renaming is supported
func UpdateEndpoint(projectProvider provider.ProjectProvider, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		kubermaticProject, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		kubermaticProject.Spec.Name = req.Body.Name
		kubermaticProject.Spec.Labels = req.Body.Labels
		project, err := projectProvider.Update(userInfo, kubermaticProject)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		projectOwners, err := getOwnersForProject(userInfo, kubermaticProject, memberProvider, userProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalProjectToExternal(project, projectOwners), nil
	}
}

// GeEndpoint defines an HTTP endpoint for getting a project
func GetEndpoint(projectProvider provider.ProjectProvider, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		kubermaticProject, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		projectOwners, err := getOwnersForProject(userInfo, kubermaticProject, memberProvider, userProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalProjectToExternal(kubermaticProject, projectOwners), nil
	}
}

func convertInternalProjectToExternal(kubermaticProject *kubermaticapiv1.Project, projectOwners []apiv1.User) *apiv1.Project {
	return &apiv1.Project{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                kubermaticProject.Name,
			Name:              kubermaticProject.Spec.Name,
			CreationTimestamp: apiv1.NewTime(kubermaticProject.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if kubermaticProject.DeletionTimestamp != nil {
					dt := apiv1.NewTime(kubermaticProject.DeletionTimestamp.Time)
					return &dt
				}
				return nil
			}(),
		},
		Labels: kubermaticProject.Labels,
		Status: kubermaticProject.Status.Phase,
		Owners: projectOwners,
	}
}

func getOwnersForProject(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) ([]apiv1.User, error) {
	allProjectMembers, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{SkipPrivilegeVerification: true})
	if err != nil {
		return nil, err
	}
	projectOwners := []apiv1.User{}
	for _, projectMember := range allProjectMembers {
		if rbac.ExtractGroupPrefix(projectMember.Spec.Group) == rbac.OwnerGroupNamePrefix {
			user, err := userProvider.UserByEmail(projectMember.Spec.UserEmail)
			if err != nil {
				continue
			}
			projectOwners = append(projectOwners, apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					Name: user.Spec.Name,
				},
				Email: user.Spec.Email,
			})
		}
	}
	return projectOwners, nil
}

// updateRq defines HTTP request for updateProject
// swagger:parameters updateProject
type updateRq struct {
	common.ProjectReq
	// in: body
	Body apiv1.Project
}

// validate validates updateProject request
func (r updateRq) validate() error {

	if len(r.ProjectID) == 0 {
		return fmt.Errorf("the id of the project cannot be empty")
	}
	if len(r.Body.Name) == 0 {
		return fmt.Errorf("the name of the project cannot be empty")
	}
	return nil
}

// DecodeUpdateRq decodes an HTTP request into updateRq
func DecodeUpdateRq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateRq

	pReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pReq.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// projectReq defines HTTP request for createProject endpoint
// swagger:parameters createProject
type projectReq struct {
	// in:body
	Body struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels,omitempty"`
	}
}

// DecodeCreate decodes an HTTP request into projectReq
func DecodeCreate(c context.Context, r *http.Request) (interface{}, error) {
	var req projectReq

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// deleteRq defines HTTP request for deleteProject endpoint
// swagger:parameters deleteProject
type deleteRq struct {
	common.ProjectReq
}

// DecodeDelete decodes an HTTP request into deleteRq
func DecodeDelete(c context.Context, r *http.Request) (interface{}, error) {
	req, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, nil
	}
	return deleteRq{ProjectReq: req.(common.ProjectReq)}, err
}
