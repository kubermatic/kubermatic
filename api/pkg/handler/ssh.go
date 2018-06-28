package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/api/v2"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func createSSHKeyEndpoint(dp provider.SSHKeyProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)

		req, ok := request.(CreateSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		return dp.CreateSSHKey(req.Spec.Name, req.Spec.PublicKey, user)
	}
}

func newCreateSSHKeyEndpoint(keyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NewCreateSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		key, err := keyProvider.Create(user, project, req.Key.Metadata.DisplayName, req.Key.Spec.PublicKey)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		apiKey := v2.NewSSHKey{
			Metadata: v2.ObjectMeta{
				Name:              key.Name,
				CreationTimestamp: key.CreationTimestamp.Time,
			},
			Spec: v2.NewSSHKeySpec{
				Fingerprint: key.Spec.Fingerprint,
				PublicKey:   key.Spec.PublicKey,
			},
		}
		return apiKey, nil
	}
}

func newDeleteSSHKeyEndpoint(keyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NewDeleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		err = keyProvider.Delete(user, project, req.SSHKeyName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func deleteSSHKeyEndpoint(dp provider.SSHKeyProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
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

func listSSHKeyEndpoint(dp provider.SSHKeyProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		return dp.SSHKeys(user)
	}
}

func newListSSHKeyEndpoint(keyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NewListSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectName) == 0 {

			return nil, errors.NewBadRequest("the name of the project to delete cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		keys, err := keyProvider.List(user, project, nil)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		apiKeys := convertInternalSSHKeysToExternal(keys)
		return apiKeys, nil
	}
}

func convertInternalSSHKeysToExternal(internalKeys []*kubermaticapiv1.UserSSHKey) []v2.NewSSHKey {
	apiKeys := make([]v2.NewSSHKey, len(internalKeys))
	for index, key := range internalKeys {
		apiKey := v2.NewSSHKey{
			Metadata: v2.ObjectMeta{
				Name:              key.Name,
				CreationTimestamp: key.CreationTimestamp.Time,
			},
			Spec: v2.NewSSHKeySpec{
				Fingerprint: key.Spec.Fingerprint,
				PublicKey:   key.Spec.PublicKey,
			},
		}
		apiKeys[index] = apiKey
	}
	return apiKeys
}

// NewListSSHKeyReq defined HTTP request for newListSHHKeys endpoint
// swagger:parameters newListSSHKeys
type NewListSSHKeyReq struct {
	// in: path
	ProjectName string `json:"project_id"`
}

func newDecodeListSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewListSSHKeyReq
	var err error
	req.ProjectName, err = decodeProjectPathReq(c, r)
	return req, err
}

// NewDeleteSSHKeyReq defines HTTP request for newDeleteSSHKey endpoint
// swagger:parameters newDeleteSSHKey
type NewDeleteSSHKeyReq struct {
	// in: path
	ProjectName string `json:"project_id"`
	// in: path
	SSHKeyName string `json:"key_name"`
}

func newDecodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewDeleteSSHKeyReq

	// project_id is actually an internal name of the object
	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, fmt.Errorf("'project_id' parameter is required in order to delete ssh key that belong to the project")
	}
	req.ProjectName = projectName

	sshKeyName, ok := mux.Vars(r)["key_name"]
	if !ok {
		return nil, fmt.Errorf("'key_name' parameter is required in order to delete ssh key")
	}

	req.ProjectName = projectName
	req.SSHKeyName = sshKeyName
	return req, nil
}

// NewCreateSSHKeyReq represent a request for specific data to create a new SSH key
// swagger:parameters newCreateSSHKey
type NewCreateSSHKeyReq struct {
	// swagger:ignore
	Key apiv2.NewSSHKey `json:"-"`
	// in: path
	ProjectName string `json:"project_id"`
}

func newDecodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewCreateSSHKeyReq

	// project_id is actually an internal name of the object
	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, fmt.Errorf("'project_id' parameter is required in order to list ssh Keys that belong to the project")
	}
	req.ProjectName = projectName

	req.Key = apiv2.NewSSHKey{}
	if err := json.NewDecoder(r.Body).Decode(&req.Key); err != nil {
		return nil, errors.NewBadRequest("unable to parse the input, err = %v", err.Error())
	}

	if len(req.Key.Metadata.Name) != 0 {
		return nil, fmt.Errorf("'metadata.name' field cannot be set, please set 'metadata.displayName' instead")
	}
	if len(req.Key.Spec.PublicKey) == 0 {
		return nil, fmt.Errorf("'spec.publicKey' field cannot be empty")
	}
	if len(req.Key.Metadata.DisplayName) == 0 {
		return nil, fmt.Errorf("'metadata.displayName' field cannot be empty")
	}

	return req, nil
}
