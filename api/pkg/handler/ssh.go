package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/errors"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createSSHKeyReq struct {
	userReq
	*v1.UserSSHKey
}

func decodeCreateSSHKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req createSSHKeyReq

	// Decode user info
	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = ur.(userReq)
	req.UserSSHKey = &v1.UserSSHKey{}

	// Decode
	if err = json.NewDecoder(r.Body).Decode(req.UserSSHKey); err != nil {
		return nil, errors.NewBadRequest("Error parsing the input, got %q", err.Error())
	}

	return req, nil
}

func createSSHKeyEndpoint(c crdclient.Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		key, err := v1.NewUserSSHKeyBuilder().
			SetName(req.Spec.Name).
			SetOwner(req.user.Name).
			SetRawKey(req.Spec.PublicKey).
			Build()
		if err != nil {
			return nil, err
		}
		return c.KubermaticV1().UserSSHKeies().Create(key)
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
		return nil, fmt.Errorf("delte key needs a parameter 'meta_name'")
	}

	return req, nil
}

func deleteSSHKeyEndpoint(c crdclient.Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		key, err := c.KubermaticV1().UserSSHKeies().Get(req.metaName, metav1.GetOptions{})
		if err != nil {
			glog.V(5).Info(err)
			return nil, fmt.Errorf("can't access key %q", req.metaName)
		}
		if key.Spec.Owner != req.user.Name {
			err = fmt.Errorf("user %q is not permitted to delete the key %q", req.user.Name, req.metaName)
			glog.Warning(err)
			return nil, err
		}
		return nil, c.KubermaticV1().UserSSHKeies().Delete(req.metaName, metav1.NewDeleteOptions(100))
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

func listSSHKeyEndpoint(c crdclient.Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters, add user credentials")
		}

		opts, err := ssh.UserListOptions(req.user.Name)
		if err != nil {
			return nil, err
		}
		glog.V(7).Infof("searching for users SSH keys with label selector: (%s)", opts.LabelSelector)
		listing, err := c.KubermaticV1().UserSSHKeies().List(opts)
		if err != nil {
			return nil, err
		}

		return listing.Items, err
	}
}
