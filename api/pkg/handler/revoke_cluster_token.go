package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// RevokeClusterTokenReq is consumed by
// swagger:model RevokeClusterToken
type RevokeClusterToken struct {
	Token string `json:"token"`
}

type revokeClusterTokenReq struct {
	// in: path
	Cluster string
	// in: body
	RevokeClusterToken
}

// RevokeClusterTokenReq
// swagger:response RevokeClusterTokenResp
type RevokeClusterTokenResp struct {
}

func decodeRevokeClusterTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req revokeClusterTokenReq
	req.Cluster = mux.Vars(r)["cluster"]

	var body RevokeClusterToken
	err := json.NewDecoder(r.Body).Decode(&body)
	req.RevokeClusterToken = body
	if err != nil {
		return nil, err
	}

	return req, nil
}

// revokeClusterToken starts a cluster upgrade to a specific version
// swagger:route PUT /api/v1/cluster/{cluster}/revoke-admin-token cluster upgrade revokeClusterToken
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: RevokeClusterTokenReq
func (r Routing) revokeClusterToken() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(revokeClusterToken(r.provider))),
		decodeRevokeClusterTokenReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func revokeClusterToken(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req, ok := request.(revokeClusterTokenReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, RevokeClusterToken{})
		}

		k, err := kp.Cluster(user, req.Cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.Cluster)
			}
			return nil, err
		}

		// Sanity check does the User really has access to the token and can revoke it
		if k.Address.AdminToken == req.RevokeClusterToken.Token {
			return kp.RevokeClusterToken(user, k.Name)
		}
		if k.Status.Phase != v1.RunningClusterStatusPhase {
			return nil, errors.NewBadRequest("Cluster is not running yet!")
		}

		return nil, errors.NewBadRequest(fmt.Sprintf("Wrong AdminToken want: %q, got %q", k.Address.AdminToken, req.RevokeClusterToken.Token))
	}
}
