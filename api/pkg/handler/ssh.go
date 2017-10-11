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

func createSSHKeyEndpoint(c crdclient.Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := GetUser(ctx)
		if err != nil {
			return nil, err
		}
		req, ok := request.(createSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		key, err := v1.NewUserSSHKeyBuilder().
			SetName(req.Spec.Name).
			SetOwner(user.Name).
			SetRawKey(req.Spec.PublicKey).
			Build()
		if err != nil {
			return nil, err
		}
		return c.KubermaticV1().UserSSHKeies().Create(key)
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

func deleteSSHKeyEndpoint(c crdclient.Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := GetUser(ctx)
		if err != nil {
			return nil, err
		}
		req, ok := request.(deleteSSHKeyReq)
		if !ok {
			return nil, errors.NewBadRequest("Bad parameters")
		}

		key, err := c.KubermaticV1().UserSSHKeies().Get(req.metaName, metav1.GetOptions{})
		if err != nil {
			glog.V(5).Info(err)
			return nil, fmt.Errorf("can't access key %q", req.metaName)
		}
		if key.Spec.Owner != user.Name {
			err = fmt.Errorf("user %q is not permitted to delete the key %q", user.Name, req.metaName)
			glog.Warning(err)
			return nil, err
		}
		return nil, c.KubermaticV1().UserSSHKeies().Delete(req.metaName, metav1.NewDeleteOptions(100))
	}
}

type listSSHKeyReq struct {
}

func decodeListSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listSSHKeyReq
	return req, nil
}

func listSSHKeyEndpoint(c crdclient.Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := GetUser(ctx)
		if err != nil {
			return nil, err
		}

		opts, err := ssh.UserListOptions(user.Name)
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
