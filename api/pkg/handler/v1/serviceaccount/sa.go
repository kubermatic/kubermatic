package serviceaccount

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
	sa "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// serviceAccountGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for service account.
var serviceAccountGroupsPrefixes = []string{
	rbac.EditorGroupNamePrefix,
	rbac.ViewerGroupNamePrefix,
}

// AddEndpoint adds the given service account to the given project
func AddEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		err := req.Validate()
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

// AddReq defines HTTP request for addServiceAccountToProject
// swagger:parameters addServiceAccountToProject
type AddReq struct {
	common.ProjectReq
	// in: body
	Body apiv1.ServiceAccount
}

// Validate validates AddReq request
func (r AddReq) Validate() error {
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

func convertInternalServiceAccountToExternal(internal *kubermaticapiv1.User) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internal.Name,
			Name:              internal.Spec.Name,
			CreationTimestamp: apiv1.NewTime(internal.CreationTimestamp.Time),
		},
		Group: internal.Labels[sa.ServiceAccountLabelGroup],
	}
}
