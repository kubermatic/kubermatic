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

package user

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"

	"k8s.io/apimachinery/pkg/api/errors"
)

// DeleteEndpoint deletes the given user/member from the given project
func DeleteEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider, privilegedMemberProvider provider.PrivilegedProjectMemberProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(DeleteReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		if len(req.UserID) == 0 {
			return nil, k8cerrors.NewBadRequest("the user ID cannot be empty")
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		user, err := userProvider.UserByID(req.UserID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		memberList, err := getMemberList(ctx, userInfoGetter, memberProvider, project, user.Spec.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(memberList) == 0 {
			return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot delete the user = %s from the project %s because the user is not a member of the project", user.Spec.Email, req.ProjectID))
		}
		if len(memberList) != 1 {
			return nil, k8cerrors.New(http.StatusInternalServerError, fmt.Sprintf("cannot delete the user user %s from the project, inconsistent state in database", user.Spec.Email))
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		bindingForRequestedMember := memberList[0]
		if strings.EqualFold(bindingForRequestedMember.Spec.UserEmail, userInfo.Email) {
			return nil, k8cerrors.New(http.StatusForbidden, "you cannot delete yourself from the project")
		}

		if err = deleteBinding(ctx, userInfoGetter, memberProvider, privilegedMemberProvider, req.ProjectID, bindingForRequestedMember.Name); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func deleteBinding(ctx context.Context, userInfoGetter provider.UserInfoGetter, memberProvider provider.ProjectMemberProvider, privilegedMemberProvider provider.PrivilegedProjectMemberProvider, projectID, bindingID string) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		return privilegedMemberProvider.DeleteUnsecured(bindingID)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return err
	}
	return memberProvider.Delete(userInfo, bindingID)
}

func getMemberList(ctx context.Context, userInfoGetter provider.UserInfoGetter, memberProvider provider.ProjectMemberProvider, project *kubermaticapiv1.Project, userEmail string) ([]*kubermaticapiv1.UserProjectBinding, error) {
	skipPrivilegeVerification := true

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if !userInfo.IsAdmin {
		userInfo, err = userInfoGetter(ctx, project.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		skipPrivilegeVerification = false
	}

	options := &provider.ProjectMemberListOptions{SkipPrivilegeVerification: skipPrivilegeVerification}
	if userEmail != "" {
		options = &provider.ProjectMemberListOptions{MemberEmail: userEmail, SkipPrivilegeVerification: skipPrivilegeVerification}
	}

	return memberProvider.List(userInfo, project, options)

}

// EditEndpoint changes the group the given user/member belongs in the given project
func EditEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider, privilegedMemberProvider provider.PrivilegedProjectMemberProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(EditReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(userInfo)
		if err != nil {
			return nil, err
		}
		currentMemberFromRequest := req.Body
		projectFromRequest := currentMemberFromRequest.Projects[0]

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		memberToUpdate, err := userProvider.UserByEmail(currentMemberFromRequest.Email)
		if err != nil && err == provider.ErrNotFound {
			return nil, k8cerrors.NewBadRequest("cannot add the user = %s to the project %s because the user doesn't exist.", currentMemberFromRequest.Email, projectFromRequest.ID)
		} else if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		memberList, err := getMemberList(ctx, userInfoGetter, memberProvider, project, currentMemberFromRequest.Email)
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
		updatedMemberBinding, err := updateBinding(ctx, userInfoGetter, memberProvider, privilegedMemberProvider, req.ProjectID, currentMemberBinding)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalUser := convertInternalUserToExternal(memberToUpdate, false, updatedMemberBinding)
		externalUser = filterExternalUser(externalUser, project.Name)
		return externalUser, nil
	}
}

func updateBinding(ctx context.Context, userInfoGetter provider.UserInfoGetter, memberProvider provider.ProjectMemberProvider, privilegedMemberProvider provider.PrivilegedProjectMemberProvider, projectID string, binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedMemberProvider.UpdateUnsecured(binding)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return memberProvider.Update(userInfo, binding)
}

// ListEndpoint returns user/members of the given project
func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, k8cerrors.NewBadRequest("the name of the project cannot be empty")
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		membersOfProjectBindings, err := getMemberList(ctx, userInfoGetter, memberProvider, project, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalUsers := []*apiv1.User{}
		for _, memberOfProjectBinding := range membersOfProjectBindings {
			user, err := userProvider.UserByEmail(memberOfProjectBinding.Spec.UserEmail)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			externalUser := convertInternalUserToExternal(user, false, memberOfProjectBinding)
			externalUser = filterExternalUser(externalUser, project.Name)
			externalUsers = append(externalUsers, externalUser)
		}

		return externalUsers, nil
	}
}

// AddEndpoint adds the given user to the given group within the given project
func AddEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userProvider provider.UserProvider, memberProvider provider.ProjectMemberProvider, privilegedMemberProvider provider.PrivilegedProjectMemberProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddReq)
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(userInfo)
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
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		memberList, err := getMemberList(ctx, userInfoGetter, memberProvider, project, userToInvite.Spec.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(memberList) > 0 {
			return nil, k8cerrors.New(http.StatusBadRequest, fmt.Sprintf("cannot add the user = %s to the project %s because user is already in the project", req.Body.Email, req.ProjectID))
		}

		generatedGroupName := rbac.GenerateActualGroupNameFor(project.Name, projectFromRequest.GroupPrefix)
		generatedBinding, err := createBinding(ctx, userInfoGetter, memberProvider, privilegedMemberProvider, project, userToInvite.Spec.Email, generatedGroupName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalUser := convertInternalUserToExternal(userToInvite, false, generatedBinding)
		externalUser = filterExternalUser(externalUser, project.Name)
		return externalUser, nil
	}
}

// LogoutEndpoint
func LogoutEndpoint(userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authenticatedUser := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)

		t := ctx.Value(middleware.RawTokenContextKey)
		token, ok := t.(string)
		if !ok || token == "" {
			return nil, k8cerrors.NewNotAuthorized()
		}
		e := ctx.Value(middleware.TokenExpiryContextKey)
		expiry, ok := e.(apiv1.Time)
		if !ok {
			return nil, k8cerrors.NewNotAuthorized()
		}
		return nil, userProvider.AddUserTokenToBlacklist(authenticatedUser, token, expiry)
	}
}

