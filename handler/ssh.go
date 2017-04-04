package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/extensions"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api/v1"
)

type createSSHKeyReq struct {
	userReq
	*extensions.UserSSHKey
}

func decodeCreateSSHKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req createSSHKeyReq

	// Decode user info
	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = ur.(userReq)
	req.UserSSHKey = &extensions.UserSSHKey{}

	// Decode
	if err = json.NewDecoder(r.Body).Decode(req.UserSSHKey); err != nil {
		return nil, NewBadRequest("Error parsing the input, got %q", err.Error())
	}

	return req, nil
}

func createSSHKeyEndpoint(
	clientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createSSHKeyReq)
		if !ok {
			return nil, NewBadRequest("Bad parameters")
		}

		c := clientset.SSHKeyTPR(req.user.Name)

		fingerprint, err := extensions.GenerateNormalizedFigerprint(req.UserSSHKey.PublicKey)
		if err != nil {
			return nil, NewBadRequest("Bad public key")
		}

		key := &extensions.UserSSHKey{
			Metadata: v1.ObjectMeta{
				// Metadata Name must match the regex [a-z0-9]([-a-z0-9]*[a-z0-9])? (e.g. 'my-name' or '123-abc')
				Name: extensions.ConstructNewSerialKeyName(extensions.NormalizeFingerprint(fingerprint)),
			},
			PublicKey:   req.UserSSHKey.PublicKey,
			Fingerprint: fingerprint,
			Name:        req.UserSSHKey.Name,
		}
		return c.Create(key)
	}
}

type deleteSSHKeyReq struct {
	userReq
	metaName string
}

func decodeDeleteSSHKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req deleteSSHKeyReq

	// Decode user info
	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = ur.(userReq)

	var ok bool
	if req.metaName, ok = mux.Vars(r)["meta_name"]; !ok {
		return nil, errors.New("delte key needs a parameter 'meta_name'")
	}

	return req, nil
}

func deleteSSHKeyEndpoint(
	clientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteSSHKeyReq)
		if !ok {
			return nil, NewBadRequest("Bad parameters")
		}

		c := clientset.SSHKeyTPR(req.user.Name)

		return nil, c.Delete(req.metaName, v1.NewDeleteOptions(100))
	}
}

type listSSHKeyReq struct {
	userReq
}

func decodeListSSHKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req listSSHKeyReq

	// Decode user info
	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}

	req.userReq = ur.(userReq)

	return req, nil
}

func listSSHKeyEndpoint(
	clientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listSSHKeyReq)
		if !ok {
			return nil, NewBadRequest("Bad parameters, add user credentials")
		}

		c := clientset.SSHKeyTPR(req.user.Name)
		listing, err := c.List()
		if err != nil {
			return nil, err
		}

		return listing.Items, err
	}
}
