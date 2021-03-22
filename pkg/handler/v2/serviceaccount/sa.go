/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	rbac.OwnerGroupNamePrefix,
}

// CreateEndpoint creates a new main service account for the user
func CreateEndpoint(serviceAccountProvider provider.ServiceAccountProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		saFromRequest := req.Body

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}

		// check if service account name is already reserved in the project
		existingSAList, err := listSA(userInfo, serviceAccountProvider, &saFromRequest)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingSAList) > 0 {
			return nil, errors.NewAlreadyExists("service account", saFromRequest.Name)
		}

		mainSA, err := serviceAccountProvider.CreateMainServiceAccount(userInfo, saFromRequest.Name, saFromRequest.Group)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalServiceAccountToExternal(mainSA), nil
	}
}

// ListEndpoint returns main service accounts
func ListEndpoint(serviceAccountProvider provider.ServiceAccountProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		resultList := make([]*apiv1.ServiceAccount, 0)
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		existingSAList, err := listSA(userInfo, serviceAccountProvider, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		for _, internalMainServiceAccount := range existingSAList {
			resultList = append(resultList, convertInternalServiceAccountToExternal(internalMainServiceAccount))
		}
		return resultList, nil
	}
}

// UpdateEndpoint changes the service account group and/or name in the given project
func UpdateEndpoint(serviceAccountProvider provider.ServiceAccountProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
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

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}

		sa, err := serviceAccountProvider.GetMainServiceAccount(userInfo, req.ServiceAccountID, nil)
		if err != nil {
			return nil, err
		}

		// update the service account name
		if sa.Spec.Name != saFromRequest.Name {
			// check if service account name is already reserved in the project
			existingSAList, err := listSA(userInfo, serviceAccountProvider, &saFromRequest)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			if len(existingSAList) > 0 {
				return nil, errors.NewAlreadyExists("service account", saFromRequest.Name)
			}
			sa.Spec.Name = saFromRequest.Name
		}

		sa.Labels[serviceaccount.ServiceAccountLabelGroup] = saFromRequest.Group

		updatedSA, err := serviceAccountProvider.UpdateMainServiceAccount(userInfo, sa)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result := convertInternalServiceAccountToExternal(updatedSA)
		return result, nil
	}
}

// DeleteEndpoint deletes the service account
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

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}

		// check if service account exist before deleting it
		sa, err := serviceAccountProvider.GetMainServiceAccount(userInfo, req.ServiceAccountID, nil)
		if err != nil {
			return nil, err
		}

		if err := serviceAccountProvider.DeleteMainServiceAccount(userInfo, sa); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// deleteReq defines HTTP request for deleteMainServiceAccount
// swagger:parameters deleteMainServiceAccount
type deleteReq struct {
	serviceAccountIDReq
}

// Validate validates DeleteEndpoint request
func (r deleteReq) Validate() error {
	if len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("the service account ID cannot be empty")
	}
	return nil
}

// DecodeDeleteeReq  decodes an HTTP request into deleteReq
func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteReq

	saIDReq, err := decodeServiceAccountIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ServiceAccountID = saIDReq.ServiceAccountID

	return req, nil
}

// Validate validates UpdateEndpoint request
func (r updateReq) Validate() error {
	if r.ServiceAccountID != r.Body.ID {
		return fmt.Errorf("service account ID mismatch, you requested to update ServiceAccount = %s but body contains ServiceAccount = %s", r.ServiceAccountID, r.Body.ID)
	}

	for _, existingGroupPrefix := range serviceAccountGroupsPrefixes {
		if existingGroupPrefix == r.Body.Group {
			return nil
		}
	}
	return fmt.Errorf("invalid group name %s", r.Body.Group)
}

// DecodeUpdateReq  decodes an HTTP request into updateReq
func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateReq

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

func decodeServiceAccountIDReq(_ context.Context, r *http.Request) (serviceAccountIDReq, error) {
	var req serviceAccountIDReq

	saID, ok := mux.Vars(r)["serviceaccount_id"]
	if !ok {
		return req, fmt.Errorf("'serviceaccount_id' parameter is required")
	}
	req.ServiceAccountID = saID

	return req, nil
}

// serviceAccountIDReq represents a request that contains main service account ID in the path
type serviceAccountIDReq struct {
	// in: path
	ServiceAccountID string `json:"serviceaccount_id"`
}

// updateReq defines HTTP request for updateMainServiceAccount
// swagger:parameters updateMainServiceAccount
type updateReq struct {
	// in: body
	Body apiv1.ServiceAccount

	serviceAccountIDReq
}

// addReq defines HTTP request for createMainServiceAccount
// swagger:parameters createMainServiceAccount
type addReq struct {
	// in: body
	Body apiv1.ServiceAccount
}

// Validate validates addReq request
func (r addReq) Validate() error {

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

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func listSA(userInfo *provider.UserInfo, serviceAccountProvider provider.ServiceAccountProvider, sa *apiv1.ServiceAccount) ([]*kubermaticapiv1.User, error) {
	var options *provider.ServiceAccountListOptions

	if sa != nil {
		options = &provider.ServiceAccountListOptions{ServiceAccountName: sa.Name}
	}

	return serviceAccountProvider.ListMainServiceAccounts(userInfo, options)
}

func convertInternalServiceAccountToExternal(internal *kubermaticapiv1.User) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internal.Name,
			Name:              internal.Spec.Name,
			CreationTimestamp: apiv1.NewTime(internal.CreationTimestamp.Time),
		},
		Group:  internal.Labels[serviceaccount.ServiceAccountLabelGroup],
		Status: apiv1.ServiceAccountActive,
	}
}
