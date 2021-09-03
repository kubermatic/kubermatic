/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mlaadminsetting

import (
	"context"
	"encoding/json"
	"net/http"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
)

// getReq defines HTTP request for getting MLAAdminSetting
// swagger:parameters getMLAAdminSetting
type getReq struct {
	cluster.GetClusterReq
}

// createReq defines HTTP request for creating MLAAdminSetting
// swagger:parameters createMLAAdminSetting
type createReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body apiv2.MLAAdminSetting
}

// updateReq defines HTTP request for updating MLAAdminSetting
// swagger:parameters updateMLAAdminSetting
type updateReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body apiv2.MLAAdminSetting
}

// deleteReq defines HTTP request for deleting MLAAdminSetting
// swagger:parameters deleteMLAAdminSetting
type deleteReq struct {
	cluster.GetClusterReq
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}