func createBinding(ctx context.Context, userInfoGetter provider.UserInfoGetter, memberProvider provider.ProjectMemberProvider, privilegedMemberProvider provider.PrivilegedProjectMemberProvider, project *kubermaticapiv1.Project, email, group string) (*kubermaticapiv1.UserProjectBinding, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedMemberProvider.CreateUnsecured(project, email, group)
	}

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return memberProvider.Create(userInfo, project, email, group)
}

// GetEndpoint returns info about the current user
func GetEndpoint(memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authenticatedUser := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)

		bindings, err := memberMapper.MappingsFor(authenticatedUser.Spec.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalUserToExternal(authenticatedUser, true, bindings...), nil
	}
}

// GetSettingsEndpoint returns settings of the current user
func GetSettingsEndpoint(memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authenticatedUser := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)
		return authenticatedUser.Spec.Settings, nil
	}
}

// PatchSettingsEndpoint patches settings of the current user
func PatchSettingsEndpoint(userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchSettingsReq)
		existingUser := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)

		existingSettings := existingUser.Spec.Settings
		if existingSettings == nil {
			existingSettings = &kubermaticapiv1.UserSettings{}
		}

		existingSettingsJSON, err := json.Marshal(existingSettings)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot decode existing user settings: %v", err))
		}

		patchedSettingsJSON, err := jsonpatch.MergePatch(existingSettingsJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot patch user settings: %v", err))
		}

		var patchedSettings *kubermaticapiv1.UserSettings
		err = json.Unmarshal(patchedSettingsJSON, &patchedSettings)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot decode patched user settings: %v", err))
		}

		existingUser.Spec.Settings = patchedSettings
		updatedUser, err := userProvider.UpdateUser(existingUser)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return updatedUser.Spec.Settings, nil
	}
}

func convertInternalUserToExternal(internalUser *kubermaticapiv1.User, includeSettings bool, bindings ...*kubermaticapiv1.UserProjectBinding) *apiv1.User {
	apiUser := &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalUser.Name,
			Name:              internalUser.Spec.Name,
			CreationTimestamp: apiv1.NewTime(internalUser.CreationTimestamp.Time),
		},
		Email:   internalUser.Spec.Email,
		IsAdmin: internalUser.Spec.IsAdmin,
	}

	if includeSettings {
		apiUser.Settings = internalUser.Spec.Settings
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

// PatchSettingsReq defines HTTP request for patchCurrentUserSettings
// swagger:parameters patchCurrentUserSettings
type PatchSettingsReq struct {
	// in: body
	Patch json.RawMessage
}

// DecodePatchSettingsReq  decodes an HTTP request into PatchSettingsReq
func DecodePatchSettingsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PatchSettingsReq
	var err error

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}
