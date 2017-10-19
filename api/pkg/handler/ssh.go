package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

type createSSHKeyReq struct {
	*v1.UserSSHKey
}

func decodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createSSHKeyReq
	req.UserSSHKey = &v1.UserSSHKey{}
	// Decode
	if err := json.NewDecoder(r.Body).Decode(req.UserSSHKey); err != nil {
		return nil, errors.NewBadRequest("Error parsing the input, got %q", err.Error())
	}

	return req, nil
}

func createSSHKeyEndpoint(dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req, ok := request.(createSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		return dp.CreateSSHKey(req.Spec.Name, user.ID, req.Spec.PublicKey)
	}
}

type deleteSSHKeyReq struct {
	metaName string
}

func decodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteSSHKeyReq
	var ok bool
	if req.metaName, ok = mux.Vars(r)["meta_name"]; !ok {
		return nil, fmt.Errorf("delte key needs a parameter 'meta_name'")
	}

	return req, nil
}

func deleteSSHKeyEndpoint(dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req, ok := request.(deleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		k, err := dp.SSHKey(user.ID, req.metaName)
		if err != nil {
			return nil, fmt.Errorf("failed to load ssh key: %v", err)
		}
		return nil, dp.DeleteSSHKey(user.ID, k.Name)
	}
}

type listSSHKeyReq struct {
}

func decodeListSSHKeyReq(c context.Context, _ *http.Request) (interface{}, error) {
	var req listSSHKeyReq
	return req, nil
}

func listSSHKeyEndpoint(dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		return dp.SSHKeys(user.ID)
	}
}
