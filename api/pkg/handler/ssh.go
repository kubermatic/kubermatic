package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

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

func listSSHKeyEndpoint(dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		return dp.SSHKeys(user)
	}
}
