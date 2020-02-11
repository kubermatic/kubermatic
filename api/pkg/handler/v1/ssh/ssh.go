package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func CreateEndpoint(keyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(CreateReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingKeys, err := keyProvider.List(project, &provider.SSHKeyListOptions{SSHKeyName: req.Key.Name})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(existingKeys) > 0 {
			return nil, errors.NewAlreadyExists("ssh key", req.Key.Name)
		}

		key, err := keyProvider.Create(userInfo, project, req.Key.Name, req.Key.Spec.PublicKey)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiKey := apiv1.SSHKey{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                key.Name,
				Name:              key.Spec.Name,
				CreationTimestamp: apiv1.NewTime(key.CreationTimestamp.Time),
			},
			Spec: apiv1.SSHKeySpec{
				Fingerprint: key.Spec.Fingerprint,
				PublicKey:   key.Spec.PublicKey,
			},
		}
		return apiKey, nil
	}
}

func DeleteEndpoint(keyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(DeleteReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		err = keyProvider.Delete(userInfo, req.SSHKeyID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func ListEndpoint(keyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(ListReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {

			return nil, errors.NewBadRequest("the name of the project to delete cannot be empty")
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		keys, err := keyProvider.List(project, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiKeys := common.ConvertInternalSSHKeysToExternal(keys)
		return apiKeys, nil
	}
}

// ListReq defined HTTP request for listSHHKeys endpoint
// swagger:parameters listSSHKeys
type ListReq struct {
	common.ProjectReq
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	req, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, nil
	}
	return ListReq{ProjectReq: req.(common.ProjectReq)}, err
}

// DeleteReq defines HTTP request for deleteSSHKey endpoint
// swagger:parameters deleteSSHKey
type DeleteReq struct {
	common.ProjectReq
	// in: path
	SSHKeyID string `json:"key_id"`
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteReq

	dcr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = dcr.(common.ProjectReq)
	SSHKeyID, ok := mux.Vars(r)["key_id"]
	if !ok {
		return nil, fmt.Errorf("'key_id' parameter is required in order to delete ssh key")
	}

	req.SSHKeyID = SSHKeyID
	return req, nil
}

// CreateReq represent a request for specific data to create a new SSH key
// swagger:parameters createSSHKey
type CreateReq struct {
	common.ProjectReq
	// in: body
	Key apiv1.SSHKey
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateReq

	dcr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = dcr.(common.ProjectReq)

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
