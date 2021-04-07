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

package project

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// CreateEndpoint defines an HTTP endpoint that creates a new project in the system
func CreateEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, settingsProvider provider.SettingsProvider, memberMapper provider.ProjectMemberMapper, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectRq, ok := request.(projectReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(projectRq.Body.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		user := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)
		if kubernetes.IsProjectServiceAccount(user.Spec.Email) {
			return nil, errors.New(http.StatusForbidden, "the Service Account is not allowed to create a project")
		}

		if kubernetes.IsMainServiceAccount(user.Spec.Email) {
			userOwnerEmail, ok := user.Annotations[kubernetes.ServiceAccountAnnotationOwner]
			if !ok {
				return nil, errors.New(http.StatusInternalServerError, "missing owner annotation for main service account")
			}
			user, err = userProvider.UserByEmail(userOwnerEmail)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		if err := checkProjectRestriction(user, settings); err != nil {
			return nil, err
		}

		if err := validateUserProjectsLimit(user, settings, projectProvider, privilegedProjectProvider, memberMapper, memberProvider, userProvider); err != nil {
			return nil, err
		}

		kubermaticProject, err := projectProvider.New(user, projectRq.Body.Name, projectRq.Body.Labels)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		owners := []apiv1.User{
			{
				ObjectMeta: apiv1.ObjectMeta{
					Name: user.Spec.Name,
				},
				Email: user.Spec.Email,
			},
		}

		return common.ConvertInternalProjectToExternal(kubermaticProject, owners, 0), nil
	}
}

func checkProjectRestriction(user *kubermaticapiv1.User, settings *kubermaticapiv1.KubermaticSetting) error {
	if user.Spec.IsAdmin {
		return nil
	}
	if settings.Spec.RestrictProjectCreation {
		return errors.New(http.StatusForbidden, "project creation is restricted")
	}
	return nil
}

func validateUserProjectsLimit(user *kubermaticapiv1.User, settings *kubermaticapiv1.KubermaticSetting, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, memberMapper provider.ProjectMemberMapper, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) error {
	if user.Spec.IsAdmin {
		return nil
	}
	if settings.Spec.UserProjectsLimit <= 0 {
		return nil
	}

	userMappings, err := memberMapper.MappingsFor(user.Spec.Email)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	var errorList []string
	var projectsCounter int64
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
		// get only owned projects
		projectOwners, err := common.GetOwnersForProject(userInfo, projectInternal, memberProvider, userProvider)
		if err != nil {
			return common.KubernetesErrorToHTTPError(err)
		}
		for _, owner := range projectOwners {
			if strings.EqualFold(owner.Email, user.Spec.Email) {
				projectsCounter++
			}
		}
	}
	if len(errorList) > 0 {
		return errors.NewWithDetails(http.StatusInternalServerError, "failed to get some projects, please examine details field for more info", errorList)
	}

	if projectsCounter >= settings.Spec.UserProjectsLimit {
		return errors.New(http.StatusForbidden, "reached maximum number of projects")
	}
	return nil
}

