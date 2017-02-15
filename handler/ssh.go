package handler

import (
	"encoding/json"
	"net/http"

	"errors"

	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api/v1"

	"fmt"
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

	// Decode
	if err = json.NewDecoder(r.Body).Decode(req.UserSSHKey); err != nil {
		return nil, err
	}

	return req, nil
}

func createSSHKeyEndpoint(
	kps map[string]provider.KubernetesProvider,
	clientset *extensions.WrappedClientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createSSHKeyReq)
		if !ok {
			return nil, NewBadRequest("Bad parameters")
		}

		c := clientset.SSHKeyTPR(req.user.Name)

		// calculate fingerprint
		pubKey, err := ssh.ParsePublicKey([]byte(req.UserSSHKey.PublicKey))
		if err != nil {
			return nil, NewBadRequest("Bad public key")
		}
		fingerprint := ssh.FingerprintLegacyMD5(pubKey)

		key := &extensions.UserSSHKey{
			Metadata: v1.ObjectMeta{
				Name: fmt.Sprintf("usersshkey-%s-%s", req.user.Name, strings.Trim(fingerprint, ":")),
			},
			PublicKey:   req.UserSSHKey.PublicKey,
			Fingerprint: strings.Trim(fingerprint, ":"),
			Name:        req.UserSSHKey.Name,
		}
		return c.Create(key), nil
	}
}

type deleteSSHKeyReq struct {
	userReq
	fingerprint string
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
	if req.fingerprint, ok = mux.Vars(r)["fingerprint"]; ok {
		return nil, errors.New("delte fingerprint needs a parameter 'fingerprint'")
	}

	return req, nil
}

func deleteSSHKeyEndpoint(
	kps map[string]provider.KubernetesProvider,
	clientset *extensions.WrappedClientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteSSHKeyReq)
		if !ok {
			return nil, NewBadRequest("Bad parameters")
		}

		seed := kps["master_store"]

		return seed.DeleteUserSSHKey(req.user, req.fingerprint), nil
	}
}

type listSSHKeyReq struct {
	userReq
}

func decodeListSSHKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req deleteSSHKeyReq

	// Decode user info
	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}

	req.userReq = ur.(userReq)

	return req, nil
}

func listSSHKeyEndpoint(
	kps map[string]provider.KubernetesProvider,
	clientset *extensions.WrappedClientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listSSHKeyReq)
		if !ok {
			return nil, NewBadRequest("Bad parameters")
		}

		seed := kps["master_store"]

		return seed.ListUserSSHKeys(req.user)
	}
}
