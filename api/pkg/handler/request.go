package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	"github.com/gorilla/mux"
)

type ClustersReq struct{}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	return ClustersReq{}, nil
}

// swagger:parameters performClusterUpgrage
type ClusterReq struct {
	// in: path
	Cluster string
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq
	req.Cluster = mux.Vars(r)["cluster"]
	return req, nil
}

type NewClusterReqV2 struct {
	Cluster *kubermaticv1.ClusterSpec `json:"cluster"`
	SSHKeys []string                  `json:"sshKeys"`
}

func decodeNewClusterReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req NewClusterReqV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type DCsReq struct {
}

func decodeDatacentersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCsReq

	return req, nil
}

type DCReq struct {
	DC string
}

func decodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCReq

	req.DC = mux.Vars(r)["dc"]
	return req, nil
}

// swagger:parameters kubeconfigHandler
type KubeconfigReq struct {
	ClusterReq
}

func decodeKubeconfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubeconfigReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	return req, nil
}

type NodesReq struct {
	ClusterReq
}

func decodeNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	return req, nil
}

type CreateNodesReq struct {
	ClusterReq
	Instances int          `json:"instances"`
	Spec      api.NodeSpec `json:"spec"`
}

func decodeCreateNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateNodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type NodeReq struct {
	NodesReq
	NodeName string
}

func decodeNodeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeReq

	cr, err := decodeNodesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.NodesReq = cr.(NodesReq)
	req.NodeName = mux.Vars(r)["node"]

	return req, nil
}

type CreateSSHKeyReq struct {
	*kubermaticv1.UserSSHKey
}

func decodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateSSHKeyReq
	req.UserSSHKey = &kubermaticv1.UserSSHKey{}
	// Decode
	if err := json.NewDecoder(r.Body).Decode(req.UserSSHKey); err != nil {
		return nil, errors.NewBadRequest("Error parsing the input, got %q", err.Error())
	}

	return req, nil
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

type ListSSHKeyReq struct {
}

func decodeListSSHKeyReq(c context.Context, _ *http.Request) (interface{}, error) {
	var req ListSSHKeyReq
	return req, nil
}

// swagger:parameters performClusterUpgrage
type UpgradeReq struct {
	// UpgradeReq contains parameter for an update request
	//
	ClusterReq
	// in: body
	To string
}

func decodeUpgradeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpgradeReq

	dr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = dr.(ClusterReq)

	defer func() {
		if err := r.Body.Close(); err != nil {
			panic(err)
		}
	}()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	v := new(struct {
		To string
	})

	err = json.Unmarshal(b, v)
	if err != nil {
		return nil, err
	}

	req.To = v.To

	return req, nil
}
