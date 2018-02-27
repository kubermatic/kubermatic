package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// ClustersReq represent a request for clusters specific data
// swagger:parameters listClusters
type ClustersReq struct {
	DCReq
}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClustersReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// ClusterReq represent a request for cluster specific data
// swagger:parameters getCluster deleteCluster getClusterKubeconfig getClusterNodes getClusterNodesV2
type ClusterReq struct {
	DCReq
	// in: path
	Cluster string `json:"cluster"`
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq
	req.Cluster = mux.Vars(r)["cluster"]

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// NewClusterReq represent a request for clusters specific data
// swagger:parameters createCluster
type NewClusterReq struct {
	DCReq
	// in: body
	Body NewClusterReqBody
}

// NewClusterReqBody represents the body of a new cluster request
type NewClusterReqBody struct {
	Cluster *kubermaticv1.ClusterSpec `json:"cluster"`
	SSHKeys []string                  `json:"sshKeys"`
}

func decodeNewClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewClusterReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// DCsReq represent a request for datacenters specific data
type DCsReq struct{}

func decodeDatacentersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCsReq

	return req, nil
}

type DCGetter interface {
	GetDC() string
}

// DCReq represent a request for datacenter specific data
// swagger:parameters getDatacenter
type DCReq struct {
	// in: path
	DC string `json:"dc"`
}

func (req DCReq) GetDC() string {
	return req.DC
}

func decodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCReq

	req.DC = mux.Vars(r)["dc"]
	return req, nil
}

// DoSizesReq represent a request for digitalocean sizes
type DoSizesReq struct {
	DoToken string
}

func decodeDoSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DoSizesReq

	req.DoToken = r.Header.Get("DoToken")
	return req, nil
}

func decodeKubeconfigReq(c context.Context, r *http.Request) (interface{}, error) {
	req, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// CreateNodesReq represent a request for specific data to create a node
// swagger:parameters createClusterNodes
type CreateNodesReq struct {
	ClusterReq
	// in: body
	Body CreateNodesReqBody
}

// CreateNodesReqBody represents the request body of a create nodes request
type CreateNodesReqBody struct {
	Instances int            `json:"instances"`
	Spec      apiv1.NodeSpec `json:"spec"`
}

func decodeCreateNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateNodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// NodeReq represent a request for node specific data
// swagger:parameters getClusterNode deleteClusterNode
type NodeReq struct {
	ClusterReq
	// in: path
	NodeName string `json:"node"`
}

func decodeNodeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)
	req.NodeName = mux.Vars(r)["node"]

	return req, nil
}

// CreateSSHKeyReq represent a request for specific data to create a new SSH key
type CreateSSHKeyReq struct {
	apiv1.SSHKey
}

func decodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateSSHKeyReq
	req.SSHKey = apiv1.SSHKey{}

	if err := json.NewDecoder(r.Body).Decode(&req.SSHKey); err != nil {
		return nil, errors.NewBadRequest("Error parsing the input, got %q", err.Error())
	}

	return req, nil
}

// DeleteSSHKeyReq represent a request for deleting a SSH key
// swagger:parameters deleteSSHKey
type DeleteSSHKeyReq struct {
	// in: path
	MetaName string `json:"meta_name"`
}

func decodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteSSHKeyReq
	var ok bool
	if req.MetaName, ok = mux.Vars(r)["meta_name"]; !ok {
		return nil, fmt.Errorf("delte key needs a parameter 'meta_name'")
	}

	return req, nil
}

// ListSSHKeyReq represent a request for listing all user SSH keys
type ListSSHKeyReq struct{}

func decodeListSSHKeyReq(c context.Context, _ *http.Request) (interface{}, error) {
	var req ListSSHKeyReq
	return req, nil
}

// UpgradeReq represent a request for cluster upgrade specific data
type UpgradeReq struct {
	ClusterReq
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

func decodeEmptyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req struct{}
	return req, nil
}