// ListEndpoint defines an HTTP endpoint for listing projects
func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, memberMapper provider.ProjectMemberMapper, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider, clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(ListReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if req.DisplayAll && userInfo.IsAdmin {
			return getAllProjectsForAdmin(userInfo, projectProvider, memberProvider, userProvider, clusterProviderGetter, seedsGetter)
		}
		projects := []*apiv1.Project{}
		userMappings, err := memberMapper.MappingsFor(userInfo.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var errorList []string
		for _, mapping := range userMappings {
			userInfo := &provider.UserInfo{Email: mapping.Spec.UserEmail, Group: mapping.Spec.Group}
			projectInternal, err := projectProvider.Get(userInfo, mapping.Spec.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
			if err != nil {
				if isStatus(err, http.StatusNotFound) {
					continue
				}
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

			projectOwners, err := common.GetOwnersForProject(userInfo, projectInternal, memberProvider, userProvider)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			clustersNumber, err := getNumberOfClustersForProject(clusterProviderGetter, seedsGetter, projectInternal)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			projects = append(projects, common.ConvertInternalProjectToExternal(projectInternal, projectOwners, clustersNumber))

		}

		if len(errorList) > 0 {
			return nil, errors.NewWithDetails(http.StatusInternalServerError, "failed to get some projects, please examine details field for more info", errorList)
		}
		return projects, nil
	}
}

func getAllProjectsForAdmin(userInfo *provider.UserInfo, projectProvider provider.ProjectProvider, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider, clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) ([]*apiv1.Project, error) {
	projects := []*apiv1.Project{}
	projectList, err := projectProvider.List(nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clustersNumbers, err := getNumberOfClusters(clusterProviderGetter, seedsGetter)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	for _, project := range projectList {
		projectOwners, err := common.GetOwnersForProject(userInfo, project, memberProvider, userProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		projects = append(projects, common.ConvertInternalProjectToExternal(project, projectOwners, clustersNumbers[project.Name]))
	}

	return projects, nil
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
func DeleteEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}

		// check if admin user
		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		// allow to delete any project for the admin user
		if adminUserInfo.IsAdmin {
			err := privilegedProjectProvider.DeleteUnsecured(req.ProjectID)
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, deleteProjectByRegularUser(ctx, userInfoGetter, projectProvider, req.ProjectID)
	}
}

func deleteProjectByRegularUser(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, projectID string) error {
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	err = projectProvider.Delete(userInfo, projectID)
	return common.KubernetesErrorToHTTPError(err)
}

// UpdateEndpoint defines an HTTP endpoint that updates an existing project in the system
// in the current implementation only project renaming is supported
func UpdateEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider, userInfoGetter provider.UserInfoGetter, clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		kubermaticProject.Spec.Name = req.Body.Name
		kubermaticProject.Labels = req.Body.Labels

		project, err := updateProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, kubermaticProject)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		projectOwners, err := common.GetOwnersForProject(adminUserInfo, kubermaticProject, memberProvider, userProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		clustersNumber, err := getNumberOfClustersForProject(clusterProviderGetter, seedsGetter, project)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return common.ConvertInternalProjectToExternal(project, projectOwners, clustersNumber), nil
	}
}

func updateProject(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, kubermaticProject *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}

	if adminUserInfo.IsAdmin {
		return privilegedProjectProvider.UpdateUnsecured(kubermaticProject)
	}
	userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
	if err != nil {
		return nil, err
	}
	return projectProvider.Update(userInfo, kubermaticProject)
}

// GeEndpoint defines an HTTP endpoint for getting a project
func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider, userInfoGetter provider.UserInfoGetter, clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		projectOwners, err := common.GetOwnersForProject(adminUserInfo, kubermaticProject, memberProvider, userProvider)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		clustersNumber, err := getNumberOfClustersForProject(clusterProviderGetter, seedsGetter, kubermaticProject)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return common.ConvertInternalProjectToExternal(kubermaticProject, projectOwners, clustersNumber), nil
	}
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

// ListReq defines HTTP request for listProjects endpoint
// swagger:parameters listProjects
type ListReq struct {
	// in: query
	DisplayAll bool `json:"displayAll,omitempty"`
}

func DecodeList(c context.Context, r *http.Request) (interface{}, error) {
	var req ListReq
	var displayAll bool
	var err error

	queryParam := r.URL.Query().Get("displayAll")

	if queryParam != "" {
		displayAll, err = strconv.ParseBool(queryParam)
		if err != nil {
			return nil, fmt.Errorf("wrong query parameter: %v", err)
		}
	}
	req.DisplayAll = displayAll

	return req, nil
}

func getNumberOfClustersForProject(clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter, project *kubermaticapiv1.Project) (int, error) {
	var clustersNumber int
	seeds, err := seedsGetter()
	if err != nil {
		return clustersNumber, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	for datacenter, seed := range seeds {
		clusterProvider, err := clusterProviderGetter(seed)
		if err != nil {
			return clustersNumber, errors.NewNotFound("cluster-provider", datacenter)
		}
		clusters, err := clusterProvider.List(project, nil)
		if err != nil {
			return clustersNumber, err
		}
		clustersNumber += len(clusters.Items)
	}

	return clustersNumber, nil
}

func getNumberOfClusters(clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) (map[string]int, error) {
	clustersNumber := map[string]int{}
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	for datacenter, seed := range seeds {
		clusterProvider, err := clusterProviderGetter(seed)
		if err != nil {
			return nil, errors.NewNotFound("cluster-provider", datacenter)
		}
		clusters, err := clusterProvider.ListAll()
		if err != nil {
			return clustersNumber, err
		}
		for _, cluster := range clusters.Items {
			projectName, ok := cluster.Labels[kubermaticapiv1.ProjectIDLabelKey]
			if ok {
				clustersNumber[projectName]++
			}
		}
	}

	return clustersNumber, nil
}
