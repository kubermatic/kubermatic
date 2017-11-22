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

type CreateSSHKeyReq struct {
	*v1.UserSSHKey
}

func decodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateSSHKeyReq
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
		req, ok := request.(CreateSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		return dp.CreateSSHKey(req.Spec.Name, req.Spec.PublicKey, user)
	}
}

type DeleteSSHKeyReq struct {
	MetaName string
}

func decodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteSSHKeyReq
	var ok bool
	if req.MetaName, ok = mux.Vars(r)["meta_name"]; !ok {
		return nil, fmt.Errorf("delte key needs a parameter 'meta_name'")
	}

	return req, nil
}

func deleteSSHKeyEndpoint(dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req, ok := request.(DeleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		k, err := dp.SSHKey(user, req.MetaName)
		if err != nil {
			return nil, fmt.Errorf("failed to load ssh key: %v", err)
		}
		return nil, dp.DeleteSSHKey(k.Name, user)
	}
}

type ListSSHKeyReq struct {
}

func decodeListSSHKeyReq(c context.Context, _ *http.Request) (interface{}, error) {
	var req ListSSHKeyReq
	return req, nil
}

func listSSHKeyEndpoint(dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		return dp.SSHKeys(user)
	}
}
