package serviceaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	"k8s.io/api/core/v1"
)

// CreateTokenEndpoint adds the given token to the service account
func CreateTokenEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, serviceAccountTokenProvider provider.ServiceAccountTokenProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addTokenReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		sa, err := serviceAccountProvider.Get(userInfo, req.ServiceAccountID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		secret, err := serviceAccountTokenProvider.Create(userInfo, sa, req.Body.Name, project.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalTokenToExternal(secret), nil
	}
}

// addTokenReq defines HTTP request for addTokenToServiceAccount
// swagger:parameters addTokenToServiceAccount
type addTokenReq struct {
	common.ProjectReq
	idReq
	// in: body
	Body apiv1.ServiceAccountToken
}

// Validate validates addTokenReq request
func (r addTokenReq) Validate() error {
	if len(r.Body.Name) == 0 || len(r.ProjectID) == 0 || len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("the name, service account ID and project ID cannot be empty")
	}

	return nil
}

// DecodeAddReq  decodes an HTTP request into addReq
func DecodeAddTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req addTokenReq

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

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func convertInternalTokenToExternal(internal *v1.Secret) *apiv1.ServiceAccountToken {
	return &apiv1.ServiceAccountToken{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internal.Name,
			Name:              internal.Name,
			CreationTimestamp: apiv1.NewTime(internal.CreationTimestamp.Time),
		},
		Token: string(internal.Data["token"]),
	}
}
