package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func createSSHKeyEndpoint(keyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(CreateSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		key, err := keyProvider.Create(userInfo, project, req.Key.Name, req.Key.Spec.PublicKey)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		apiKey := apiv1.SSHKey{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                key.Name,
				Name:              key.Spec.Name,
				CreationTimestamp: key.CreationTimestamp.Time,
			},
			Spec: apiv1.SSHKeySpec{
				Fingerprint: key.Spec.Fingerprint,
				PublicKey:   key.Spec.PublicKey,
			},
		}
		return apiKey, nil
	}
}

func deleteSSHKeyEndpoint(keyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(DeleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		err = keyProvider.Delete(userInfo, req.SSHKeyID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func listSSHKeyEndpoint(keyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(ListSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {

			return nil, errors.NewBadRequest("the name of the project to delete cannot be empty")
		}

		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		keys, err := keyProvider.List(project, nil)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		apiKeys := convertInternalSSHKeysToExternal(keys)
		return apiKeys, nil
	}
}

func convertInternalSSHKeysToExternal(internalKeys []*kubermaticapiv1.UserSSHKey) []*apiv1.SSHKey {
	apiKeys := make([]*apiv1.SSHKey, len(internalKeys))
	for index, key := range internalKeys {
		apiKey := &apiv1.SSHKey{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                key.Name,
				Name:              key.Spec.Name,
				CreationTimestamp: key.CreationTimestamp.Time,
			},
			Spec: apiv1.SSHKeySpec{
				Fingerprint: key.Spec.Fingerprint,
				PublicKey:   key.Spec.PublicKey,
			},
		}
		apiKeys[index] = apiKey
	}
	return apiKeys
}

// ListSSHKeyReq defined HTTP request for listSHHKeys endpoint
// swagger:parameters listSSHKeys
type ListSSHKeyReq struct {
	ProjectReq
}

func decodeListSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	req, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, nil
	}
	return ListSSHKeyReq{ProjectReq: req.(ProjectReq)}, err
}

// DeleteSSHKeyReq defines HTTP request for deleteSSHKey endpoint
// swagger:parameters deleteSSHKey
type DeleteSSHKeyReq struct {
	ProjectReq
	// in: path
	SSHKeyID string `json:"key_id"`
}

func decodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteSSHKeyReq

	dcr, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = dcr.(ProjectReq)
	SSHKeyID, ok := mux.Vars(r)["key_id"]
	if !ok {
		return nil, fmt.Errorf("'key_id' parameter is required in order to delete ssh key")
	}

	req.SSHKeyID = SSHKeyID
	return req, nil
}

// CreateSSHKeyReq represent a request for specific data to create a new SSH key
// swagger:parameters createSSHKey
type CreateSSHKeyReq struct {
	ProjectReq
	// swagger:ignore
	Key apiv1.SSHKey `json:"-"`
}

func decodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateSSHKeyReq

	dcr, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = dcr.(ProjectReq)

	req.Key = apiv1.SSHKey{}
	if err := json.NewDecoder(r.Body).Decode(&req.Key); err != nil {
		return nil, errors.NewBadRequest("unable to parse the input, err = %v", err.Error())
	}

	if len(req.Key.Name) == 0 {
		return nil, fmt.Errorf("'name' field cannot be empty")
	}
	if len(req.Key.Spec.PublicKey) == 0 {
		return nil, fmt.Errorf("'spec.publicKey' field cannot be empty")
	}

	return req, nil
}
