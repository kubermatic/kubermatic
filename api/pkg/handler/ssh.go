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
		req, ok := request.(newCreateSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		project, err := projectProvider.Get(user, req.projectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		key, err := keyProvider.Create(user, project, req.Metadata.DisplayName, req.Spec.PublicKey)
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
		req, ok := request.(newDeleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		project, err := projectProvider.Get(user, req.projectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		err = keyProvider.Delete(user, project, req.sshKeyName)
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
		projectName, ok := request.(string)
		if !ok {
			return nil, errors.NewBadRequest("the name of the project to delete cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		project, err := projectProvider.Get(user, projectName)
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

func newDecodeListSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	return decodeProjectPathReq(c, r)
}

type newDeleteSSHKeyReq struct {
	projectName string
	sshKeyName  string
}

func newDecodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req newDeleteSSHKeyReq

	// project_id is actually an internal name of the object
	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, fmt.Errorf("'project_id' parameter is required in order to delete ssh key that belong to the project")
	}
	req.projectName = projectName

	sshKeyName, ok := mux.Vars(r)["key_name"]
	if !ok {
		return nil, fmt.Errorf("'key_name' parameter is required in order to delete ssh key")
	}

	req.projectName = projectName
	req.sshKeyName = sshKeyName
	return req, nil
}

// newCreateSSHKeyReq represent a request for specific data to create a new SSH key
type newCreateSSHKeyReq struct {
	apiv2.NewSSHKey
	projectName string
}

func newDecodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req newCreateSSHKeyReq

	// project_id is actually an internal name of the object
	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, fmt.Errorf("'project_id' parameter is required in order to list ssh Keys that belong to the project")
	}
	req.projectName = projectName

	req.NewSSHKey = apiv2.NewSSHKey{}
	if err := json.NewDecoder(r.Body).Decode(&req.NewSSHKey); err != nil {
		return nil, errors.NewBadRequest("unable to parse the input, err = %v", err.Error())
	}

	if len(req.Metadata.Name) != 0 {
		return nil, fmt.Errorf("'metadata.name' field cannot be set, please set 'metadata.displayName' instead")
	}
	if len(req.Spec.PublicKey) == 0 {
		return nil, fmt.Errorf("'spec.publicKey' field cannot be empty")
	}
	if len(req.Metadata.DisplayName) == 0 {
		return nil, fmt.Errorf("'metadata.displayName' field cannot be empty")
	}

	return req, nil
}
